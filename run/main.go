package main

import (
	"fmt"
	"net/http"

	"github.com/gocroot/route"
)

func main() {
	fmt.Println("Server is running on http://localhost:8080")
	http.HandleFunc("/", route.URL)
	http.ListenAndServe(":8080", nil)
}
