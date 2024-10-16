package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func InsertDataMenu(respw http.ResponseWriter, req *http.Request) {
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

	var tokoInput model.Toko
	err = json.NewDecoder(req.Body).Decode(&tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Body tidak valid"
		respn.Response = err.Error()
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

	existingToko.Menu = append(existingToko.Menu, tokoInput.Menu...)

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
			"alamat":    existingToko.Alamat,
			"user":      docuser,
			"menu":      existingToko.Menu,
		},
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetPageMenuByToko(respw http.ResponseWriter, req *http.Request) {
	// Ambil parameter slug dari query params, bukan URL params
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
	err := config.Mongoconn.Collection("menu").FindOne(req.Context(), bson.M{"slug": slug}).Decode(&toko)
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
		"alamat":    toko.Alamat,
		"owner":     toko.User,
		"data":      toko.Menu,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetDataMenu(respw http.ResponseWriter, req *http.Request) {
	// Mendapatkan data dari MongoDB berdasarkan parameter yang diberikan
	data, err := atdb.GetAllDoc[[]model.Menu](config.Mongoconn, "menu", primitive.M{"name": "Sayur Lodeh Gaming"})
	if err != nil {
		// Jika terjadi error saat mendapatkan data, kembalikan response error
		var respn model.Response
		respn.Status = "Error: Data menu tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Tambahkan pengecekan apakah data kosong
	if len(data) == 0 {
		var respn model.Response
		respn.Status = "Error: Data menu kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Jika data ditemukan, kirimkan data dalam bentuk JSON
	at.WriteJSON(respw, http.StatusOK, data)
}
