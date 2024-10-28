package model

type PaymentOrder struct {
	Orders        []Orders        `json:"order" bson:"order"`
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

type Orders struct {
	Menu     []Menu `json:"menu" bson:"menu"`
	Quantity int    `json:"quantity" bson:"quantity"`
}
