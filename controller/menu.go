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

	// Cek apakah user yang melakukan request ada di koleksi "user" berdasarkan nomor telepon dari token (payload.Id)
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	filter := bson.M{"user.phonenumber": payload.Id} // Cari toko berdasarkan nomor telepon pemilik
	existingToko, err := atdb.GetOneDoc[model.Toko](config.Mongoconn, "toko", filter)
	if err != nil {
		// Jika toko tidak ditemukan, kembalikan response error
		var respn model.Response
		respn.Status = "Error: Toko tidak ditemukan"
		respn.Response = "Toko untuk pengguna ini tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Jika toko ditemukan, tambahkan menu baru ke array menu yang sudah ada
	existingToko.Menu = append(existingToko.Menu, tokoInput.Menu...)

	// Update data menu toko di database
	update := bson.M{
		"$set": bson.M{
			"menu": existingToko.Menu, // Update dengan menu yang baru ditambahkan
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

func CreateToko(respw http.ResponseWriter, req *http.Request) {
	// Validasi token
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

	// Set user yang membuat toko
	tokoInput.User = []model.Userdomyikado{docuser}

	// Jika data menu tidak ada (kosong atau null), inisialisasi sebagai array kosong
	if tokoInput.Menu == nil {
		tokoInput.Menu = []model.Menu{}
	}

	// Insert data toko ke database
	dataToko, err := atdb.InsertOneDoc(config.Mongoconn, "menu", tokoInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Gagal Insert Database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}

	// Buat response sukses dengan data yang lebih lengkap
	response := map[string]interface{}{
		"status":  "success",
		"message": "Toko berhasil dibuat",
		"data": map[string]interface{}{
			"id":        dataToko,
			"nama_toko": tokoInput.NamaToko,
			"slug":      tokoInput.Slug,
			"alamat":    tokoInput.Alamat,
			"user":      tokoInput.User, // informasi user
			"menu":      tokoInput.Menu, // informasi menu
		},
	}

	// Response sukses dengan data lengkap
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
