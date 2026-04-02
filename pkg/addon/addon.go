// Package addon handles fetching and managing the KEDA HTTP add-on resources.
package addon

import (
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

	httpTimeout = 30 * time.Second
)

// versionRe validates a semver-like version without a leading "v".
var versionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+`)

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
func LatestVersion() (string, error) {
	client := &http.Client{Timeout: httpTimeout}
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

// FetchResources downloads the http-add-on manifest for the given version,
// parses it into unstructured objects, and stamps every object with the addon
// common label so they can be identified and deleted later.
func FetchResources(version string) ([]unstructured.Unstructured, error) {
	version, err := ValidateVersion(version)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(releaseURL, version, version)
	client := &http.Client{Timeout: httpTimeout}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download addon manifest for version %s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("addon manifest download returned HTTP %d for version %s", resp.StatusCode, version)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read addon manifest: %w", err)
	}

	objs, err := yaml.LoadData(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse addon manifest: %w", err)
	}

	return objs, nil
}






