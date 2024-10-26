package model

type PaymentOrder struct {
	Menu          []Menu          `json:"menu" bson:"menu"`
	Total         int             `json:"total" bson:"total"`
	User          []Userdomyikado `json:"user" bson:"user"`
	Payment       string          `json:"payment" bson:"payment"`
	PaymentMethod string          `json:"paymentMethod" bson:"paymentMethod"`
}

type PaymentOrdertDev struct {
	Menu          []Menu          `json:"menu" bson:"menu"`
	Total         int             `json:"total" bson:"total"`
	User          []Userdomyikado `json:"user" bson:"user"`
	Payment       string          `json:"payment" bson:"payment"`
	PaymentMethod string          `json:"paymentMethod" bson:"paymentMethod"`
}
