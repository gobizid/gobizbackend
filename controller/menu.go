package controller

import (
	"encoding/json"
	"fmt"
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

	err = req.ParseMultipartForm(10 << 20) // Batas 10MB
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

		hashedFileName := ghupload.CalculateHash(fileContent) + header.Filename[strings.LastIndex(header.Filename, "."):] // Tambahkan ekstensi asli file

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

	price, _ := strconv.Atoi(menuPrice)
	rating, _ := strconv.ParseFloat(menuRating, 64)
	sold, _ := strconv.Atoi(menuSold)

	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	filter := bson.M{
		"user": bson.M{
			"$elemMatch": bson.M{
				"phonenumber": bson.M{"$regex": payload.Id},
			},
		},
	}

	existingToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		respn.Response = "payload.Id:" + payload.Id + "|" + err.Error() + "|" + "data docuser: " + docuser.PhoneNumber + fmt.Sprintf("%v", filter)
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Create the menu with an empty Diskon (null equivalent)
	newMenu := model.Menu{
		Name:   menuName,
		Price:  price,
		Diskon: nil, // Setting Diskon to nil (null equivalent in Go)
		Rating: rating,
		Sold:   sold,
		Image:  menuImageURL,
	}

	existingToko.Menu = append(existingToko.Menu, newMenu)

	update := bson.M{
		"$set": bson.M{
			"menu": existingToko.Menu,
		},
	}
	_, err = config.Mongoconn.Collection("menu").UpdateOne(req.Context(), filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate data menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Menu berhasil ditambahkan ke toko",
		"data": map[string]interface{}{
			"id":        existingToko.ID.Hex(),
			"nama_toko": existingToko.NamaToko,
			"slug":      existingToko.Slug,
			"category":  existingToko.Category,
			"alamat":    existingToko.Alamat,
			"user":      docuser,
			"menu":      existingToko.Menu,
		},
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetPageMenuByToko(respw http.ResponseWriter, req *http.Request) {
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

	// Ambil parameter slug dari query params
	slug := req.URL.Query().Get("slug")
	if slug == "" {
			var respn model.Response
			respn.Status = "Error: Slug tidak ditemukan"
			respn.Response = "Slug tidak disertakan dalam permintaan"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
	}

	// Cari toko berdasarkan slug
	var toko model.Toko
	err = config.Mongoconn.Collection("menu").FindOne(req.Context(), bson.M{"slug": slug}).Decode(&toko)
	if err != nil {
			var respn model.Response
			respn.Status = "Error: Toko tidak ditemukan"
			respn.Response = "Slug: " + slug + ", Error: " + err.Error()
			at.WriteJSON(respw, http.StatusNotFound, respn)
			return
	}

	// Jika toko ditemukan, kembalikan data menu toko tersebut
	response := map[string]interface{}{
			"status":    "success",
			"message":   "Menu berhasil diambil",
			"nama_toko": toko.NamaToko,
			"slug":      toko.Slug,
			"category":  toko.Category,
			"alamat":    toko.Alamat,
			"owner":     toko.User,
			"data":      toko.Menu,
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
	data, err := atdb.GetAllDoc[[]model.Toko](config.Mongoconn, "menu", primitive.M{})
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

	var allMenus []map[string]interface{}

	for _, toko := range data {
		for _, menu := range toko.Menu {
			// Calculate the final price considering the discount
			finalPrice := menu.Price

			// Check if the menu has a valid discount
			if len(menu.Diskon) > 0 {
				// Assuming the first discount in the array is applied
				diskon := menu.Diskon[0]
				if diskon.JenisDiskon == "Persentase" {
					// Apply percentage discount
					discountValue := float64(menu.Price) * float64(diskon.NilaiDiskon) / 100
					finalPrice = menu.Price - int(discountValue)
				}
			}

			// Append the menu details including the discounted price
			allMenus = append(allMenus, map[string]interface{}{
				"name":        menu.Name,
				"price":       menu.Price,
				"final_price": finalPrice, // Include the final price after discount
				"diskon":      menu.Diskon,
				"rating":      menu.Rating,
				"sold":        menu.Sold,
				"image":       menu.Image,
			})
		}
	}

	at.WriteJSON(respw, http.StatusOK, allMenus)
}

func GetDataMenuByCategory(respw http.ResponseWriter, req *http.Request) {
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

	category := req.URL.Query().Get("category")
	if category == "" {
			var respn model.Response
			respn.Status = "Error: Kategori tidak ditemukan"
			respn.Response = "Kategori tidak disertakan dalam permintaan"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
	}

	var menu model.Menu
	err = config.Mongoconn.Collection("menu").FindOne(req.Context(), bson.M{"category": category}).Decode(&menu)
	if err != nil {
			var respn model.Response
			respn.Status = "Error: Menu tidak ditemukan"
			respn.Response = "Category: " + category + ", Error: " + err.Error()
			at.WriteJSON(respw, http.StatusNotFound, respn)
			return
	}

	response := map[string]interface{}{
			"status":  "success",
			"message": "Menu berhasil diambil",
			"name":    menu.Name,
			"image":   menu.Image,
			"diskon":  menu.Diskon,
			"price":   menu.Price,
			"rating":  menu.Rating,
			"sold":    menu.Sold,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func InsertDiskonToMenu(respw http.ResponseWriter, req *http.Request) {
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

	menuIndexStr := req.URL.Query().Get("menu_index")
	if menuIndexStr == "" {
		var respn model.Response
		respn.Status = "Error: Index Menu tidak ditemukan"
		respn.Response = "Index Menu tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	menuIndex, err := strconv.Atoi(menuIndexStr)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid MenuIndex"
		respn.Response = "MenuIndex should be a valid integer"
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

	filter := bson.M{"user.phonenumber": payload.Id}
	MenuDataToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		respn.Response = "Toko with the user's phone number does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	if menuIndex < 0 || menuIndex >= len(MenuDataToko.Menu) {
		var respn model.Response
		respn.Status = "Error: Menu index out of bounds"
		respn.Response = fmt.Sprintf("Invalid menu index: %d, Menu length: %d, data toko: %v", menuIndex, len(MenuDataToko.Menu), MenuDataToko.NamaToko)
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	DataDiskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", bson.M{"_id": diskonObjID})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Diskon not found"
		respn.Response = "Diskon with the given ID does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	MenuDataToko.Menu[menuIndex].Diskon = append(MenuDataToko.Menu[menuIndex].Diskon, DataDiskon)

	update := bson.M{
		"$set": bson.M{
			"menu": MenuDataToko.Menu,
		},
	}

	_, err = config.Mongoconn.Collection("menu").UpdateOne(req.Context(), filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to update menu with diskon"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	var respn model.Response
	user := payload.Id
	respn.Status = "Success"
	respn.Response = "Diskon added to the menu successfully"

	responseData := map[string]interface{}{
		"user":    user,
		"message": respn.Response,
		"status":  respn.Status,
	}

	at.WriteJSON(respw, http.StatusOK, responseData)
}

// Belum fix
func UpdateDiskonInMenu(respw http.ResponseWriter, req *http.Request) {
	// Dekode token untuk validasi
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

	// Ambil parameter menu_index dari query string
	menuIndexStr := req.URL.Query().Get("menu_index")
	if menuIndexStr == "" {
		var respn model.Response
		respn.Status = "Error: Index Menu tidak ditemukan"
		respn.Response = "Index Menu tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Konversi menuIndex menjadi integer
	menuIndex, err := strconv.Atoi(menuIndexStr)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid MenuIndex"
		respn.Response = "MenuIndex should be a valid integer"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Dekode request body untuk mendapatkan DiskonID
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

	// Konversi DiskonID menjadi ObjectID MongoDB
	diskonObjID, err := primitive.ObjectIDFromHex(requestDiskon.DiskonID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid DiskonID"
		respn.Response = "Invalid diskon ID format"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Filter berdasarkan nomor telepon user dari token payload
	filter := bson.M{"user.phonenumber": payload.Id}
	MenuDataToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		respn.Response = "Toko with the user's phone number does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Cek apakah menuIndex valid
	if menuIndex < 0 || menuIndex >= len(MenuDataToko.Menu) {
		var respn model.Response
		respn.Status = "Error: Menu index out of bounds"
		respn.Response = fmt.Sprintf("Invalid menu index: %d, Menu length: %d", menuIndex, len(MenuDataToko.Menu))
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cari diskon berdasarkan DiskonID
	DataDiskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", bson.M{"_id": diskonObjID})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Diskon not found"
		respn.Response = "Diskon with the given ID does not exist"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Update diskon pada menu di index yang ditentukan
	MenuDataToko.Menu[menuIndex].Diskon = append(MenuDataToko.Menu[menuIndex].Diskon, DataDiskon)

	// Lakukan update pada database
	update := bson.M{
		"$set": bson.M{
			"menu": MenuDataToko.Menu,
		},
	}

	_, err = config.Mongoconn.Collection("menu").UpdateOne(req.Context(), filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to update menu with diskon"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Response berhasil
	var respn model.Response
	respn.Status = "Success"
	respn.Response = "Diskon updated in the menu successfully"

	responseData := map[string]interface{}{
		"user":    payload.Id,
		"message": respn.Response,
		"status":  respn.Status,
	}

	at.WriteJSON(respw, http.StatusOK, responseData)
}

