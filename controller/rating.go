package controller

import (
	"encoding/json"
	"fmt"
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

func GetAllRatingByMenu(respw http.ResponseWriter, req *http.Request) {
	_, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
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

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "menu_id", Value: menuID}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "user"},
			{Key: "localField", Value: "user_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "user_info"},
		}}},
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$user_info"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
		{{Key: "$project", Value: bson.D{
			{Key: "rating", Value: 1},
			{Key: "review", Value: 1},
			{Key: "timestamp", Value: 1},
			{Key: "user_name", Value: "$user_info.name"}, // Menambahkan nama pengguna
		}}},
	}

	var ratings []model.UserRating

	err = atdb.AggregateDoc(config.Mongoconn, "rating", pipeline, &ratings)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengambil data rating"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	var respn model.Response
	respn.Status = "success"
	ratingsJSON, err := json.Marshal(ratings)
	if err != nil {
		respn.Status = "Error: Gagal mengkonversi data rating ke JSON"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}
	respn.Response = string(ratingsJSON)
	at.WriteJSON(respw, http.StatusOK, respn)
}

func AddRatingToMenu(respw http.ResponseWriter, req *http.Request) {
	// Decode token untuk mendapatkan nomor telepon
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		payload, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	filter := bson.M{"phonenumber": payload.Id}

	UserId, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", filter)
	if err != nil {
		var respn model.Response
		IdUser := UserId.ID.Hex()
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error() + " data filter: " + fmt.Sprintf("%v", filter) + " data payload: " + fmt.Sprintf("%v", payload.Id) + " ID user: " + IdUser
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	IdUser := UserId.ID

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

	newRating := model.Rating{
		ID:        primitive.NewObjectID(),
		MenuID:    menuID,
		UserID:    IdUser,
		Rating:    ratingData.Rating,
		Review:    ratingData.Review,
		Timestamp: time.Now(),
	}

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

	updateData := bson.M{"$set": bson.M{"rating": result.AverageRating, "ratingCount": result.RatingCount}}
	_, err = atdb.UpdateOneElement(config.Mongoconn, "menu", bson.M{"_id": menuID}, updateData)
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

func DeleteRating(respw http.ResponseWriter, req *http.Request) {
	payload, err := watoken.Decode(config.PublicKeyWhatsAuth, at.GetLoginFromHeader(req))
	if err != nil {
		payload, err = watoken.Decode(config.PUBLICKEY, at.GetLoginFromHeader(req))
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Token Tidak Valid"
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusForbidden, respn)
			return
		}
	}

	filter := bson.M{"phonenumber": payload.Id}
	UserId, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", filter)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotImplemented, respn)
		return
	}

	menuIDStr := req.URL.Query().Get("menuId")
	ratingIDStr := req.URL.Query().Get("ratingId")
	if menuIDStr == "" || ratingIDStr == "" {
		var respn model.Response
		respn.Status = "Error: Menu ID atau Rating ID tidak ditemukan"
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

	ratingID, err := primitive.ObjectIDFromHex(ratingIDStr)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Rating ID tidak valid"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	_, err = atdb.DeleteOneDoc(config.Mongoconn, "rating", bson.M{"_id": ratingID, "menu_id": menuID, "user_id": UserId.ID})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal menghapus rating"
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
		respn.Status = "Error: Gagal menghitung rata-rata rating menu setelah penghapusan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	updateData := bson.M{"$set": bson.M{"rating": result.AverageRating, "ratingCount": result.RatingCount}}
	_, err = atdb.UpdateOneDoc(config.Mongoconn, "menu", bson.M{"_id": menuID}, updateData)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal mengupdate rating menu setelah penghapusan rating"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	var respn model.Response
	respn.Status = "success"
	respn.Response = "Rating berhasil dihapus dan rata-rata rating menu diperbarui"
	at.WriteJSON(respw, http.StatusOK, respn)
}
