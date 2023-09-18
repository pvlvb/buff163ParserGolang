package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

func FetchMissingBuffIDs(jwtToken string) ([]string, error) {
	req, _ := http.NewRequest("GET", "http://localhost/missingbuffids", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get missing buff IDs")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %v", err)
	}

	var ids []string
	if err := json.Unmarshal(bodyBytes, &ids); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return ids, nil
}

func fetchParsingProxies(jwtToken string) ([]string, error) {
	req, err := http.NewRequest("GET", "http://localhost/fetchParsingProxies", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get parsing proxies")
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %v", err)
	}

	var proxies []string
	if err := json.Unmarshal(bodyBytes, &proxies); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return proxies, nil
}
