package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Menu struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TokoID   Toko               `json:"toko" bson:"toko"`
	Name     string             `json:"name" bson:"name"`
	Price    int                `json:"price" bson:"price"`
	Category Category           `json:"category" bson:"category"`
	Diskon   *Diskon            `json:"diskon" bson:"diskon"`
	Rating   float64            `json:"rating" bson:"rating"`
	Sold     int                `json:"sold" bson:"sold"`
	Image    string             `json:"image" bson:"image"`
}

type Rating struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MenuID    primitive.ObjectID `bson:"menu_id" json:"menu_id"`                   // Referensi ke ID menu
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`                   // Referensi ke ID pengguna yang memberi rating
	Rating    float64            `bson:"rating" json:"rating"`                     // Nilai rating (skala 1-5)
	Review    string             `bson:"review,omitempty" json:"review,omitempty"` // Ulasan atau komentar
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`               // Waktu pemberian rating
}

type Toko struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	NamaToko     string             `bson:"nama_toko" json:"nama_toko"`
	Slug         string             `bson:"slug" json:"slug"`
	Category     Category           `bson:"category" json:"category"`
	Location     []GeoJSONFeature   `bson:"location" json:"location"`
	GambarToko   string             `bson:"gambar_toko" json:"gambar_toko"`
	Description  string             `bson:"description" json:"description"`
	Rating       float64            `bson:"rating" json:"rating"`
	OpeningHours OpeningHours       `bson:"opening_hours" json:"opening_hours"`
	Alamat       Address            `bson:"alamat" json:"alamat"`
	User         []Userdomyikado    `bson:"user" json:"user"`
}

type GeoJSONFeature struct {
	Type       string            `bson:"type" json:"type"`
	Properties map[string]string `bson:"properties" json:"properties"`
	Geometry   GeoJSONGeometry   `bson:"geometry" json:"geometry"`
}

type GeoJSONGeometry struct {
	Type        string    `bson:"type" json:"type"`
	Coordinates []float64 `bson:"coordinates" json:"coordinates"`
}

type OpeningHours struct {
	Opening string `bson:"opening" json:"opening"`
	Close   string `bson:"close" json:"close"`
}

type Address struct {
	ID          primitive.ObjectID `bson:"_id" json:"id,omitempty"`
	Street      string             `bson:"street" json:"street,omitempty"`
	Province    string             `bson:"province" json:"province,omitempty"`
	City        string             `bson:"city" json:"city,omitempty"`
	Description string             `bson:"description" json:"description,omitempty"`
	PostalCode  string             `bson:"postal_code" json:"postal_code,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	UpdatedAt   time.Time          `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	User        []Userdomyikado    `bson:"user,omitempty" json:"user,omitempty"`
}

type Category struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Icon         string             `bson:"icon" json:"icon"`
	CategoryName string             `bson:"name_category" json:"name_category"`
}

type Diskon struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	JenisDiskon     string             `bson:"jenis_diskon,omitempty" json:"jenis_diskon,omitempty"`
	NilaiDiskon     int                `bson:"nilai_diskon,omitempty" json:"nilai_diskon,omitempty"`
	TanggalMulai    time.Time          `bson:"tanggal_mulai,omitempty" json:"tanggal_mulai,omitempty"`
	TanggalBerakhir time.Time          `bson:"tanggal_berakhir,omitempty" json:"tanggal_berakhir,omitempty"`
	Status          string             `bson:"status,omitempty" json:"status,omitempty"`
}

type Personalization struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	User            []Userdomyikado    `bson:"user" json:"user"`
	BackgroundColor string             `bson:"background_color" json:"backgroundColor"`
	Font            string             `bson:"font" json:"font"`
	TextColor       string             `bson:"text_color" json:"textColor"`
	ButtonStyle     string             `bson:"button_style" json:"buttonStyle"`
	BorderStyle     string             `bson:"border_style" json:"borderStyle"`   // Menambahkan gaya border (misal: "solid", "dashed", dll)
	BorderColor     string             `bson:"border_color" json:"borderColor"`   // Menambahkan warna border
	ShadowEffect    bool               `bson:"shadow_effect" json:"shadowEffect"` // Efek bayangan
	HeaderFont      string             `bson:"header_font" json:"headerFont"`     // Font untuk header
	HeaderColor     string             `bson:"header_color" json:"headerColor"`   // Warna header
	FooterText      string             `bson:"footer_text" json:"footerText"`     // Teks footer kustom
	FooterColor     string             `bson:"footer_color" json:"footerColor"`   // Warna footer
	LinkStyle       string             `bson:"link_style" json:"linkStyle"`       // Gaya link, misalnya underline atau bold
	LinkColor       string             `bson:"link_color" json:"linkColor"`       // Warna link
	CardStyle       string             `bson:"card_style" json:"cardStyle"`       // Gaya tampilan card (misal: "rounded", "elevated")
	Animation       string             `bson:"animation" json:"animation"`
	CreatedAt       time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updatedAt"`
}
