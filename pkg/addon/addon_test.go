package addon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"plain semver", "0.13.0", "0.13.0", false},
		{"with lowercase v", "v0.13.0", "0.13.0", false},
		{"with uppercase V", "V1.2.3", "1.2.3", false},
		{"invalid - text", "latest", "", true},
		{"invalid - partial", "1.2", "", true},
		{"invalid - empty", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateVersion(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLatestVersion(t *testing.T) {
	t.Run("returns first tag stripped of v prefix", func(t *testing.T) {
		tags := []struct {
			Name string `json:"name"`
		}{
			{Name: "v0.13.0"},
			{Name: "v0.12.2"},
		}
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(tags)
		}))
		defer srv.Close()

		// Override tagsURL via a test server - we need to use fetchURL-style approach
		// Since tagsURL is a const, we test via httptest by patching the client
		origURL := tagsURL
		// We can't override const, so we test LatestVersion indirectly
		// by using a custom transport that redirects
		client := srv.Client()
		transport := &urlRewriteTransport{
			base:    client.Transport,
			fromURL: origURL,
			toURL:   srv.URL,
		}
		client.Transport = transport

		version, err := LatestVersion(client)
		require.NoError(t, err)
		require.Equal(t, "0.13.0", version)
	})

	t.Run("error on empty tags", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]struct{}{})
		}))
		defer srv.Close()

		client := srv.Client()
		client.Transport = &urlRewriteTransport{
			base:    client.Transport,
			fromURL: tagsURL,
			toURL:   srv.URL,
		}

		_, err := LatestVersion(client)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no tags found")
	})

	t.Run("error on non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		client := srv.Client()
		client.Transport = &urlRewriteTransport{
			base:    client.Transport,
			fromURL: tagsURL,
			toURL:   srv.URL,
		}

		_, err := LatestVersion(client)
		require.Error(t, err)
		require.Contains(t, err.Error(), "HTTP 500")
	})
}

func TestFetchResources(t *testing.T) {
	t.Run("invalid version returns error", func(t *testing.T) {
		_, err := FetchResources(http.DefaultClient, "bad-version")
		require.Error(t, err)
	})

	t.Run("fetches and parses CRDs and manifests", func(t *testing.T) {
		crdYAML := `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: httpscaledobjects.http.keda.sh
`
		manifestYAML := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: keda-add-ons-http-operator
  namespace: keda
`
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case contains(r.URL.Path, "crds"):
				w.Write([]byte(crdYAML))
			default:
				w.Write([]byte(manifestYAML))
			}
		}))
		defer srv.Close()

		client := srv.Client()
		client.Transport = &prefixRewriteTransport{
			base:  client.Transport,
			toURL: srv.URL,
		}

		objs, err := FetchResources(client, "0.13.0")
		require.NoError(t, err)
		require.Len(t, objs, 2)
		// CRDs should come first
		require.Equal(t, "CustomResourceDefinition", objs[0].GetKind())
		require.Equal(t, "Deployment", objs[1].GetKind())
	})

	t.Run("error on CRD fetch failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		client := srv.Client()
		client.Transport = &prefixRewriteTransport{base: client.Transport, toURL: srv.URL}

		_, err := FetchResources(client, "0.13.0")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch addon CRDs")
	})
}

// urlRewriteTransport redirects requests from one URL to another for testing.
type urlRewriteTransport struct {
	base    http.RoundTripper
	fromURL string
	toURL   string
}

func (t *urlRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.String() == t.fromURL {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = t.toURL[len("http://"):]
		req.URL.Path = "/"
	}
	return t.base.RoundTrip(req)
}

// prefixRewriteTransport redirects all requests to a test server.
type prefixRewriteTransport struct {
	base  http.RoundTripper
	toURL string
}

func (t *prefixRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.toURL[len("http://"):]
	return t.base.RoundTrip(req)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

