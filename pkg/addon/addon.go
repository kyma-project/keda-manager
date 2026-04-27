// Package addon handles fetching and managing the KEDA HTTP add-on resources.
package addon

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/kyma-project/keda-manager/pkg/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	tagsURL    = "https://api.github.com/repos/kedacore/http-add-on/tags"
	releaseURL = "https://github.com/kedacore/http-add-on/releases/download/v%s/keda-add-ons-http-%s.yaml"
	crdURL     = "https://github.com/kedacore/http-add-on/releases/download/v%s/keda-add-ons-http-%s-crds.yaml"

	httpTimeout = 30 * time.Second
)

// versionRe validates a semver-like version without a leading "v".
var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+`)

// NewHTTPClient returns an *http.Client that explicitly loads system CAs so TLS works in minimal (distroless) containers.
func NewHTTPClient() (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system CA pool: %w", err)
	}
	return &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
	}, nil
}

// ValidateVersion trims a leading "v"/"V" and returns an error if the result
// is not a valid semver string.
func ValidateVersion(version string) (string, error) {
	version = strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V")
	if !versionRe.MatchString(version) {
		return "", fmt.Errorf("addon version %q is not a valid semver (expected format: MAJOR.MINOR.PATCH)", version)
	}
	return version, nil
}

// LatestVersion queries the GitHub tags API and returns the latest tag name
// with the leading "v" stripped.
func LatestVersion(client *http.Client) (string, error) {
	resp, err := client.Get(tagsURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tags API returned HTTP %d", resp.StatusCode)
	}

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return "", fmt.Errorf("failed to decode tags response: %w", err)
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("no tags found for http-add-on")
	}

	tag := strings.TrimPrefix(tags[0].Name, "v")
	return tag, nil
}

// FetchResources downloads the http-add-on CRDs and manifest for the given
// version and parses them into unstructured objects. The CRDs are prepended so
// they are applied before the rest of the resources.
func FetchResources(client *http.Client, version string) ([]unstructured.Unstructured, error) {
	version, err := ValidateVersion(version)
	if err != nil {
		return nil, err
	}

	crdObjs, err := fetchURL(client, fmt.Sprintf(crdURL, version, version), version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch addon CRDs: %w", err)
	}

	objs, err := fetchURL(client, fmt.Sprintf(releaseURL, version, version), version)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch addon manifest: %w", err)
	}

	return append(crdObjs, objs...), nil
}

// fetchURL downloads a single URL and parses it into unstructured objects.
func fetchURL(client *http.Client, url, version string) ([]unstructured.Unstructured, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download of %s returned HTTP %d for version %s", url, resp.StatusCode, version)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from %s: %w", url, err)
	}

	objs, err := yaml.LoadData(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse response from %s: %w", url, err)
	}

	return objs, nil
}
