package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gocroot/config"
	"github.com/gocroot/helper/at"
	"github.com/gocroot/helper/atapi"
	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/jualin"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
)

// Fungsi untuk menangani request order
func HandleOrder(w http.ResponseWriter, r *http.Request) {
	namalapak := at.GetParam(r)
	var orderRequest jualin.PaymentRequest

	// Decode JSON request ke struct
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&orderRequest); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	_, err := atdb.InsertOneDoc(config.Mongoconn, "order", orderRequest)
	if err != nil {
		http.Error(w, "Insert Database Gagal", http.StatusBadRequest)
		return
	}

	//kirim pesan ke tenant
	message := "*Pesanan Masuk " + namalapak + "*\n" + orderRequest.User.Name + "\n" + orderRequest.User.Whatsapp + "\n" + orderRequest.User.Address + "\n" + createOrderMessage(orderRequest.Orders) + "\nTotal: " + strconv.Itoa(orderRequest.Total) + "\nPembayaran: " + orderRequest.PaymentMethod
	newmsg := model.SendText{
		To:       "6282184952582",
		IsGroup:  false,
		Messages: message,
	}
	_, _, err = atapi.PostStructWithToken[model.Response]("token", config.WAAPIToken, newmsg, config.WAAPIMessage)
	if err != nil {
		http.Error(w, "Gagal Mengirim pesan", http.StatusBadRequest)
		return
	}
	// Cetak data order ke terminal (bisa diganti dengan logic lain, misal menyimpan ke database)
	fmt.Printf("Received Order: %+v\n", orderRequest)

	// Kirim response kembali ke client
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"status": "success", "message": "Order received"}
	json.NewEncoder(w).Encode(response)
}

// Fungsi untuk membuat pesan dari orders
func createOrderMessage(orders []jualin.Order) string {
	var orderStrings []string

	for _, order := range orders {
		orderString := fmt.Sprintf("%s x%d - Rp %d", order.Name, order.Quantity, order.Price)
		orderStrings = append(orderStrings, orderString)
	}

	// Gabungkan semua orders menjadi satu string dengan new line sebagai separator
	return strings.Join(orderStrings, "\n")
}

func GetDataOrder(w http.ResponseWriter, r *http.Request) {
	// Variabel untuk menampung hasil query
	var orders []jualin.Order

	// Menggunakan helper GetAllDoc untuk mengambil data dari MongoDB
	filter := bson.M{} // Filter kosong untuk mengambil semua dokumen
	orders, err := atdb.GetAllDoc[[]jualin.Order](config.Mongoconn, "order", filter)
	if err != nil {
		http.Error(w, "Gagal mendapatkan data order", http.StatusInternalServerError)
		return
	}

	// Mengembalikan data order dalam bentuk JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
