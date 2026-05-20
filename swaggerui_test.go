package swaggerui_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/swaggerui"
)

var (
	initializerURLsPattern        = regexp.MustCompile(`(?m)^\s*urls:\s*(\[.*\]),$`)
	initializerPrimaryNamePattern = regexp.MustCompile(`(?m)^\s*"urls\.primaryName":\s*(".*"),$`)
)

func TestHandler_ServesSingleSpecAndAssets(t *testing.T) {
	t.Parallel()

	handler := swaggerui.Handler([]byte(`{"openapi":"3.1.0"}`))

	testCases := map[string]struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		"should_serve_embedded_spec_when_swagger_spec_is_requested": {
			path:       "/swagger_spec",
			wantStatus: http.StatusOK,
			wantBody:   `{"openapi":"3.1.0"}`,
		},
		"should_serve_static_asset_when_asset_exists": {
			path:       "/swagger-ui.css",
			wantStatus: http.StatusOK,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body, status := get(t, handler, tc.path)

			require.Equal(t, tc.wantStatus, status)
			if tc.wantBody != "" {
				assert.Equal(t, tc.wantBody, body)
			}
		})
	}
}

func TestHandlerWithSpecs_ServesSpecsByGeneratedURL(t *testing.T) {
	t.Parallel()

	handler := swaggerui.HandlerWithSpecs([]swaggerui.Spec{
		{Name: "Public API", Content: []byte(`{"openapi":"3.1.0","info":{"title":"public"}}`)},
		{Name: "Admin API", Content: []byte(`{"openapi":"3.1.0","info":{"title":"admin"}}`)},
		{Name: `Special / ? % " API`, Content: []byte(`{"openapi":"3.1.0","info":{"title":"special"}}`)},
	})

	testCases := map[string]struct {
		path         string
		wantStatus   int
		wantBodyPart string
	}{
		"should_serve_public_spec_when_public_api_slug_is_requested": {
			path:         "/swagger/public_api.json",
			wantStatus:   http.StatusOK,
			wantBodyPart: `"title":"public"`,
		},
		"should_serve_admin_spec_when_admin_api_slug_is_requested": {
			path:         "/swagger/admin_api.json",
			wantStatus:   http.StatusOK,
			wantBodyPart: `"title":"admin"`,
		},
		"should_serve_special_spec_when_url_contains_escaped_special_characters": {
			path:         "/swagger/special_%2F_%3F_%25_%22_api.json",
			wantStatus:   http.StatusOK,
			wantBodyPart: `"title":"special"`,
		},
		"should_return_not_found_when_spec_does_not_exist": {
			path:       "/swagger/missing_api.json",
			wantStatus: http.StatusNotFound,
		},
		"should_return_not_found_when_json_extension_is_missing": {
			path:       "/swagger/public_api",
			wantStatus: http.StatusNotFound,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body, status := get(t, handler, tc.path)

			require.Equal(t, tc.wantStatus, status, body)
			if tc.wantBodyPart != "" {
				assert.Contains(t, body, tc.wantBodyPart)
			}
		})
	}
}

func TestHandlerWithSpecs_GeneratesInitializer(t *testing.T) {
	t.Parallel()

	handler := swaggerui.HandlerWithSpecs([]swaggerui.Spec{
		{Name: `Public "API"`, Content: []byte(`public`)},
		{Name: "Admin API", Content: []byte(`admin`)},
	}, swaggerui.WithPrimaryName("Admin API"))

	body, status := get(t, handler, "/swagger-initializer.js")
	require.Equal(t, http.StatusOK, status)

	config := parseInitializerConfig(t, body)

	wantURLs := map[string]string{
		`Public "API"`: "./swagger/public_%22api%22.json",
		"Admin API":    "./swagger/admin_api.json",
	}
	require.Len(t, config.URLs, len(wantURLs))
	for _, got := range config.URLs {
		wantURL, ok := wantURLs[got.Name]
		require.True(t, ok, "unexpected spec in initializer: %#v", got)
		assert.Equal(t, wantURL, got.URL, "spec name: %s", got.Name)
	}
	assert.Equal(t, "Admin API", config.PrimaryName)

	for _, want := range []string{
		"SwaggerUIStandalonePreset",
		"SwaggerUIBundle.plugins.DownloadUrl",
		`layout: "StandaloneLayout"`,
	} {
		assert.Contains(t, body, want)
	}
}

func TestHandlerWithSpecs_DefaultsPrimaryNameToFirstSpec(t *testing.T) {
	t.Parallel()

	handler := swaggerui.HandlerWithSpecs([]swaggerui.Spec{
		{Name: "Public API", Content: []byte(`public`)},
		{Name: "Admin API", Content: []byte(`admin`)},
	})

	body, status := get(t, handler, "/swagger-initializer.js")
	require.Equal(t, http.StatusOK, status)

	config := parseInitializerConfig(t, body)
	assert.Equal(t, "Public API", config.PrimaryName)
}

func TestHandlerWithSpecs_PanicsOnInvalidConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		fn func()
	}{
		"should_panic_when_specs_are_empty": {
			fn: func() {
				swaggerui.HandlerWithSpecs(nil)
			},
		},
		"should_panic_when_spec_name_is_empty": {
			fn: func() {
				swaggerui.HandlerWithSpecs([]swaggerui.Spec{{Content: []byte("spec")}})
			},
		},
		"should_panic_when_spec_content_is_empty": {
			fn: func() {
				swaggerui.HandlerWithSpecs([]swaggerui.Spec{{Name: "API"}})
			},
		},
		"should_panic_when_spec_names_generate_same_url": {
			fn: func() {
				swaggerui.HandlerWithSpecs([]swaggerui.Spec{
					{Name: "API", Content: []byte("one")},
					{Name: "api", Content: []byte("two")},
				})
			},
		},
		"should_panic_when_distinct_names_generate_same_url": {
			fn: func() {
				swaggerui.HandlerWithSpecs([]swaggerui.Spec{
					{Name: "Public API", Content: []byte("one")},
					{Name: "public_api", Content: []byte("two")},
				})
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Panics(t, tc.fn)
		})
	}
}

type initializerConfig struct {
	URLs        []initializerURL
	PrimaryName string
}

type initializerURL struct {
	URL  string `json:"url"`
	Name string `json:"name"`
}

func parseInitializerConfig(t *testing.T, body string) initializerConfig {
	t.Helper()

	urlsMatch := initializerURLsPattern.FindStringSubmatch(body)
	require.Len(t, urlsMatch, 2, "initializer urls config not found in:\n%s", body)

	var urls []initializerURL
	require.NoError(t, json.Unmarshal([]byte(urlsMatch[1]), &urls))

	primaryMatch := initializerPrimaryNamePattern.FindStringSubmatch(body)
	require.Len(t, primaryMatch, 2, "initializer primary name config not found in:\n%s", body)

	var primaryName string
	require.NoError(t, json.Unmarshal([]byte(primaryMatch[1]), &primaryName))

	return initializerConfig{
		URLs:        urls,
		PrimaryName: primaryName,
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
	require.NoError(t, err)

	return string(bodyBytes), resp.StatusCode
}
