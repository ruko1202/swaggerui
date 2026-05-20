package main

import (
	_ "embed"
	"log"
	"net/http"

	"github.com/ruko1202/swaggerui"
)

//go:embed spec/petstore.yml
var petStoreSpec []byte

//go:embed spec/admin.yml
var adminSpec []byte

func main() {
	log.SetFlags(0)
	mux := http.NewServeMux()
	mux.HandleFunc("/pet/", petHandler)
	mux.Handle("/swagger-single/", http.StripPrefix("/swagger-single", swaggerui.Handler(petStoreSpec)))
	mux.Handle("/swagger-multi/", http.StripPrefix("/swagger-multi", swaggerui.HandlerWithSpecs(
		[]swaggerui.Spec{
			{Name: "Petstore API", Content: petStoreSpec},
			{Name: "Admin API", Content: adminSpec},
		},
		swaggerui.WithPrimaryName("Petstore API"),
	)))
	log.Println("serving on http://localhost:8080")
	log.Println("swagger on http://localhost:8080/swagger-single")
	log.Println("multi swagger on http://localhost:8080/swagger-multi")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
