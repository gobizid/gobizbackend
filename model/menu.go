package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Menu struct {
	Name          string  `json:"name" bson:"name"`
	Price         int     `json:"price" bson:"price"`
	OriginalPrice int     `json:"originalPrice" bson:"originalPrice"`
	Rating        float64 `json:"rating" bson:"rating"`
	Sold          int     `json:"sold" bson:"sold"`
	Image         string  `json:"image" bson:"image"`
}

type Toko struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	NamaToko string             `bson:"nama_toko" json:"nama_toko"`
	Slug     string             `bson:"slug" json:"slug"`
	Alamat   string             `bson:"alamat" json:"alamat"`
	User     []Userdomyikado    `bson:"user" json:"user"`
	Menu     []Menu             `bson:"menu" json:"menu"`
}
