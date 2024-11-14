package controller

import (
	"net/http"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetUserProfile(respw http.ResponseWriter, req *http.Request) {
	tokenLogin := at.GetLoginFromHeader(req)
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, tokenLogin)
	if err != nil {
		payload, err = watoken.Decode(config.PUBLICKEY, tokenLogin)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token tidak valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	phonenumber := payload.Id
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": phonenumber})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data pengguna tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	responseData := model.Response{
		Status:   "Success",
		Response: "Data pengguna berhasil diambil",
		Info:     "Profil pengguna ditemukan",
	}

	// Menambahkan data pengguna ke dalam response
	response := map[string]interface{}{
		"response": responseData,
		"data":     docuser,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func GetAllUser(respw http.ResponseWriter, req *http.Request) {
	tokenLogin := at.GetLoginFromHeader(req)
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, tokenLogin)
	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, tokenLogin)
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token tidak valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	data, err := atdb.GetAllDoc[[]model.Userdomyikado](config.Mongoconn, "user", primitive.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: User Tidak Ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	if len(data) == 0 {
		var respn model.Response
		respn.Status = "Error: Data kategori kosong"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	responseData := model.Response{
		Status:   "Success",
		Response: "Data pengguna berhasil diambil",
		Info:     "Profil pengguna ditemukan",
		Location: "GetAllUser",
	}

	response := map[string]interface{}{
		"response": responseData,
		"data":     data,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}
