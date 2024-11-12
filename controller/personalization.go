package controller

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreatePersonalization(respw http.ResponseWriter, req *http.Request) {
	// Decode token untuk validasi
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

	// Decode request body untuk mendapatkan data personalization
	var personalization model.Personalization
	if err := json.NewDecoder(req.Body).Decode(&personalization); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = "Failed to parse request body"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Tambahkan data pengguna ke personalization
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}
	personalization.User = []model.Userdomyikado{docuser}

	// Atur ID dan waktu pembuatan/pengecekan
	personalization.ID = primitive.NewObjectID()
	personalization.CreatedAt = time.Now()
	personalization.UpdatedAt = time.Now()

	// Insert personalization ke database
	personalizationID, err := atdb.InsertOneDoc(config.Mongoconn, "personalization", personalization)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal membuat Personalization"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Ambil data personalization yang baru saja dimasukkan
	savedPersonalization, err := atdb.GetOneDoc[model.Personalization](config.Mongoconn, "personalization", primitive.M{"_id": personalizationID})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengambil data Personalization"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Response sukses
	response := map[string]interface{}{
		"status":            "success",
		"personalizationID": savedPersonalization.ID,
		"data":              savedPersonalization, // Kembalikan data personalization yang telah disimpan
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func GetPersonalization(respw http.ResponseWriter, req *http.Request) {
	// Decode token untuk validasi pengguna
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

	// Ambil data pengguna dari database
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data pengguna tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Cari data personalization berdasarkan pengguna yang login
	var personalizations []model.Personalization
	filter := primitive.M{"user.phonenumber": docuser.Phonenumber}
	cursor, err := atdb.FindDocs(config.Mongoconn, "personalization", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengambil data Personalization"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Decode hasil pencarian menjadi slice Personalization
	if err := cursor.All(req.Context(), &personalizations); err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal membaca data Personalization"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Jika tidak ditemukan data personalization
	if len(personalizations) == 0 {
		var respn model.Response
		respn.Status = "Data Personalization tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Response sukses
	response := map[string]interface{}{
		"status": "success",
		"data":   personalizations,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}
