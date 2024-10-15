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

func InsertDataMenu(respw http.ResponseWriter, req *http.Request) {
	// Dekode token dari header untuk memverifikasi pengguna
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

	// Decode body request menjadi struct Toko
	var tokoInput model.Toko
	err = json.NewDecoder(req.Body).Decode(&tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Body tidak valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Cek apakah user yang melakukan request ada di koleksi "user" berdasarkan nomor telepon
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	tokoInput.User = []model.Userdomyikado{docuser}
	// Cek apakah toko sudah ada di database
	filter := bson.M{"slug": tokoInput.Slug} // Mencari toko berdasarkan slug unik
	// existingToko := model.Toko{}
	// docuser, err = atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", filter).Decode(&existingToko)
	// if err != nil {
	// 	// Jika toko tidak ditemukan, kembalikan response error
	// 	var respn model.Response
	// 	respn.Status = "Error: Data toko tidak ditemukan"
	// 	respn.Response = err.Error()
	// 	at.WriteJSON(respw, http.StatusNotFound, respn)
	// 	return
	// }

	// Jika toko ditemukan, update data menu toko tersebut
	update := bson.M{
		"$set": bson.M{
			"menu": tokoInput.Menu, // Mengupdate data menu berdasarkan input
		},
	}
	_, err = config.Mongoconn.Collection("toko").UpdateOne(req.Context(), filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate data menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Response sukses jika data menu berhasil diupdate
	var respn model.Response
	respn.Status = "Success: Data menu berhasil diupdate"
	respn.Response = "Menu has been updated"
	at.WriteJSON(respw, http.StatusOK, respn)
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
