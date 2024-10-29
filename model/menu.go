package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Menu struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name   string   `json:"name" bson:"name"`
	Price  int      `json:"price" bson:"price"`
	Diskon []Diskon `json:"diskon" bson:"diskon"`
	Rating float64  `json:"rating" bson:"rating"`
	Sold   int      `json:"sold" bson:"sold"`
	Image  string   `json:"image" bson:"image"`
}

type Toko struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	NamaToko   string             `bson:"nama_toko" json:"nama_toko"`
	Slug       string             `bson:"slug" json:"slug"`
	Category   Category           `bson:"category" json:"category"`
	GambarToko string             `bson:"gambar_toko" json:"gambar_toko"`
	Alamat     Address            `bson:"alamat" json:"alamat"`
	User       []Userdomyikado    `bson:"user" json:"user"`
	Menu       []Menu             `bson:"menu" json:"menu"`
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
	CategoryName string             `bson:"name_category" json:"name_category"`
}

type Diskon struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Toko            []Toko             `bson:"toko,omitempty" json:"toko,omitempty"`
	JenisDiskon     string             `bson:"jenis_diskon,omitempty" json:"jenis_diskon,omitempty"`
	NilaiDiskon     int                `bson:"nilai_diskon,omitempty" json:"nilai_diskon,omitempty"`
	TanggalMulai    time.Time          `bson:"tanggal_mulai,omitempty" json:"tanggal_mulai,omitempty"`
	TanggalBerakhir time.Time          `bson:"tanggal_berakhir,omitempty" json:"tanggal_berakhir,omitempty"`
	Status          string             `bson:"status,omitempty" json:"status,omitempty"`
}
