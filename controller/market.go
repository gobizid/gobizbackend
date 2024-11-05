package controller

import (
	"context"
	"io"
	"net/http"
	"strconv"
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
	// Decode token and get user information
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

	hashedFileName := ghupload.CalculateHash(fileContent) + header.Filename[strings.LastIndex(header.Filename, "."):] // Add file extension

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
	categoryID := req.FormValue("category_id")
	latitudeStr := req.FormValue("latitude")
	longtitudeStr := req.FormValue("longtitude")
	descriptionMarket := req.FormValue("description")
	rating := req.FormValue("rating")
	openingHours := req.FormValue("opening_hours")
	street := req.FormValue("alamat.street")
	province := req.FormValue("alamat.province")
	city := req.FormValue("alamat.city")
	description := req.FormValue("alamat.description")
	postalCode := req.FormValue("alamat.postal_code")

	// Konversi latitude dan longitude ke float64
	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Latitude tidak valid"
		respn.Response = "Latitude harus berupa angka desimal"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	longtitude, err := strconv.ParseFloat(longtitudeStr, 64)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Longitude tidak valid"
		respn.Response = "Longitude harus berupa angka desimal"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	objectCategoryID, err := primitive.ObjectIDFromHex(categoryID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Kategori ID tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	categoryDoc, err := atdb.GetOneDoc[model.Category](config.Mongoconn, "category", primitive.M{"_id": objectCategoryID})
	if err != nil || categoryDoc.ID == primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Kategori tidak ditemukan"
		respn.Response = "ID yang dicari: " + categoryID
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	docTokoUser, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", primitive.M{"user.phonenumber": payload.Id})
	if err == nil && docTokoUser.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: User sudah memiliki toko"
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	docTokoNama, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", primitive.M{"nama_toko": namaToko})
	if err == nil && docTokoNama.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Nama Toko sudah digunakan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Validasi format slug, jika mengandung spasi maka ubah spasi menjadi "-"
	// if strings.Contains(slug, " ") {
	// 	slug = strings.ReplaceAll(slug, " ", "-")
	// }

	// Validasi slug tidak mengandung spasi
	if strings.Contains(slug, " ") {
		var respn model.Response
		respn.Status = "Error: Slug tidak boleh mengandung spasi. Gunakan format 'nama-toko'."
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	docTokoSlug, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", primitive.M{"slug": slug})
	if err == nil && docTokoSlug.ID != primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Slug Toko sudah digunakan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Fetch user document
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	tokoInput := model.Toko{
		NamaToko:     namaToko,
		Slug:         slug,
		Category:     categoryDoc,
		Latitude:     latitude,
		Longtitude:   longtitude,
		GambarToko:   gambarTokoURL,
		Description:  descriptionMarket,
		Rating:       rating,
		OpeningHours: openingHours,
		Alamat: model.Address{
			Street:      street,
			Province:    province,
			City:        city,
			Description: description,
			PostalCode:  postalCode,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		User: []model.Userdomyikado{docuser},
	}

	dataToko, err := atdb.InsertOneDoc(config.Mongoconn, "toko", tokoInput)
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
		},
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func GetAllMarket(respw http.ResponseWriter, req *http.Request) {
	tokos, err := atdb.GetAllDoc[[]model.Toko](config.Mongoconn, "toko", primitive.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data market tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	if len(tokos) == 0 {
		var respn model.Response
		respn.Status = "Error: Data market kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var allMarkets []map[string]interface{}

	for _, toko := range tokos {
		allMarkets = append(allMarkets, map[string]interface{}{
			"id":          toko.ID.Hex(),
			"nama_toko":   toko.NamaToko,
			"slug":        toko.Slug,
			"category":    toko.Category,
			"gambar_toko": toko.GambarToko,
			"alamat": map[string]interface{}{
				"street":      toko.Alamat.Street,
				"province":    toko.Alamat.Province,
				"city":        toko.Alamat.City,
				"description": toko.Alamat.Description,
				"postal_code": toko.Alamat.PostalCode,
			},
			"user": toko.User,
		})
	}

	at.WriteJSON(respw, http.StatusOK, allMarkets)
}

func UpdateToko(respw http.ResponseWriter, req *http.Request) {
	// Ambil token dari header
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

	// Ambil ID toko dari query parameter
	tokoID := req.URL.Query().Get("id")
	if tokoID == "" {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Konversi tokoID dari string ke ObjectID MongoDB
	objectID, err := primitive.ObjectIDFromHex(tokoID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Ambil data toko dari database
	var existingToko model.Toko
	filter := bson.M{"_id": objectID}
	err = config.Mongoconn.Collection("toko").FindOne(context.TODO(), filter).Decode(&existingToko)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Cek apakah user adalah pemilik toko
	if existingToko.User[0].PhoneNumber != payload.Id {
		var respn model.Response
		respn.Status = "Error: User tidak memiliki hak akses"
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Parsing form data
	err = req.ParseMultipartForm(10 << 20)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal memproses form data"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Validasi gambar toko (opsional)
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
		hashedFileName := ghupload.CalculateHash(fileContent) + header.Filename[strings.LastIndex(header.Filename, "."):]
		// Upload gambar ke GitHub
		content, _, err := ghupload.GithubUpload(config.GHAccessToken, "Rolly Maulana Awangga", "awangga@gmail.com", fileContent, "gobizid", "img", "tokoImages/"+hashedFileName, true)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Gagal mengupload gambar ke GitHub"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusInternalServerError, respn)
			return
		}
		gambarTokoURL = *content.Content.HTMLURL
	}

	// Validasi dan Ambil Data dari Form
	namaToko := req.FormValue("nama_toko")
	slug := req.FormValue("slug")
	category := req.FormValue("category")
	street := req.FormValue("alamat.street")
	province := req.FormValue("alamat.province")
	city := req.FormValue("alamat.city")
	description := req.FormValue("alamat.description")
	postalCode := req.FormValue("alamat.postal_code")

	// Validasi slug tidak mengandung spasi
	if strings.Contains(slug, " ") {
		var respn model.Response
		respn.Status = "Error: Slug tidak boleh mengandung spasi. Gunakan format 'nama-toko'."
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Validasi kategori jika ada
	if category != "" {
		objectCategoryID, err := primitive.ObjectIDFromHex(category)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: ID Kategori tidak valid"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}
		categoryDoc, err := atdb.GetOneDoc[model.Category](config.Mongoconn, "category", primitive.M{"_id": objectCategoryID})
		if err != nil || categoryDoc.ID == primitive.NilObjectID {
			var respn model.Response
			respn.Status = "Error: Kategori tidak ditemukan"
			respn.Response = "ID yang dicari: " + category
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}
	}

	// Validasi nama toko unik
	if namaToko != "" && namaToko != existingToko.NamaToko {
		docTokoNama, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", primitive.M{"nama_toko": namaToko})
		if err == nil && docTokoNama.ID != primitive.NilObjectID {
			var respn model.Response
			respn.Status = "Error: Nama Toko sudah digunakan"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}
	}

	// Update Data
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

	update := bson.M{"$set": updateData}
	_, err = config.Mongoconn.Collection("toko").UpdateOne(context.TODO(), filter, update)
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

func GetTokoByID(respw http.ResponseWriter, req *http.Request) {
	tokoID := req.URL.Query().Get("id")
	if tokoID == "" {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(tokoID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	filter := bson.M{"_id": objectID}
	dataToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	data := model.Toko{
		ID:         dataToko.ID,
		NamaToko:   dataToko.NamaToko,
		Slug:       dataToko.Slug,
		Category:   dataToko.Category,
		Alamat:     dataToko.Alamat,
		GambarToko: dataToko.GambarToko,
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Toko ditemukan",
		"data":    data,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func DeleteTokoByID(respw http.ResponseWriter, req *http.Request) {
	// Ambil token dari header
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

	// Ambil ID toko dari query parameter
	tokoID := req.URL.Query().Get("id")
	if tokoID == "" {
		var respn model.Response
		respn.Status = "Error: ID Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Konversi tokoID dari string ke ObjectID MongoDB
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
	err = config.Mongoconn.Collection("toko").FindOne(context.TODO(), filter).Decode(&existingToko)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Cek apakah user yang melakukan penghapusan adalah pemilik toko
	if existingToko.User[0].PhoneNumber != payload.Id {
		var respn model.Response
		respn.Status = "Error: User tidak memiliki hak akses untuk menghapus toko ini"
		at.WriteJSON(respw, http.StatusForbidden, respn)
		return
	}

	// Coba hapus data toko dari database berdasarkan ID
	result, err := config.Mongoconn.Collection("toko").DeleteOne(context.TODO(), filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal menghapus toko"
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Jika tidak ada dokumen yang dihapus, berarti toko tidak ditemukan
	if result.DeletedCount == 0 {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Kirim response sukses
	response := map[string]interface{}{
		"status":  "success",
		"message": "Toko berhasil dihapus",
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetAllMarketAddress(respw http.ResponseWriter, req *http.Request) {
	// Mengambil semua data toko dari collection 'toko' yang berisi informasi alamat dan user
	tokos, err := atdb.GetAllDoc[[]model.Toko](config.Mongoconn, "toko", primitive.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data toko tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Jika tidak ada data toko
	if len(tokos) == 0 {
		var respn model.Response
		respn.Status = "Error: Data toko kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var allAddress []map[string]interface{}

	for _, toko := range tokos {
		address := toko.Alamat
		for _, user := range toko.User {
			allAddress = append(allAddress, map[string]interface{}{
				"street":      address.Street,
				"province":    address.Province,
				"city":        address.City,
				"description": address.Description,
				"postal_code": address.PostalCode,
				"user": map[string]interface{}{
					"nama": user.Name,
				},
			})
		}
	}

	// Mengembalikan data market dalam format JSON
	at.WriteJSON(respw, http.StatusOK, allAddress)
}

func GetAllSlug(respw http.ResponseWriter, req *http.Request) {
	// Mengambil semua data toko dari collection 'menu'
	tokos, err := atdb.GetAllDoc[[]model.Toko](config.Mongoconn, "toko", primitive.M{})
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
			"id":        toko.ID.Hex(),
			"nama_toko": toko.NamaToko,
			"slug":      toko.Slug,
			"user":      toko.User, // Tambahkan informasi user jika diperlukan
		})
	}

	// Mengembalikan data market dalam format JSON
	at.WriteJSON(respw, http.StatusOK, allMarkets)
}

func GetAllNamaToko(respw http.ResponseWriter, req *http.Request) {
	tokos, err := atdb.GetAllDoc[[]model.Toko](config.Mongoconn, "toko", primitive.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data market tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	if len(tokos) == 0 {
		var respn model.Response
		respn.Status = "Error: Data market kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var allMarkets []map[string]interface{}

	for _, toko := range tokos {
		allMarkets = append(allMarkets, map[string]interface{}{
			"id":        toko.ID.Hex(),
			"nama_toko": toko.NamaToko,
			"user":      toko.User,
		})
	}

	at.WriteJSON(respw, http.StatusOK, allMarkets)
}

func GetPageMenuByToko(respw http.ResponseWriter, req *http.Request) {
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	slug := req.URL.Query().Get("slug")
	if slug == "" {
		var respn model.Response
		respn.Status = "Error: Slug tidak ditemukan"
		respn.Response = "Slug tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	var toko model.Toko
	err = config.Mongoconn.Collection("toko").FindOne(req.Context(), bson.M{"slug": slug}).Decode(&toko)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		respn.Response = "Slug: " + slug + ", Error: " + err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var menus []model.Menu
	cursor, err := config.Mongoconn.Collection("menu").Find(req.Context(), bson.M{"toko": toko.ID})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengambil data menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}
	defer cursor.Close(req.Context())

	if err = cursor.All(req.Context(), &menus); err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal memproses data menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}
	alamat := map[string]interface{}{
		"street":      toko.Alamat.Street,
		"province":    toko.Alamat.Province,
		"city":        toko.Alamat.City,
		"description": toko.Alamat.Description,
		"postal_code": toko.Alamat.PostalCode,
	}
	owner := map[string]interface{}{
		"name":        toko.User[0].Name,
		"phonenumber": toko.User[0].PhoneNumber,
		"email":       toko.User[0].Email,
	}

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Menu berhasil diambil",
		"nama_toko": toko.NamaToko,
		"slug":      toko.Slug,
		"category":  toko.Category.CategoryName,
		"alamat":    alamat,
		"owner":     owner,
		"data":      menus,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}
