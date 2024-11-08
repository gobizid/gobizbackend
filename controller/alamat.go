package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateAlamat(respw http.ResponseWriter, req *http.Request) {
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

	var dataAddress model.Address
	err = json.NewDecoder(req.Body).Decode(&dataAddress)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Body Tidak Valid"
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

	addAlamat := model.Address{
		ID:          primitive.NewObjectID(),
		Street:      dataAddress.Street,
		Province:    dataAddress.Province,
		City:        dataAddress.City,
		Description: dataAddress.Description,
		PostalCode:  dataAddress.PostalCode,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		User:        []model.Userdomyikado{docuser},
	}

	_, err = atdb.InsertOneDoc(config.Mongoconn, "address", addAlamat)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal Insert Data"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	at.WriteJSON(respw, http.StatusOK, addAlamat)
}

func UpdateAlamat(respw http.ResponseWriter, req *http.Request) {
	// Ambil dan decode token dari header
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

	// Ambil ID alamat dari query parameter
	addressID := req.URL.Query().Get("id")
	if addressID == "" {
		var respn model.Response
		respn.Status = "Error: ID Alamat tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Coba konversi addressID ke ObjectID MongoDB
	objectID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Alamat tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Decode body request untuk mendapatkan data alamat baru
	var updatedData model.Address
	err = json.NewDecoder(req.Body).Decode(&updatedData)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Body Tidak Valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Ambil data user berdasarkan nomor telepon di payload token
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	// Cek apakah alamat yang akan di-update milik user yang sedang login
	filter := bson.M{"_id": objectID, "user.phonenumber": docuser.PhoneNumber}
	_, err = atdb.GetOneDoc[model.Address](config.Mongoconn, "address", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Alamat tidak ditemukan atau Anda tidak memiliki hak akses"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Buat data update
	updateData := bson.M{
		"street":      updatedData.Street,
		"province":    updatedData.Province,
		"city":        updatedData.City,
		"description": updatedData.Description,
		"postal_code": updatedData.PostalCode,
		"updated_at":  time.Now(),
	}

	update := bson.M{"$set": updateData}
	_, err = atdb.UpdateOneDoc(config.Mongoconn, "address", filter, update)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate alamat"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotModified, respn)
		return
	}

	// Kirim response sukses
	response := map[string]interface{}{
		"status":  "success",
		"message": "Alamat berhasil diupdate",
		"data":    updateData,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func DeleteAlamat(respw http.ResponseWriter, req *http.Request) {
	// Ambil dan decode token dari header
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

	// Ambil ID alamat dari query parameter
	addressID := req.URL.Query().Get("id")
	if addressID == "" {
		var respn model.Response
		respn.Status = "Error: ID Alamat tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Coba konversi addressID ke ObjectID MongoDB
	objectID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Alamat tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Ambil data user berdasarkan nomor telepon di payload token
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	// Cek apakah alamat yang akan dihapus milik user yang sedang login
	filter := bson.M{"_id": objectID, "user.phonenumber": docuser.PhoneNumber}
	result, err := atdb.DeleteOneDoc(config.Mongoconn, "address", filter)
	if err != nil || result.DeletedCount == 0 {
		var respn model.Response
		respn.Status = "Error: Alamat tidak ditemukan atau Anda tidak memiliki hak akses"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Kirim response sukses
	response := map[string]interface{}{
		"status":  "success",
		"message": "Alamat berhasil dihapus",
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetAlamatByID(respw http.ResponseWriter, req *http.Request) {
	// Decode token dari header untuk mendapatkan data user
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

	// Ambil ID alamat dari query parameter
	addressID := req.URL.Query().Get("id")
	if addressID == "" {
		var respn model.Response
		respn.Status = "Error: ID Alamat tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Coba konversi addressID ke ObjectID MongoDB
	objectID, err := primitive.ObjectIDFromHex(addressID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: ID Alamat tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Ambil data user berdasarkan nomor telepon di payload token
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	// Ambil data alamat dari MongoDB berdasarkan ID dan nomor telepon user
	var alamat model.Address
	filter := bson.M{"_id": objectID, "user.phonenumber": docuser.PhoneNumber}
	err = config.Mongoconn.Collection("address").FindOne(context.TODO(), filter).Decode(&alamat)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Alamat tidak ditemukan atau Anda tidak memiliki hak akses"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Kirim response dengan data alamat
	response := map[string]interface{}{
		"status": "success",
		"data":   alamat,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func GetAllCities(respw http.ResponseWriter, req *http.Request) {
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
	collection := config.Mongoconn.Collection("address")

	// Distinct untuk mengambil kota unik
	cities, err := collection.Distinct(req.Context(), "city", bson.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to fetch cities"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	at.WriteJSON(respw, http.StatusOK, cities)
}

func GetAllProvinces(respw http.ResponseWriter, req *http.Request) {
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
	collection := config.Mongoconn.Collection("address")

	// Distinct untuk mengambil provinsi unik
	provinces, err := collection.Distinct(req.Context(), "province", bson.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to fetch provinces"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	at.WriteJSON(respw, http.StatusOK, provinces)
}

func GetAllPostalCodes(respw http.ResponseWriter, req *http.Request) {
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

	collection := config.Mongoconn.Collection("address")

	// Distinct untuk mengambil kode pos unik
	postalCodes, err := collection.Distinct(req.Context(), "postal_code", bson.M{})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Failed to fetch postal codes"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	at.WriteJSON(respw, http.StatusOK, postalCodes)
}
