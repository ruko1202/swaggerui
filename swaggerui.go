package swaggerui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate go run generator/generate.go

var (
	//go:embed assets
	assets    embed.FS
	assetsDir = "assets"
)

// Handler returns a handler that will serve a self-hosted Swagger UI with your spec embedded.
func Handler(spec []byte) http.Handler {
	// render the index template with the proper spec name inserted
	mux := http.NewServeMux()
	mux.HandleFunc("/swagger_spec", byteHandler(spec))
	mux.Handle("/", http.FileServer(staticFSHandler()))
	return mux
}

func staticFSHandler() http.FileSystem {
	static, _ := fs.Sub(assets, assetsDir)

	return http.FS(static)
}

func byteHandler(b []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(b)
	}
}
