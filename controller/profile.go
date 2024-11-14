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
		Location: "GetUserProfile",
	}

	// Menambahkan data pengguna ke dalam response
	response := map[string]interface{}{
		"response": responseData,
		"data":     docuser,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}