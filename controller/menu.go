package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/ghupload"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func InsertDataMenu(respw http.ResponseWriter, req *http.Request) {
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

	var menuImageURL string
	file, header, err := req.FormFile("menuImage")
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
		GitHubAccessToken := config.GHAccessToken
		GitHubAuthorName := "Rolly Maulana Awangga"
		GitHubAuthorEmail := "awangga@gmail.com"
		githubOrg := "gobizid"
		githubRepo := "img"
		pathFile := "menuImages/" + hashedFileName
		replace := true

		content, _, err := ghupload.GithubUpload(GitHubAccessToken, GitHubAuthorName, GitHubAuthorEmail, fileContent, githubOrg, githubRepo, pathFile, replace)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Gagal mengupload gambar ke GitHub"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusInternalServerError, respn)
			return
		}

		menuImageURL = *content.Content.HTMLURL
	}

	menuName := req.FormValue("name")
	menuPrice := req.FormValue("price")
	menuRating := req.FormValue("rating")
	menuSold := req.FormValue("sold")
	categoryID := req.FormValue("category_id")

	price, _ := strconv.Atoi(menuPrice)
	rating, _ := strconv.ParseFloat(menuRating, 64)
	sold, _ := strconv.Atoi(menuSold)

	filter := bson.M{
		"user.phonenumber": payload.Id,
	}

	existingToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	filterCategory := bson.M{
		"_id": func() primitive.ObjectID {
			objID, err := primitive.ObjectIDFromHex(categoryID)
			if err != nil {
				var respn model.Response
				respn.Status = "Error: Invalid Category ID"
				respn.Response = err.Error()
				at.WriteJSON(respw, http.StatusBadRequest, respn)
				return primitive.NilObjectID
			}
			return objID
		}(),
	}
	dataCategory, err := atdb.GetOneDoc[model.Category](config.Mongoconn, "category", filterCategory)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Category not found"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	menuID := primitive.NewObjectID()

	newMenu := model.Menu{
		ID:       menuID,
		TokoID:   existingToko,
		Name:     menuName,
		Price:    price,
		Category: dataCategory,
		Diskon:   nil,
		Rating:   rating,
		Sold:     sold,
		Image:    menuImageURL,
	}

	_, err = atdb.InsertOneDoc(config.Mongoconn, "menu", newMenu)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal memasukkan data menu ke database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Menu berhasil ditambahkan ke toko",
		"data": map[string]interface{}{
			"menu_id":  menuID.Hex(),
			"name":     newMenu.Name,
			"price":    newMenu.Price,
			"category": newMenu.Category.CategoryName,
			"rating":   newMenu.Rating,
			"toko": map[string]interface{}{
				"id":   existingToko.ID.Hex(),
				"name": existingToko.NamaToko,
				"slug": existingToko.Slug,
			},
		},
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetDataMenu(respw http.ResponseWriter, req *http.Request) {
	// Tambah validasi akses token
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

	data, err := atdb.GetAllDoc[[]model.Menu](config.Mongoconn, "menu", primitive.M{"name": "Sayur Lodeh Gaming"})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data menu tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	if len(data) == 0 {
		var respn model.Response
		respn.Status = "Error: Data menu kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Jika data ditemukan, kirimkan data dalam bentuk JSON
	at.WriteJSON(respw, http.StatusOK, data)
}

func GetAllMenu(respw http.ResponseWriter, req *http.Request) {
	data, err := atdb.GetAllDoc[[]model.Menu](config.Mongoconn, "menu", primitive.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data menu tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var menus []map[string]interface{}
	for _, menu := range data {
		imageUrl := strings.Replace(menu.Image, "github.com", "raw.githubusercontent.com", 1)
		imageUrls := strings.Replace(imageUrl, "/blob/", "/", 1)

		finalPrice := menu.Price
		diskonValue := 0.00
		potonganHarga := 0.00

		if menu.Diskon != nil && menu.Diskon.Status == "Active" {
			if menu.Diskon.JenisDiskon == "Persentase" {
				diskonAmount := float64(menu.Price) * (float64(menu.Diskon.NilaiDiskon) / 100)
				finalPrice = menu.Price - int(diskonAmount)
				diskonValue = float64(menu.Diskon.NilaiDiskon)
				potonganHarga = diskonAmount
			} else if menu.Diskon.JenisDiskon == "Nominal" {
				finalPrice = menu.Price - menu.Diskon.NilaiDiskon
				if finalPrice < 0 {
					finalPrice = 0
				}
				diskonValue = float64(menu.Diskon.NilaiDiskon)
				potonganHarga = float64(menu.Diskon.NilaiDiskon)
			}
		}

		menus = append(menus, map[string]interface{}{
			"toko":         menu.TokoID.NamaToko,
			"menu":         menu.Name,
			"price_awal":   menu.Price,
			"price":        finalPrice,
			"nilai_diskon": diskonValue,
			"diskon":       potonganHarga,
			"rating":       menu.Rating,
			"sold":         menu.Sold,
			"image":        imageUrls,
		})
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Data menu berhasil diambil",
		"data":    menus,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func AddDiskonToMenu(respw http.ResponseWriter, req *http.Request) {
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

	idMenu := req.URL.Query().Get("id_menu")
	if idMenu == "" {
		var respn model.Response
		respn.Status = "Error: ID Menu tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	var requestDiskon struct {
		DiskonID string `json:"diskonId"`
	}
	if err := json.NewDecoder(req.Body).Decode(&requestDiskon); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = "Failed to parse request body"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	diskonObjID, err := primitive.ObjectIDFromHex(requestDiskon.DiskonID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid DiskonID"
		respn.Response = "Invalid diskon ID format"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	menuObjID, err := primitive.ObjectIDFromHex(idMenu)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid ID Menu format"
		respn.Response = "Invalid menu ID format"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	dataDiskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", bson.M{"_id": diskonObjID})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Diskon not found"
		respn.Response = "Diskon with the given ID does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	filter := bson.M{"_id": menuObjID}
	update := bson.M{"diskon": dataDiskon}

	dataMenuUpdate, err := atdb.UpdateOneDoc(config.Mongoconn, "menu", filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to update menu" + err.Error()
		respn.Response = "Could not add discount to the menu"
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	if dataMenuUpdate.MatchedCount == 0 {
		var respn model.Response
		respn.Status = "Error: Menu not found"
		respn.Response = "Menu with the given ID does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	response := map[string]interface{}{
		"user":    payload.Id,
		"message": "Diskon added to the menu successfully",
		"status":  "Success",
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetDataMenuByCategory(respw http.ResponseWriter, req *http.Request) {
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

	category := req.URL.Query().Get("category")
	if category == "" {
		var respn model.Response
		respn.Status = "Error: Kategori tidak ditemukan"
		respn.Response = "Kategori tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	filter := bson.M{"category.name_category": category}
	menus, err := atdb.GetAllDoc[[]model.Menu](config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Menu tidak ditemukan"
		respn.Response = "Category: " + category + ", Error: " + err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var menusByCategory []map[string]interface{}

	for _, menu := range menus {
		imageURL := strings.Replace(menu.Image, "github.com", "raw.githubusercontent.com", 1)
		imageURL = strings.Replace(imageURL, "/blob/", "/", 1)

		finalPrice := menu.Price
		diskonValue := 0.00
		potonganHarga := 0.00

		if menu.Diskon != nil && menu.Diskon.Status == "Active" {
			if menu.Diskon.JenisDiskon == "Persentase" {
				diskonAmount := float64(menu.Price) * (float64(menu.Diskon.NilaiDiskon) / 100)
				finalPrice = menu.Price - int(diskonAmount)
				diskonValue = float64(menu.Diskon.NilaiDiskon)
				potonganHarga = diskonAmount
			} else if menu.Diskon.JenisDiskon == "Nominal" {
				finalPrice = menu.Price - menu.Diskon.NilaiDiskon
				if finalPrice < 0 {
					finalPrice = 0
				}
				diskonValue = float64(menu.Diskon.NilaiDiskon)
				potonganHarga = float64(menu.Diskon.NilaiDiskon)
			}
		}

		menusByCategory = append(menusByCategory, map[string]interface{}{
			"toko_name":      menu.TokoID.NamaToko,
			"name":           menu.Name,
			"price":          menu.Price,
			"final_price":    finalPrice,
			"discount_nilai": diskonValue,
			"diskon":         potonganHarga,
			"rating":         menu.Rating,
			"sold":           menu.Sold,
			"image":          imageURL,
		})
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Menu berhasil diambil berdasarkan kategori",
		"data":    menusByCategory,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func UpdateDataMenu(respw http.ResponseWriter, req *http.Request) {
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		payload, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	menuID := req.URL.Query().Get("id")
	if menuID == "" {
		var respn model.Response
		respn.Status = "Error: ID Menu tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(menuID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Menu tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	filter := bson.M{"_id": objectID}
	dataMenu, err := atdb.GetOneDoc[model.Menu](config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Menu tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
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

	var menuImageURL string = dataMenu.Image
	file, header, err := req.FormFile("menuImage")
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
		GitHubAccessToken := config.GHAccessToken
		GitHubAuthorName := "Rolly Maulana Awangga"
		GitHubAuthorEmail := "awangga@gmail.com"
		githubOrg := "gobizid"
		githubRepo := "img"
		pathFile := "menuImages/" + hashedFileName
		replace := true

		content, _, err := ghupload.GithubUpload(GitHubAccessToken, GitHubAuthorName, GitHubAuthorEmail, fileContent, githubOrg, githubRepo, pathFile, replace)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Gagal mengupload gambar ke GitHub"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusInternalServerError, respn)
			return
		}

		menuImageURL = *content.Content.HTMLURL
	}

	menuName := req.FormValue("name")
	menuPrice := req.FormValue("price")
	menuRating := req.FormValue("rating")
	menuSold := req.FormValue("sold")
	categoryID := req.FormValue("category_id")

	price, _ := strconv.Atoi(menuPrice)
	rating, _ := strconv.ParseFloat(menuRating, 64)
	sold, _ := strconv.Atoi(menuSold)

	// Ambil data kategori berdasarkan ID jika diberikan
	var existingCategory model.Category
	if categoryID != "" {
		categoryObjID, err := primitive.ObjectIDFromHex(categoryID)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: ID Kategori tidak valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}
		categoryFilter := bson.M{"_id": categoryObjID}
		existingCategory, err = atdb.GetOneDoc[model.Category](config.Mongoconn, "category", categoryFilter)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Kategori tidak ditemukan"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusNotFound, respn)
			return
		}
	}

	updateFields := bson.M{
		"name":   menuName,
		"price":  price,
		"rating": rating,
		"sold":   sold,
		"image":  menuImageURL,
	}

	if categoryID != "" {
		updateFields["category"] = existingCategory
	}

	_, err = atdb.UpdateOneDoc(config.Mongoconn, "menu", filter, updateFields)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate data menu di database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Menu berhasil diperbarui",
		"data": map[string]interface{}{
			"nama":    payload.Alias,
			"menu_id": objectID.Hex(),
			"name":    menuName,
			"price":   price,
			"rating":  rating,
			"image":   menuImageURL,
			"category": map[string]interface{}{
				"id":            existingCategory.ID.Hex(),
				"name_category": existingCategory.CategoryName,
			},
		},
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func DeleteDataMenu(respw http.ResponseWriter, req *http.Request) {
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		payload, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	menuId := req.URL.Query().Get("menuId")
	if menuId == "" {
		var respn model.Response
		respn.Status = "Error: ID Menu tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(menuId)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Menu tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	filter := bson.M{"_id": objectID}
	deleteData, err := atdb.DeleteOneDoc(config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal menghapus data menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Menu berhasil dihapus",
		"data": map[string]interface{}{
			"user":    payload.Id,
			"menu_id": objectID.Hex(),
			"deleted": deleteData.DeletedCount,
		},
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func DeleteDiskonFromMenu(respw http.ResponseWriter, req *http.Request) {

}

func UpdateDiskonInMenu(respw http.ResponseWriter, req *http.Request) {
	// Decode and validate token
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

	// Retrieve menu ID and discount ID from request
	idMenu := req.URL.Query().Get("id_menu")
	if idMenu == "" {
		var respn model.Response
		respn.Status = "Error: ID Menu tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	var requestDiskon struct {
		DiskonID string `json:"diskonId"`
	}
	if err := json.NewDecoder(req.Body).Decode(&requestDiskon); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = "Failed to parse request body"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Convert IDs to ObjectID
	menuObjID, err := primitive.ObjectIDFromHex(idMenu)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid ID Menu format"
		respn.Response = "Invalid menu ID format"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	diskonObjID, err := primitive.ObjectIDFromHex(requestDiskon.DiskonID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid DiskonID"
		respn.Response = "Invalid diskon ID format"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Retrieve discount data
	dataDiskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", bson.M{"_id": diskonObjID})
	if err != nil || dataDiskon.ID == primitive.NilObjectID {
		var respn model.Response
		respn.Status = "Error: Diskon not found"
		respn.Response = "Diskon with the given ID does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Update menu document to add discount as an object
	filter := bson.M{"_id": menuObjID}
	update := bson.M{
		"$set": bson.M{
			"diskon": dataDiskon,
		},
	}


	_, err = config.Mongoconn.Collection("menu").UpdateOne(context.TODO(), filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to update menu"
		respn.Response = "Could not update discount in the menu. Error: " + err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"message": "Diskon updated in the menu successfully",
		"status":  "Success",
	}
	at.WriteJSON(respw, http.StatusOK, response)
}


func GetMenuById(respw http.ResponseWriter, req *http.Request) {

}
