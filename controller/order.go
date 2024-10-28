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
	"github.com/gocroot/helper/watoken"
	"github.com/gocroot/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func CreateOrder(respw http.ResponseWriter, req *http.Request) {
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

	docuser, err := atdb.GetOneDoc[model.Userdomyikado](config.Mongoconn, "user", primitive.M{"phonenumber": payload.Id})
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Data user tidak ditemukan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusNotFound, respn)
		return
	}

	var orderRequest struct {
		Menu          []string `json:"menu"`
		Quantity      []int    `json:"quantity"`
		Payment       string   `json:"payment"`
		PaymentMethod string   `json:"paymentMethod"`
	}
	namaToko := req.URL.Query().Get("namatoko")
	if err := json.NewDecoder(req.Body).Decode(&orderRequest); err != nil {
		var respn model.Response
		respn.Status = "Error: Bad Request"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	if len(orderRequest.Menu) != len(orderRequest.Quantity) {
		var respn model.Response
		respn.Status = "Error: Jumlah menu dan kuantitas tidak sesuai"
		respn.Response = "Jumlah item menu harus sama dengan jumlah item quantity"
		at.WriteJSON(respw, http.StatusBadRequest, respn)
		return
	}

	var totalAmount int
	var orderedItems []model.Menu
	for i, menuName := range orderRequest.Menu {
		dataMenu, err := atdb.GetOneDoc[model.Menu](config.Mongoconn, "menu", primitive.M{"name": menuName})
		if err != nil {
			var respn model.Response
			respn.Status = "Error: Data menu tidak ditemukan: " + strings.Join(orderRequest.Menu, ", ")
			respn.Response = err.Error()
			at.WriteJSON(respw, http.StatusNotFound, respn)
			return
		}

		// Apply discount if available
		finalPrice := dataMenu.Price
		if dataMenu.Diskon != nil {
			if dataMenu.Diskon[0].JenisDiskon == "Persentase" {
				discountValue := float64(dataMenu.Price) * float64(dataMenu.Diskon[0].NilaiDiskon) / 100
				finalPrice = dataMenu.Price - int(discountValue)
			} else {
				finalPrice = dataMenu.Price - dataMenu.Diskon[0].NilaiDiskon
			}
		}

		// Update total amount with the final price and quantity
		quantity := orderRequest.Quantity[i]
		totalAmount += finalPrice * quantity

		// Append ordered item with calculated final price
		orderedItem := model.Menu{
			Name:   dataMenu.Name,
			Price:  finalPrice,
			Image:  dataMenu.Image,
			Rating: dataMenu.Rating,
			Sold:   dataMenu.Sold,
		}
		orderedItems = append(orderedItems, orderedItem)
	}

	// Create order input with total amount and ordered items
	orderInput := model.PaymentOrder{
		User:          []model.Userdomyikado{docuser},
		Orders:        []model.Orders{{Menu: orderedItems, Quantity: 1}},
		Total:         totalAmount,
		Payment:       orderRequest.Payment,
		PaymentMethod: orderRequest.PaymentMethod,
	}

	response, err := atdb.InsertOneDoc(config.Mongoconn, "order", orderInput)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal Insert Database"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	// Message to WhatsApp
	message := fmt.Sprintf("*Pesanan Masuk %s*\nNama: %s\nNo HP: %s\nAlamat: %s\n%s\nTotal: Rp %d\nPembayaran: %s",
		namaToko, docuser.Name, docuser.PhoneNumber, docuser.Address, createOrderMessageDev(orderedItems, orderRequest.Quantity), totalAmount, orderRequest.PaymentMethod)
	newmsg := model.SendText{
		To:       "6282184952582",
		IsGroup:  false,
		Messages: message,
	}
	_, _, err = atapi.PostStructWithToken[model.Response]("token", config.WAAPIToken, newmsg, config.WAAPIMessage)
	if err != nil {
		var respn model.Response
		respn.Status = "Error: Gagal Mengirim Pesan"
		respn.Response = err.Error()
		at.WriteJSON(respw, http.StatusInternalServerError, respn)
		return
	}

	at.WriteJSON(respw, http.StatusOK, response)
}

func createOrderMessageDev(orders []model.Menu, quantities []int) string {
	var orderStrings []string

	for i, order := range orders {
		orderString := fmt.Sprintf("%s x%d - Rp %d", order.Name, quantities[i], order.Price*quantities[i])
		orderStrings = append(orderStrings, orderString)
	}

	// Gabungkan semua orders menjadi satu string dengan new line sebagai separator
	return strings.Join(orderStrings, "\n")
}
