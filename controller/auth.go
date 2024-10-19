package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/auth"
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

var PrivateKey string = os.Getenv("PRKEY")

func Auth(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid request"})
		return
	}

	// Ambil kredensial dari database
	creds, err := atdb.GetOneDoc[auth.GoogleCredential](config.Mongoconn, "credentials", bson.M{})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"message": "Database Connection Problem: Unable to fetch credentials"})
		return
	}

	// Verifikasi ID token menggunakan client_id
	payload, err := auth.VerifyIDToken(request.Token, creds.ClientID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid token: Token verification failed"})
		return
	}

	userInfo := model.Userdomyikado{
		Name:                 payload.Claims["name"].(string),
		Email:                payload.Claims["email"].(string),
		GoogleProfilePicture: payload.Claims["picture"].(string),
	}

	// Simpan atau perbarui informasi pengguna di database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := config.Mongoconn.Collection("user")
	filter := bson.M{"email": userInfo.Email}

	var existingUser model.Userdomyikado
	err = collection.FindOne(ctx, filter).Decode(&existingUser)
	if err != nil || existingUser.PhoneNumber == "" {
		// User does not exist or exists but has no phone number, request QR scan
		response := map[string]interface{}{
			"message": "Please scan the QR code to provide your phone number",
			"user":    userInfo,
			"token":   "",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	} else if existingUser.PhoneNumber != "" {
		token, err := watoken.EncodeforHours(existingUser.PhoneNumber, existingUser.Name, config.PrivateKey, 18) // Generating a token for 18 hours
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "Token generation failed"})
			return
		}
		response := map[string]interface{}{
			"message": "Authenticated successfully",
			"user":    userInfo,
			"token":   token,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	update := bson.M{
		"$set": userInfo,
	}
	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"message": "Failed to save user info: Database update failed"})
		return
	}

	response := map[string]interface{}{
		"user": userInfo,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func GeneratePasswordHandler(respw http.ResponseWriter, r *http.Request) {
	var request struct {
		PhoneNumber string `json:"phonenumber"`
		Captcha     string `json:"captcha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		var respn model.Response
		respn.Status = "Invalid Request"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}
	// Validate CAPTCHA
	captchaResponse, err := http.PostForm("https://challenges.cloudflare.com/turnstile/v0/siteverify", url.Values{
		"secret":   {"0x4AAAAAAAfj2NjfaHRBhkd2VjcfmRe5gvI"},
		"response": {request.Captcha},
	})
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to verify captcha"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusServiceUnavailable, respn)
		return
	}
	defer captchaResponse.Body.Close()

	var captchaResult struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(captchaResponse.Body).Decode(&captchaResult); err != nil {
		var respn model.Response
		respn.Status = "Failed to decode captcha response"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}
	if !captchaResult.Success {
		var respn model.Response
		respn.Status = "Unauthorized"
		respn.Response = "Invalid captcha"
		at.WriteJSON(respw, http.StatusUnauthorized, respn)
		return
	}

	// Validate phone number
	re := regexp.MustCompile(`^62\d{9,15}$`)
	if !re.MatchString(request.PhoneNumber) {
		var respn model.Response
		respn.Status = "Bad Request"
		respn.Response = "Invalid phone number format"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Check if phone number exists in the 'user' collection
	userFilter := bson.M{"phonenumber": request.PhoneNumber}
	_, err = atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", userFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Unauthorized"
		respn.Response = "Phone number not registered"
		at.WriteJSON(respw, http.StatusUnauthorized, respn)
		return
	}

	// Generate random password
	randomPassword, err := auth.GenerateRandomPassword(12)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to generate password"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(randomPassword)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to hash password"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Update or insert the user in the database
	stpFilter := bson.M{"phonenumber": request.PhoneNumber}
	_, err = atdb.GetOneDoc[model.Stp](config.Mongoconn, "stp", stpFilter)
	var responseMessage string

	if err == mongo.ErrNoDocuments {
		// Document not found, insert new one
		newUser := model.Stp{
			PhoneNumber:  request.PhoneNumber,
			PasswordHash: hashedPassword,
			CreatedAt:    time.Now(),
		}
		_, err = atdb.InsertOneDoc(config.Mongoconn, "stp", newUser)
		if err != nil {
			var respn model.Response
			respn.Status = "Failed to insert new user"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusNotModified, respn)
			return
		}
		responseMessage = "New user created and password generated successfully"
	} else {
		// Document found, update the existing one
		stpUpdate := bson.M{
			"phonenumber": request.PhoneNumber,
			"password":    hashedPassword,
			"createdAt":   time.Now(),
		}
		_, err = atdb.UpdateOneDoc(config.Mongoconn, "stp", stpFilter, stpUpdate)
		if err != nil {
			var respn model.Response
			respn.Status = "Failed to update user"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusInternalServerError, respn)
			return
		}
		responseMessage = "User info updated and password generated successfully"
	}

	// Respond with success and the generated password
	response := map[string]interface{}{
		"message":     responseMessage,
		"phonenumber": request.PhoneNumber,
	}
	at.WriteJSON(respw, http.StatusOK, response)

	// Send the random password via WhatsApp
	auth.SendWhatsAppPassword(respw, request.PhoneNumber, randomPassword)
}

var (
	rl = auth.NewRateLimiter(1, 5) // 1 request per second, burst of 5
)

func VerifyPasswordHandler(respw http.ResponseWriter, r *http.Request) {
	var request struct {
		PhoneNumber string `json:"phonenumber"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		var respn model.Response
		respn.Status = "Invalid Request"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Implementasi rate limiting
	limiter := rl.GetLimiter(request.PhoneNumber)
	if !limiter.Allow() {
		var respn model.Response
		respn.Status = "Too Many Requests"
		respn.Response = "Please try again later."
		at.WriteJSON(respw, http.StatusTooManyRequests, respn)
		return
	}

	// Find user in the database
	userFilter := bson.M{"phonenumber": request.PhoneNumber}
	user, err := atdb.GetOneDoc[model.Stp](config.Mongoconn, "stp", userFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to verify password"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusUnauthorized, respn)
		return
	}

	// Verify password and expiry
	if time.Now().After(user.CreatedAt.Add(4 * time.Minute)) {
		var respn model.Response
		respn.Status = "Unauthorized"
		respn.Response = "Password Expired"
		at.WriteJSON(respw, http.StatusUnauthorized, respn)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password))
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to verify password"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusUnauthorized, respn)
		return
	}

	// Find user in the 'user' collection
	myiUserFilter := bson.M{"phonenumber": request.PhoneNumber}
	existingUser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", myiUserFilter)
	if err != nil {
		var respn model.Response
		respn.Status = "Unauthorized"
		respn.Response = "Phone number not registered"
		at.WriteJSON(respw, http.StatusUnauthorized, respn)
		return
	}

	token, err := watoken.EncodeforHours(existingUser.PhoneNumber, existingUser.Name, config.PrivateKey, 18)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to give the token"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"message": "Authenticated successfully",
		"token":   token,
		"name":    existingUser.Name,
	}

	// Respond with success
	at.WriteJSON(respw, http.StatusOK, response)
}

