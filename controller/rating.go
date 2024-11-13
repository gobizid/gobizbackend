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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddRatingToMenu(respw http.ResponseWriter, req *http.Request) {
	// Decode token untuk mendapatkan nomor telepon
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		_, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	// Ambil user berdasarkan nomor telepon dari payload
	filter := bson.M{"user.phonenumber": payload.Id}
	UserId, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	IdUser := UserId.ID

	// Ambil `menuId` dari query parameter
	menuIDStr := req.URL.Query().Get("menuId")
	if menuIDStr == "" {
		var respn model.Response
		respn.Status = "Error: Menu ID tidak ditemukan"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	menuID, err := primitive.ObjectIDFromHex(menuIDStr)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Menu ID tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Parse request body untuk mendapatkan rating dan review
	var ratingData struct {
		Rating float64 `json:"rating"`
		Review string  `json:"review"`
	}

	if err := json.NewDecoder(req.Body).Decode(&ratingData); err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal memproses data rating"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	// Buat rating baru
	newRating := model.Rating{
		ID:        primitive.NewObjectID(),
		MenuID:    menuID,
		UserID:    IdUser,
		Rating:    ratingData.Rating,
		Review:    ratingData.Review,
		Timestamp: time.Now(),
	}

	// Simpan rating ke koleksi `rating` menggunakan helper `InsertOneDoc`
	_, err = atdb.InsertOneDoc(config.Mongoconn, "rating", newRating)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal menyimpan data rating"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "menu_id", Value: menuID}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$menu_id"},
			{Key: "averageRating", Value: bson.D{{Key: "$avg", Value: "$rating"}}},
			{Key: "ratingCount", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	var result struct {
		AverageRating float64 `bson:"averageRating"`
		RatingCount   int     `bson:"ratingCount"`
	}
	err = atdb.AggregateDoc(config.Mongoconn, "rating", pipeline, &result)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal menghitung rata-rata rating menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	updateData := bson.M{"$set": bson.M{"rating": result.AverageRating}}
	_, err = atdb.UpdateOneDoc(config.Mongoconn, "menu", bson.M{"_id": menuID}, updateData)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate rating menu"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	var respn model.Response
	respn.Status = "success"
	respn.Response = "Rating berhasil ditambahkan dan rata-rata rating menu diperbarui"
	at.WriteJSON(respw, http.StatusOK, respn)
}
