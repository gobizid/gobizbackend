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
	docToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", filter)
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

	diskonInput := model.Diskon{
		Toko:            []model.Toko{docToko},
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