func ResendPasswordHandler(respw http.ResponseWriter, r *http.Request) {
	var request struct {
		PhoneNumber string `json:"phonenumber"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		var respn model.Response
		respn.Status = "Invalid Request"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Generate random password
	randomPassword, err := auth.GenerateRandomPassword(12)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to generate password"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Hash the password
	hashedPassword, err := auth.HashPassword(randomPassword)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to hash password"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Check if phone number exists in the 'stp' collection
	stpFilter := bson.M{"phonenumber": request.PhoneNumber}
	_, stpErr := atdb.GetOneDoc[model.Stp](config.Mongoconn, "stp", stpFilter)

	if stpErr == mongo.ErrNoDocuments {
		// Document not found, insert new one
		newUser := model.Stp{
			PhoneNumber:  request.PhoneNumber,
			PasswordHash: hashedPassword,
			CreatedAt:    time.Now(),
		}
		_, err = atdb.InsertOneDoc(config.Mongoconn, "stp", newUser)
		if err != nil {
			var respn model.Response
			respn.Status = "Failed to insert new user"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusInternalServerError, respn)
			return
		}
		responseMessage := "New user created and password generated successfully"

		// Respond with success and the generated password
		response := map[string]interface{}{
			"message":     responseMessage,
			"phonenumber": request.PhoneNumber,
		}
		at.WriteJSON(respw, http.StatusOK, response)

		// Send the random password via WhatsApp
		auth.SendWhatsAppPassword(respw, request.PhoneNumber, randomPassword)
		return
	} else if stpErr != nil {
		var respn model.Response
		respn.Status = "Failed to fetch user info"
		respn.Response = stpErr.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Document found, update the existing one
	stpUpdate := bson.M{
		"phonenumber": request.PhoneNumber,
		"password":    hashedPassword,
		"createdAt":   time.Now(),
	}
	_, err = atdb.UpdateOneDoc(config.Mongoconn, "stp", stpFilter, stpUpdate)
	if err != nil {
		var respn model.Response
		respn.Status = "Failed to update user"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}
	responseMessage := "User info updated and password generated successfully"

	response := map[string]interface{}{
		"message":     responseMessage,
		"phonenumber": request.PhoneNumber,
	}
	at.WriteJSON(respw, http.StatusOK, response)

	// Send the random password via WhatsApp
	auth.SendWhatsAppPassword(respw, request.PhoneNumber, randomPassword)
}

func RegisterAkunPenjual(respw http.ResponseWriter, r *http.Request) {
	var request model.Userdomyikado
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respn := model.Response{
			Status:   "Invalid Request",
			Response: err.Error(),
		}
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	re := regexp.MustCompile(`^62\d{9,15}$`)
	if !re.MatchString(request.PhoneNumber) {
		respn := model.Response{
			Status:   "Bad Request",
			Response: "Invalid phone number format",
		}
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	hashedPassword, err := auth.HashPassword(request.Password)
	if err != nil {
		respn := model.Response{
			Status:   "Failed to hash password",
			Response: err.Error(),
		}
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	newUser := model.Userdomyikado{
		Name:          request.Name,
		PhoneNumber:   request.PhoneNumber,
		Email:         request.Email,
		Team:          "pd.my.id",
		Scope:         "dev",
		LinkedDevice:  "v4.public.eyJhbGlhcyI6IlJvbGx5IE1hdWxhbmEgQXdhbmdnYSIsImV4cCI6IjIwMjktMDgtMDlUMTQ6MzQ6MjlaIiwiaWF0IjoiMjAyNC0wOC0wOVQwODozNDoyOVoiLCJpZCI6IjYyODEzMTIwMDAzMDAiLCJuYmYiOiIyMDI0LTA4LTA5VDA4OjM0OjI5WiJ9FXnQi5vnQ7YXHteepJ14Xcc-wPc0PLQ0n4LSbGFijfdkStVeD6QIDuwQGeaq7xETWmmtFXjfkmmfDG0WHmAlBA",
		JumlahAntrian: 7,
		Password:      hashedPassword,
	}

	_, err = atdb.InsertOneDoc(config.Mongoconn, "user", newUser)
	if err != nil {
		respn := model.Response{
			Status:   "Failed to insert new user",
			Response: err.Error(),
		}
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	response := map[string]interface{}{
		"message":       "New user created successfully",
		"name":          newUser.Name,
		"phonenumber":   newUser.PhoneNumber,
		"email":         newUser.Email,
		"team":          newUser.Team,
		"scope":         newUser.Scope,
		"jumlahAntrian": newUser.JumlahAntrian,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func LoginAkunPenjual(respw http.ResponseWriter, r *http.Request) {
	var userRequest model.Userdomyikado

	if err := json.NewDecoder(r.Body).Decode(&userRequest); err != nil {
		response := model.Response{
			Status:   "Invalid Request",
			Response: err.Error(),
		}
		at.WriteJSON(respw, http.StatusBadRequest, response)
		return
	}

	var storedUser model.Userdomyikado
	err := config.Mongoconn.Collection("user").FindOne(context.Background(), bson.M{"email": userRequest.Email}).Decode(&storedUser)
	if err != nil {
		response := model.Response{
			Status:   "Error: Toko tidak ditemukan",
			Response: "Error: " + err.Error(),
		}
		at.WriteJSON(respw, http.StatusNotFound, response)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(userRequest.Password))
	if err != nil {
		response := model.Response{
			Status:   "Failed to verify password",
			Response: "Invalid password",
		}
		at.WriteJSON(respw, http.StatusUnauthorized, response)
		return
	}

	encryptedToken, err := watoken.EncodeforHours(storedUser.PhoneNumber, storedUser.Name, PrivateKey, 18)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: token gagal generate"
		respn.Response = ", Error: " + err.Error() + "phonenumber:" + storedUser.PhoneNumber + "|privatekey: " + PrivateKey
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	response := map[string]interface{}{
		"message": "Login successful",
		"name":    storedUser.Name,
		"email":   storedUser.Email,
		"phone":   storedUser.PhoneNumber,
		"team":    storedUser.Team,
		"scope":   storedUser.Scope,
		"token":   encryptedToken,
		"antrian": storedUser.JumlahAntrian,
	}

	at.WriteJSON(respw, http.StatusOK, response)
}
