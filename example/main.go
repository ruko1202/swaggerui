package main

import (
	_ "embed"
	"log"
	"net/http"

	"github.com/ruko1202/swaggerui"
)

//go:embed spec/petstore.yml
var spec []byte

func main() {
	log.SetFlags(0)
	mux := http.NewServeMux()
	mux.HandleFunc("/pet/", petHandler)
	mux.Handle("/swagger/", http.StripPrefix("/swagger", swaggerui.Handler(spec)))
	log.Println("serving on http://localhost:8080")
	log.Println("swagger on http://localhost:8080/swagger")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
