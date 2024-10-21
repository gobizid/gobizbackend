package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/ghupload"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateToko(respw http.ResponseWriter, req *http.Request) {
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Token Tidak Valid"
		respn.Info = at.GetSecretFromHeader(req)
		respn.Location = "Decode Token Error"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	err = req.ParseMultipartForm(10 << 20)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal memproses form data"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	file, header, err := req.FormFile("tokoImage")
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gambar toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}
	defer file.Close()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal membaca file"
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	hashedFileName := ghupload.CalculateHash(fileContent) + header.Filename[strings.LastIndex(header.Filename, "."):] // Tambahkan ekstensi asli file

	GitHubAccessToken := config.GHAccessToken
	GitHubAuthorName := "Rolly Maulana Awangga"
	GitHubAuthorEmail := "awangga@gmail.com"
	githubOrg := "gobizid"
	githubRepo := "img"
	pathFile := "tokoImages/" + hashedFileName
	replace := true

	content, _, err := ghupload.GithubUpload(GitHubAccessToken, GitHubAuthorName, GitHubAuthorEmail, fileContent, githubOrg, githubRepo, pathFile, replace)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupload gambar ke GitHub"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	gambarTokoURL := *content.Content.HTMLURL

	namaToko := req.FormValue("nama_toko")
	slug := req.FormValue("slug")
	category := req.FormValue("category")
	street := req.FormValue("alamat.street")
	province := req.FormValue("alamat.province")
	city := req.FormValue("alamat.city")
	description := req.FormValue("alamat.description")
	postalCode := req.FormValue("alamat.postal_code")

	tokoInput := model.Toko{
		NamaToko:   namaToko,
		Slug:       slug,
		Category:   category,
		GambarToko: gambarTokoURL,
		Alamat: model.Address{
			Street:      street,
			Province:    province,
			City:        city,
			Description: description,
			PostalCode:  postalCode,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	docTokoUser, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", primitive.M{"user.phonenumber": payload.Id})
	if err == nil && docTokoUser.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: User sudah memiliki toko"
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	docTokoNama, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", primitive.M{"nama_toko": tokoInput.NamaToko})
	if err == nil && docTokoNama.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Nama Toko sudah digunakan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	docTokoSlug, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", primitive.M{"slug": tokoInput.Slug})
	if err == nil && docTokoSlug.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Slug Toko sudah digunakan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}
	tokoInput.User = []model.Userdomyikado{docuser}

	if tokoInput.Menu == nil {
		tokoInput.Menu = []model.Menu{}
	}

	dataToko, err := atdb.InsertOneDoc(config.Mongoconn, "menu", tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Gagal Insert Database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}

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
			"gambar_toko": gambarTokoURL,
			"user":        tokoInput.User,
			"menu":        tokoInput.Menu,
		},
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func GetAllMarket(respw http.ResponseWriter, req *http.Request) {
	// Mengambil semua data toko dari collection 'menu'
	tokos, err := atdb.GetAllDoc[[]model.Toko](config.Mongoconn, "menu", primitive.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data market tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Jika tidak ada data toko
	if len(tokos) == 0 {
		var respn model.Response
		respn.Status = "Error: Data market kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Menyimpan hasil semua market (toko)
	var allMarkets []map[string]interface{}

	for _, toko := range tokos {
		// Menambahkan setiap toko ke dalam hasil
		allMarkets = append(allMarkets, map[string]interface{}{
			"id":           toko.ID.Hex(),
			"nama_toko":    toko.NamaToko,
			"slug":         toko.Slug,
			"category":     toko.Category,
			"gambar_toko":  toko.GambarToko,
			"alamat": map[string]interface{}{
				"street":      toko.Alamat.Street,
				"province":    toko.Alamat.Province,
				"city":        toko.Alamat.City,
				"description": toko.Alamat.Description,
				"postal_code": toko.Alamat.PostalCode,
			},
			"user": toko.User, // Tambahkan informasi user jika diperlukan
		})
	}

	// Mengembalikan data market dalam format JSON
	at.WriteJSON(respw, http.StatusOK, allMarkets)
}

func UpdateToko(respw http.ResponseWriter, req *http.Request) {
    // Ambil token dari header
    payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
    if err != nil {
        var respn model.Response
        respn.Status = "Error: Token Tidak Valid"
        respn.Info = at.GetSecretFromHeader(req)
        respn.Location = "Decode Token Error"
        respn.Response = err.Error()
        at.WriteJSON(respw, http.StatusForbidden, respn)
        return
    }

    // Ambil ID toko dari query parameter
    tokoID := req.URL.Query().Get("id")
    if tokoID == "" {
        var respn model.Response
        respn.Status = "Error: ID Toko tidak ditemukan"
        at.WriteJSON(respw, http.StatusBadRequest, respn)
        return
    }

    // Coba konversi tokoID dari string ke ObjectID MongoDB
    objectID, err := primitive.ObjectIDFromHex(tokoID)
    if err != nil {
        var respn model.Response
        respn.Status = "Error: ID Toko tidak valid"
        at.WriteJSON(respw, http.StatusBadRequest, respn)
        return
    }

    // Coba ambil data toko dari database berdasarkan ID
    var existingToko model.Toko
    filter := bson.M{"_id": objectID}
    err = config.Mongoconn.Collection("menu").FindOne(context.TODO(), filter).Decode(&existingToko)
    if err != nil {
        var respn model.Response
        respn.Status = "Error: Toko tidak ditemukan"
        at.WriteJSON(respw, http.StatusNotFound, respn)
        return
    }

    // Cek apakah user yang melakukan update adalah pemilik toko
    if existingToko.User[0].PhoneNumber != payload.Id {
        var respn model.Response
        respn.Status = "Error: User tidak memiliki hak akses untuk mengupdate toko ini"
        at.WriteJSON(respw, http.StatusForbidden, respn)
        return
    }

    // Parsing form data (dengan batasan 10MB)
    err = req.ParseMultipartForm(10 << 20)
    if err != nil {
        var respn model.Response
        respn.Status = "Error: Gagal memproses form data"
        respn.Response = err.Error()
        at.WriteJSON(respw, http.StatusBadRequest, respn)
        return
    }

    // Handle file upload untuk gambar toko (opsional)
    var gambarTokoURL string
    file, header, err := req.FormFile("tokoImage")
    if err == nil {
        defer file.Close()

        fileContent, err := io.ReadAll(file)
        if err != nil {
            var respn model.Response
            respn.Status = "Error: Gagal membaca file"
            at.WriteJSON(respw, http.StatusInternalServerError, respn)
            return
        }

        // Generate nama file yang aman dengan hashing
        hashedFileName := ghupload.CalculateHash(fileContent) + header.Filename[strings.LastIndex(header.Filename, "."):]

        // Upload gambar ke GitHub
        GitHubAccessToken := config.GHAccessToken
        GitHubAuthorName := "Rolly Maulana Awangga"
        GitHubAuthorEmail := "awangga@gmail.com"
        githubOrg := "gobizid"
        githubRepo := "img"
        pathFile := "tokoImages/" + hashedFileName
        replace := true

        content, _, err := ghupload.GithubUpload(GitHubAccessToken, GitHubAuthorName, GitHubAuthorEmail, fileContent, githubOrg, githubRepo, pathFile, replace)
        if err != nil {
            var respn model.Response
            respn.Status = "Error: Gagal mengupload gambar ke GitHub"
            respn.Response = err.Error()
            at.WriteJSON(respw, http.StatusInternalServerError, respn)
            return
        }

        gambarTokoURL = *content.Content.HTMLURL
    }

    // Ambil data dari form yang akan di-update
    namaToko := req.FormValue("nama_toko")
    slug := req.FormValue("slug")
    category := req.FormValue("category")
    street := req.FormValue("alamat.street")
    province := req.FormValue("alamat.province")
    city := req.FormValue("alamat.city")
    description := req.FormValue("alamat.description")
    postalCode := req.FormValue("alamat.postal_code")

    // Buat data update untuk di MongoDB
    updateData := bson.M{}
    if namaToko != "" {
        updateData["nama_toko"] = namaToko
    }
    if slug != "" {
        updateData["slug"] = slug
    }
    if category != "" {
        updateData["category"] = category
    }
    if street != "" {
        updateData["alamat.street"] = street
    }
    if province != "" {
        updateData["alamat.province"] = province
    }
    if city != "" {
        updateData["alamat.city"] = city
    }
    if description != "" {
        updateData["alamat.description"] = description
    }
    if postalCode != "" {
        updateData["alamat.postal_code"] = postalCode
    }
    if gambarTokoURL != "" {
        updateData["gambar_toko"] = gambarTokoURL
    }

    // Lakukan update di MongoDB
    update := bson.M{"$set": updateData}
    _, err = config.Mongoconn.Collection("menu").UpdateOne(context.TODO(), filter, update)
    if err != nil {
        var respn model.Response
        respn.Status = "Error: Gagal mengupdate toko"
        respn.Response = err.Error()
        at.WriteJSON(respw, http.StatusNotModified, respn)
        return
    }

    // Kirim response sukses
    response := map[string]interface{}{
        "status":  "success",
        "message": "Toko berhasil diupdate",
        "data":    updateData,
    }
    at.WriteJSON(respw, http.StatusOK, response)
}

func CreateCategory(respw http.ResponseWriter, req *http.Request) {
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))

	if err != nil {
		payload, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))

		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Info = at.GetSecretFromHeader(req)
			respn.Location = "Decode Token Error"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	var category model.Category
	if err := json.NewDecoder(req.Body).Decode(&category); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	newCategory := model.Category{
		Name_category: category.Name_category,
	}
	_, err = atdb.InsertOneDoc(config.Mongoconn, "category", newCategory)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal Insert Database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}
	response := map[string]interface{}{
		"status":  "success",
		"message": "Kategori berhasil ditambahkan",
		"name":    payload.Alias,
		"data":    newCategory,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}
