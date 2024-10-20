package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/ghupload"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateToko(respw http.ResponseWriter, req *http.Request) {
	// Validasi token
	payload, err := watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Token Tidak Valid"
		respn.Info = at.GetSecretFromHeader(req)
		respn.Location = "Decode Token Error"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Parse multipart form data (untuk menerima file upload)
	err = req.ParseMultipartForm(10 << 20) // Batas 10MB
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal memproses form data"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Ambil file gambar dari form data
	file, header, err := req.FormFile("tokoImage")
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gambar toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}
	defer file.Close()

	// Baca isi file gambar
	fileContent, err := io.ReadAll(file)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal membaca file"
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Buat nama file unik dengan hashing
	hashedFileName := ghupload.CalculateHash(fileContent) + header.Filename[strings.LastIndex(header.Filename, "."):] // Tambahkan ekstensi asli file

	// Upload file ke GitHub
	GitHubAccessToken := config.GHAccessToken
	GitHubAuthorName := "Rolly Maulana Awangga"
	GitHubAuthorEmail := "awangga@gmail.com"
	githubOrg := "gobizid"
	githubRepo := "img"
	pathFile := "tokoImages/" + hashedFileName
	replace := true

	// Upload gambar ke GitHub menggunakan fungsi yang sudah ada
	content, _, err := ghupload.GithubUpload(GitHubAccessToken, GitHubAuthorName, GitHubAuthorEmail, fileContent, githubOrg, githubRepo, pathFile, replace)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupload gambar ke GitHub"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// URL gambar yang diupload ke GitHub
	gambarTokoURL := *content.Content.HTMLURL

	// Decode body request menjadi struct Toko (untuk data toko)
	var tokoInput model.Toko
	err = json.NewDecoder(req.Body).Decode(&tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Body tidak valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cek apakah user sudah memiliki toko
	docTokoUser, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", primitive.M{"user.phonenumber": payload.Id})
	if err == nil && docTokoUser.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: User sudah memiliki toko"
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Cek apakah nama toko sudah ada
	docTokoNama, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", primitive.M{"nama_toko": tokoInput.NamaToko})
	if err == nil && docTokoNama.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Nama Toko sudah digunakan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cek apakah slug toko sudah ada
	docTokoSlug, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", primitive.M{"slug": tokoInput.Slug})
	if err == nil && docTokoSlug.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Slug Toko sudah digunakan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Set user yang membuat toko
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}
	tokoInput.User = []model.Userdomyikado{docuser}

	// Simpan URL gambar ke struct toko
	tokoInput.GambarToko = gambarTokoURL

	// Jika data menu tidak ada (kosong atau null), inisialisasi sebagai array kosong
	if tokoInput.Menu == nil {
		tokoInput.Menu = []model.Menu{}
	}

	// Insert data toko ke database
	dataToko, err := atdb.InsertOneDoc(config.Mongoconn, "menu", tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Gagal Insert Database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}

	// Buat response sukses dengan data yang lebih lengkap
	response := map[string]interface{}{
		"status":  "success",
		"message": "Toko berhasil dibuat",
		"data": map[string]interface{}{
			"id":         dataToko,
			"nama_toko":  tokoInput.NamaToko,
			"slug":       tokoInput.Slug,
			"categories": tokoInput.Category,
			"alamat": map[string]interface{}{
				"street":      tokoInput.Alamat.Street,
				"province":    tokoInput.Alamat.Province,
				"city":        tokoInput.Alamat.City,
				"description": tokoInput.Alamat.Description,
				"postal_code": tokoInput.Alamat.PostalCode,
				"createdAt":   tokoInput.Alamat.CreatedAt,
				"updatedAt":   tokoInput.Alamat.UpdatedAt,
			},
			"gambar_toko": gambarTokoURL,  // URL gambar toko yang di-upload
			"user":        tokoInput.User, // informasi user
			"menu":        tokoInput.Menu, // informasi menu
		},
	}

	// Response sukses dengan data lengkap
	at.WriteJSON(respw, http.StatusOK, response)
}

func UpdateToko(respw http.ResponseWriter, req *http.Request) {
	// Validasi token
	payload, err := watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Token Tidak Valid"
		respn.Info = at.GetSecretFromHeader(req)
		respn.Location = "Decode Token Error"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Ambil ID toko dari parameter URL
	vars := mux.Vars(req) // Menggunakan Gorilla Mux
	tokoID := vars["id"]
	if tokoID == "" {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Decode body request menjadi struct Toko
	var tokoInput model.Toko
	err = json.NewDecoder(req.Body).Decode(&tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Body tidak valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cek apakah toko dengan ID yang diberikan ada
	objectID, err := primitive.ObjectIDFromHex(tokoID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	var existingToko model.Toko
	filter := bson.M{"_id": objectID}
	err = config.Mongoconn.Collection("menu").FindOne(context.TODO(), filter).Decode(&existingToko)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Cek apakah user yang ingin mengupdate toko adalah pemilik toko
	if existingToko.User[0].PhoneNumber != payload.Id {
		var respn model.Response
		respn.Status = "Error: User tidak memiliki hak akses untuk mengupdate toko ini"
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Cek apakah nama toko sudah digunakan oleh toko lain
	if tokoInput.NamaToko != "" && tokoInput.NamaToko != existingToko.NamaToko {
		var tokoNama model.Toko
		err := config.Mongoconn.Collection("menu").FindOne(context.TODO(), bson.M{"nama_toko": tokoInput.NamaToko}).Decode(&tokoNama)
		if err == nil {
			var respn model.Response
			respn.Status = "Error: Nama Toko sudah digunakan"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}
	}

	// Cek apakah slug toko sudah digunakan oleh toko lain
	if tokoInput.Slug != "" && tokoInput.Slug != existingToko.Slug {
		var tokoSlug model.Toko
		err := config.Mongoconn.Collection("menu").FindOne(context.TODO(), bson.M{"slug": tokoInput.Slug}).Decode(&tokoSlug)
		if err == nil {
			var respn model.Response
			respn.Status = "Error: Slug Toko sudah digunakan"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}
	}

	// Update data toko
	updateData := bson.M{}
	if tokoInput.NamaToko != "" {
		updateData["nama_toko"] = tokoInput.NamaToko
	}
	if tokoInput.Slug != "" {
		updateData["slug"] = tokoInput.Slug
	}
	if tokoInput.Category != "" {
		updateData["category"] = tokoInput.Category
	}
	if tokoInput.Alamat.Street != "" {
		updateData["alamat.street"] = tokoInput.Alamat.Street
	}
	if tokoInput.Alamat.Province != "" {
		updateData["alamat.province"] = tokoInput.Alamat.Province
	}
	if tokoInput.Alamat.City != "" {
		updateData["alamat.city"] = tokoInput.Alamat.City
	}
	if tokoInput.Alamat.Description != "" {
		updateData["alamat.description"] = tokoInput.Alamat.Description
	}
	if tokoInput.Alamat.PostalCode != "" {
		updateData["alamat.postal_code"] = tokoInput.Alamat.PostalCode
	}

	// Jika menu ingin diperbarui
	if tokoInput.Menu != nil {
		updateData["menu"] = tokoInput.Menu
	}

	// Lakukan update ke database
	update := bson.M{"$set": updateData}
	_, err = config.Mongoconn.Collection("menu").UpdateOne(context.TODO(), filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate toko"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}

	// Buat response sukses
	response := map[string]interface{}{
		"status":  "success",
		"message": "Toko berhasil diupdate",
		"data":    tokoInput,
	}

	// Response sukses
	at.WriteJSON(respw, http.StatusOK, response)
}
