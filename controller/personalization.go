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

    // Response sukses
    response := map[string]interface{}{
        "status":           "success",
        "personalizationID": personalizationID,
    }

    at.WriteJSON(respw, http.StatusOK, response)
}

