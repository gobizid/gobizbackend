package controller

import (
	"encoding/json"
	"net/http"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateDiskon(respw http.ResponseWriter, req *http.Request) {
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

	filter := bson.M{"user.phonenumber": payload.Id}
	docToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "menu", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Store not found"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var datadiskon model.Diskon
	if err := json.NewDecoder(req.Body).Decode(&datadiskon); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	tokoForDiskon := model.Toko{
		ID:       docToko.ID,
		NamaToko: docToko.NamaToko,
		Category: docToko.Category,
		User:     docToko.User,
	}

	diskonInput := model.Diskon{
		Toko:            []model.Toko{tokoForDiskon},
		JenisDiskon:     datadiskon.JenisDiskon,
		NilaiDiskon:     datadiskon.NilaiDiskon,
		TanggalMulai:    datadiskon.TanggalMulai,
		TanggalBerakhir: datadiskon.TanggalBerakhir,
		Status:          datadiskon.Status,
	}

	InsertData, err := atdb.InsertOneDoc(config.Mongoconn, "diskon", diskonInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to create new diskon"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "Diskon berhasil ditambahkan",
		"data":    InsertData,
	}

	at.WriteJSON(respw, http.StatusCreated, response)
}

func GetAllDiskon(respw http.ResponseWriter, req *http.Request) {
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

	filter := bson.M{
		"toko": bson.M{
			"$elemMatch": bson.M{
				"user": bson.M{
					"$elemMatch": bson.M{
						"phonenumber": bson.M{"$regex": payload.Id},
					},
				},
			},
		},
	}

	diskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Store not found"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}
	response := map[string]interface{}{
		"Status":           "success",
		"message":          "Diskon berhasil ditemukan",
		"toko":             diskon.Toko,
		"jenis_diskon":     diskon.JenisDiskon,
		"nilai_diskon":     diskon.NilaiDiskon,
		"tanggal_mulai":    diskon.TanggalMulai,
		"tanggal_berakhir": diskon.TanggalBerakhir,
		"status":           diskon.Status,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func UpdateDiskon(respw http.ResponseWriter, req *http.Request) {
	// Dekode token untuk validasi
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
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

	// Ambil DiskonID dari URL parameter
	diskonID := req.URL.Query().Get("id")
	if diskonID == "" {
		var respn model.Response
		respn.Status = "Error: DiskonID tidak ditemukan"
		respn.Response = "DiskonID tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Konversi DiskonID menjadi ObjectID MongoDB
	diskonObjID, err := primitive.ObjectIDFromHex(diskonID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid DiskonID"
		respn.Response = "DiskonID format is invalid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cari diskon berdasarkan DiskonID
	diskonFilter := bson.M{"_id": diskonObjID}
	existingDiskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", diskonFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Diskon tidak ditemukan"
		respn.Response = "Diskon dengan ID ini tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Decode request body untuk mendapatkan data diskon baru
	var updateDiskon model.Diskon
	if err := json.NewDecoder(req.Body).Decode(&updateDiskon); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = "Failed to parse request body"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Update diskon yang ada dengan data yang baru
	existingDiskon.JenisDiskon = updateDiskon.JenisDiskon
	existingDiskon.NilaiDiskon = updateDiskon.NilaiDiskon
	existingDiskon.TanggalMulai = updateDiskon.TanggalMulai
	existingDiskon.TanggalBerakhir = updateDiskon.TanggalBerakhir
	existingDiskon.Status = updateDiskon.Status

	// Lakukan update di database
	update := bson.M{
		"$set": existingDiskon,
	}

	_, err = config.Mongoconn.Collection("diskon").UpdateOne(req.Context(), diskonFilter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to update diskon"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Response sukses
	response := map[string]interface{}{
		"status":  "success",
		"message": "Diskon berhasil diperbarui",
		"data":    existingDiskon,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func DeleteDiskon(respw http.ResponseWriter, req *http.Request) {
	// Dekode token untuk validasi
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
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

	// Ambil DiskonID dari URL parameter
	diskonID := req.URL.Query().Get("id")
	if diskonID == "" {
		var respn model.Response
		respn.Status = "Error: DiskonID tidak ditemukan"
		respn.Response = "DiskonID tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Konversi DiskonID menjadi ObjectID MongoDB
	diskonObjID, err := primitive.ObjectIDFromHex(diskonID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid DiskonID"
		respn.Response = "DiskonID format is invalid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cari diskon berdasarkan DiskonID
	diskonFilter := bson.M{"_id": diskonObjID}
	_, err = atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", diskonFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Diskon tidak ditemukan"
		respn.Response = "Diskon dengan ID ini tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Hapus diskon dari database
	_, err = config.Mongoconn.Collection("diskon").DeleteOne(req.Context(), diskonFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal menghapus diskon"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Response sukses
	response := map[string]interface{}{
		"status":  "success",
		"message": "Diskon berhasil dihapus",
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func GetDiskonById(respw http.ResponseWriter, req *http.Request) {
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
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

	diskonID := req.URL.Query().Get("id_diskon")
	if diskonID == "" {
		var respn model.Response
		respn.Status = "Error: DiskonID tidak ditemukan"
		respn.Response = "DiskonID tidak disertakan dalam permintaan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	diskonObjID, err := primitive.ObjectIDFromHex(diskonID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Invalid DiskonID"
		respn.Response = "DiskonID format is invalid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	diskonFilter := bson.M{"_id": diskonObjID}
	diskon, err := atdb.GetOneDoc[model.Diskon](config.Mongoconn, "diskon", diskonFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Diskon tidak ditemukan"
		respn.Response = "Diskon dengan ID ini tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	response := map[string]interface{}{
		"statusResponse":   "success",
		"message":          "Diskon berhasil ditemukan",
		"toko":             diskon.Toko,
		"jenis_diskon":     diskon.JenisDiskon,
		"nilai_diskon":     diskon.NilaiDiskon,
		"tanggal_mulai":    diskon.TanggalMulai,
		"tanggal_berakhir": diskon.TanggalBerakhir,
		"status":           diskon.Status,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}
