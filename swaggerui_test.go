package swaggerui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesSingleSpecAndAssets(t *testing.T) {
	handler := Handler([]byte(`{"openapi":"3.1.0"}`))

	body, status := get(t, handler, "/swagger_spec")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if body != `{"openapi":"3.1.0"}` {
		t.Fatalf("unexpected spec body: %q", body)
	}

	_, status = get(t, handler, "/swagger-ui.css")
	if status != http.StatusOK {
		t.Fatalf("expected static asset status 200, got %d", status)
	}
}

func TestHandlerWithSpecsServesEachSpec(t *testing.T) {
	handler := HandlerWithSpecs([]Spec{
		{Name: "Public API", Content: []byte(`{"openapi":"3.1.0","info":{"title":"public"}}`)},
		{Name: "Admin API", Content: []byte(`{"openapi":"3.1.0","info":{"title":"admin"}}`)},
		{Name: `Special / ? % " API`, Content: []byte(`{"openapi":"3.1.0","info":{"title":"special"}}`)},
	})

	body, status := get(t, handler, "/swagger/public_api.json")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !strings.Contains(body, `"title":"public"`) {
		t.Fatalf("unexpected first spec body: %q", body)
	}

	body, status = get(t, handler, "/swagger/admin_api.json")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !strings.Contains(body, `"title":"admin"`) {
		t.Fatalf("unexpected second spec body: %q", body)
	}

	body, status = get(t, handler, "/swagger/special_%2F_%3F_%25_%22_api.json")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}
	if !strings.Contains(body, `"title":"special"`) {
		t.Fatalf("unexpected special spec body: %q", body)
	}

	_, status = get(t, handler, "/swagger/missing_api.json")
	if status != http.StatusNotFound {
		t.Fatalf("expected missing spec status 404, got %d", status)
	}

	_, status = get(t, handler, "/swagger/public_api")
	if status != http.StatusNotFound {
		t.Fatalf("expected spec without .json status 404, got %d", status)
	}
}

func TestHandlerWithSpecsInitializer(t *testing.T) {
	handler := HandlerWithSpecs([]Spec{
		{Name: `Public "API"`, Content: []byte(`public`)},
		{Name: "Admin API", Content: []byte(`admin`)},
	}, WithPrimaryName("Admin API"))

	body, status := get(t, handler, "/swagger-initializer.js")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	want := []string{
		`urls: [{"url":"./swagger/public_%22api%22.json","name":"Public \"API\""},{"url":"./swagger/admin_api.json","name":"Admin API"}]`,
		`"urls.primaryName": "Admin API"`,
		`SwaggerUIStandalonePreset`,
		`SwaggerUIBundle.plugins.DownloadUrl`,
		`layout: "StandaloneLayout"`,
	}
	for _, s := range want {
		if !strings.Contains(body, s) {
			t.Fatalf("initializer missing %q in:\n%s", s, body)
		}
	}
}

func TestHandlerWithSpecsInitializerDefaultsToFirstSpec(t *testing.T) {
	handler := HandlerWithSpecs([]Spec{
		{Name: "Public API", Content: []byte(`public`)},
		{Name: "Admin API", Content: []byte(`admin`)},
	})

	body, status := get(t, handler, "/swagger-initializer.js")
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	if !strings.Contains(body, `"urls.primaryName": "Public API"`) {
		t.Fatalf("initializer should default to first spec in:\n%s", body)
	}
}

func TestHandlerWithSpecsPanicsOnInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "empty specs",
			fn: func() {
				HandlerWithSpecs(nil)
			},
		},
		{
			name: "empty name",
			fn: func() {
				HandlerWithSpecs([]Spec{{Content: []byte("spec")}})
			},
		},
		{
			name: "empty content",
			fn: func() {
				HandlerWithSpecs([]Spec{{Name: "API"}})
			},
		},
		{
			name: "duplicate name",
			fn: func() {
				HandlerWithSpecs([]Spec{
					{Name: "API", Content: []byte("one")},
					{Name: "API", Content: []byte("two")},
				})
			},
		},
		{
			name: "duplicate slug",
			fn: func() {
				HandlerWithSpecs([]Spec{
					{Name: "Public API", Content: []byte("one")},
					{Name: "public_api", Content: []byte("two")},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic")
				}
			}()
			tt.fn()
		})
	}
}

func get(t *testing.T, handler http.Handler, path string) (body string, statusCode int) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	resp := rec.Result()
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return string(bodyBytes), resp.StatusCode
}
