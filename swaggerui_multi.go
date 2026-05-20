package swaggerui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
)

//go:embed templates
var tmpl embed.FS

var initializerTemplate = template.Must(template.ParseFS(tmpl, "templates/swagger-initializer.js.tmpl"))

// Option configures a Swagger UI handler.
type Option func(*config)

// WithPrimaryName selects the document displayed when Swagger UI loads.
func WithPrimaryName(name string) Option {
	return func(c *config) {
		c.primaryName = name
	}
}

// Spec is an embedded Swagger/OpenAPI document available in the UI selector.
type Spec struct {
	Name    string
	Content []byte
}

type config struct {
	primaryName string
}

// HandlerWithSpecs returns a handler that serves Swagger UI with multiple embedded specs.
func HandlerWithSpecs(specs []Spec, opts ...Option) http.Handler {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	set, err := newSpecSet(specs, cfg.primaryName, "./swagger")
	if err != nil {
		log.Panicf("swaggerui validation: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.HandlerFunc(set.serveSpec)))
	mux.HandleFunc("/swagger-initializer.js", set.serveInitializer)
	mux.Handle("/", http.FileServer(staticFSHandler()))
	return mux
}

type resolvedSpec struct {
	URL     string `json:"url,omitempty"`
	Name    string `json:"name,omitempty"`
	content []byte
}
type specSet struct {
	specs       map[string]resolvedSpec
	primaryName string
	initializer []byte
}

func newSpecSet(specs []Spec, primaryName, rootPath string) (*specSet, error) {
	if len(specs) == 0 {
		return nil, fmt.Errorf("at least one spec is required")
	}
	if primaryName == "" {
		primaryName = specs[0].Name
	}

	set := &specSet{
		specs:       make(map[string]resolvedSpec, len(specs)),
		primaryName: primaryName,
	}

	uniq := make(map[string]struct{}, len(specs))
	for i, spec := range specs {
		// validate spec
		if spec.Name == "" {
			return nil, fmt.Errorf("spec %d has empty name", i)
		}
		if len(spec.Content) == 0 {
			return nil, fmt.Errorf("spec %q has empty content", spec.Name)
		}

		// build spec path in slug format and check for conflicts
		urlPath := fmt.Sprintf("%s.json",
			url.PathEscape(strings.ReplaceAll(strings.ToLower(spec.Name), " ", "_")),
		)
		if _, ok := uniq[urlPath]; ok {
			return nil, fmt.Errorf("spec %q URL conflicts with spec %q", spec.Name, urlPath)
		}
		uniq[urlPath] = struct{}{}

		// build spec URL and add to set
		set.specs[urlPath] = resolvedSpec{
			Name:    spec.Name,
			URL:     fmt.Sprintf("%s/%s", rootPath, urlPath),
			content: spec.Content,
		}
	}
	set.initializer = set.swaggerInitializer()

	return set, nil
}

func (s *specSet) serveSpec(w http.ResponseWriter, r *http.Request) {
	spec, ok := s.specs[url.PathEscape(r.URL.Path)]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = w.Write(spec.content)
}

func (s *specSet) serveInitializer(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(s.initializer)
}

type initializerData struct {
	URLsJSON        string
	PrimaryNameJSON string
}

func (s *specSet) swaggerInitializer() []byte {
	urls := make([]resolvedSpec, 0, len(s.specs))
	for _, spec := range s.specs {
		urls = append(urls, spec)
	}

	data := initializerData{
		URLsJSON:        mustJSON(urls, "urls"),
		PrimaryNameJSON: mustJSON(s.primaryName, "primary name"),
	}

	var b bytes.Buffer
	if err := initializerTemplate.Execute(&b, data); err != nil {
		log.Panicf("render initializer: %v", err)
	}
	return b.Bytes()
}

func mustJSON(v any, label string) string {
	b, err := json.Marshal(v)
	if err != nil {
		log.Panicf("swaggerui: marshal %s: %v", label, err)
	}
	return string(b)
}
