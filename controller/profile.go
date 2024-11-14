package controller

import (
	"encoding/json"
	"net/http"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/auth"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
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
	}

	response := map[string]interface{}{
		"response": responseData,
		"data":     data,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func GetUserByID(respw http.ResponseWriter, req *http.Request) {
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

	UserID := req.URL.Query().Get("id")
	if UserID == "" {
		var respn model.Response
		respn.Status = "Error"
		respn.Response = "ID pengguna tidak ditemukan"
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(UserID)
	if err != nil {
		var respn model.Response
		respn.Status = "Error : ID pengguna tidak valid"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	var data model.Userdomyikado
	filter := bson.M{"_id": objectID}
	_, err = atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: User tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"message": "User ditemukan",
		"data":    data,
	}
	at.WriteJSON(respw, http.StatusOK, response)
}

func UpdateProfile(respw http.ResponseWriter, req *http.Request) {
	// Step 1: Ambil token dari header untuk memverifikasi identitas pengguna
	token := at.GetLoginFromHeader(req)
	if token == "" {
		var respn model.Response
		respn.Status = "Error"
		respn.Response = "Token tidak ditemukan di header"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Step 2: Decode token untuk mendapatkan informasi pengguna
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, token)
	if err != nil {
		// Coba decode dengan kunci publik lain jika gagal
		payload, err = watoken.Decode(config.PUBLICKEY, token)
		if err != nil {
			var respn model.Response
			respn.Status = "Error"
			respn.Location = "Decode Token"
			respn.Response = "Token tidak valid"
			respn.Info = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	// Step 3: Ambil data pengguna berdasarkan phonenumber yang ada di token
	phonenumber := payload.Id // Asumsinya `Id` berisi `phonenumber`
	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": phonenumber})
	if err != nil {
		var respn model.Response
		respn.Status = "Error"
		respn.Location = "Database Lookup"
		respn.Response = "Pengguna tidak ditemukan"
		respn.Info = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	// Step 4: Decode data yang ingin diubah dari request body (update nama, email, password)
	var request struct {
		Name     string `json:"name,omitempty"`
		Email    string `json:"email,omitempty"`
		Password string `json:"password,omitempty"`
		OldPassword string `json:"old_password,omitempty"` // Menambahkan field untuk password lama
	}

	// Decode body request
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		var respn model.Response
		respn.Status = "Error"
		respn.Response = "Failed to parse request body"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Step 5: Verifikasi password lama jika ada perubahan password
	if request.Password != "" {
		if request.OldPassword == "" {
			var respn model.Response
			respn.Status = "Error"
			respn.Response = "Password lama diperlukan untuk mengganti password"
			at.WriteJSON(respw, http.StatusBadRequest, respn)
			return
		}

		// Verifikasi password lama
		err = bcrypt.CompareHashAndPassword([]byte(docuser.Password), []byte(request.OldPassword))
		if err != nil {
			response := model.Response{
				Status:   "Failed to verify password",
				Response: "Password lama tidak valid",
			}
			at.WriteJSON(respw, http.StatusUnauthorized, response)
			return
		}

		// Hash password baru
		hashedPassword, err := auth.HashPassword(request.Password)
		if err != nil {
			var respn model.Response
			respn.Status = "Error"
			respn.Response = "Gagal mengenkripsi password baru"
			at.WriteJSON(respw, http.StatusInternalServerError, respn)
			return
		}

		// Update password baru
		docuser.Password = hashedPassword
	}

	// Step 6: Update profil pengguna jika ada perubahan
	updateFields := bson.M{}
	if request.Name != "" {
		updateFields["name"] = request.Name
	}
	if request.Email != "" {
		updateFields["email"] = request.Email
	}

	// Update password jika ada perubahan
	if request.Password != "" {
		updateFields["password"] = docuser.Password // Gunakan password yang sudah di-hash
	}

	// Step 7: Perbarui data pengguna di database
	_, err = atdb.UpdateOneDoc(config.Mongoconn, "user", bson.M{"phonenumber": phonenumber}, updateFields)
	if err != nil {
		var respn model.Response
		respn.Status = "Error"
		respn.Location = "Database Update"
		respn.Response = "Gagal memperbarui profil pengguna"
		respn.Info = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Step 8: Kirimkan respons sukses
	response := map[string]interface{}{
		"message": "Profil pengguna berhasil diperbarui",
		"name":    docuser.Name,
		"email":   docuser.Email,
		"phone":   docuser.PhoneNumber,
		"role":    docuser.Role,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}