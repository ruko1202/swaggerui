# swaggerui
Based on [flowchartsman/swaggerui](https://github.com/flowchartsman/swaggerui)

Embedded, self-hosted [Swagger Ui](https://swagger.io/tools/swagger-ui/) for go servers

This module provides `swaggerui.Handler`, which you can use to serve an embedded copy of [Swagger UI](https://swagger.io/tools/swagger-ui/) as well as an embedded specification for your API.

## Install

```shell
go get github.com/ruko1202/swaggerui
```

## Development

Update embedded Swagger UI assets:

```shell
make update-assets
```

## Example usage

### Single spec

```go
package main

import (
	_ "embed"
	"log"
	"net/http"

	"github.com/ruko1202/swaggerui"
)

//go:embed swagger.json
var spec []byte

func main() {
	log.SetFlags(0)
	http.Handle("/swagger/", http.StripPrefix("/swagger", swaggerui.Handler(spec)))
	log.Println("serving on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Multiple specs

```go
package main

import (
	_ "embed"
	"log"
	"net/http"

	"github.com/ruko1202/swaggerui"
)

//go:embed public-swagger.json
var publicSpec []byte

//go:embed admin-swagger.json
var adminSpec []byte

func main() {
	log.SetFlags(0)
	http.Handle("/swagger/", http.StripPrefix("/swagger", swaggerui.HandlerWithSpecs(
		[]swaggerui.Spec{
			{Name: "Public API", Content: publicSpec},
			{Name: "Admin API", Content: adminSpec},
		},
		swaggerui.WithPrimaryName("Public API"),
	)))
	log.Println("serving on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
