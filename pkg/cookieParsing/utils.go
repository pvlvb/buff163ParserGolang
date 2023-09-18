package cookieParsing

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TODO pass randomization ranges to config
func interItemSleepDelay(account *Account) {
	// Randomize delay factor between 1 and 1.5
	randomFactor := 1.0 + rand.Float64()*0.5
	delay := time.Duration(float64(account.InterItemDelay)*randomFactor) * time.Second
	time.Sleep(delay)
}

func fetchAccount() (*Account, int, []byte, error) {
	resp, err := http.Get("http://localhost/reserveAccount")
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}

	if resp.StatusCode == 404 {
		return nil, resp.StatusCode, bodyBytes, nil
	}

	account := &Account{}
	err = json.Unmarshal(bodyBytes, account)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}

	return account, resp.StatusCode, nil, nil
}

func fetchCookieParsingBuffIDs(jwtToken string) ([]string, error) {
	req, _ := http.NewRequest("GET", "http://localhost/cookieparsingitems", nil)
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

func resetBuff163Accounts(jwtToken string) (int, error) {
	req, err := http.NewRequest("GET", "http://localhost/resetaccounts", nil)
	if err != nil {
		return 0, fmt.Errorf("failed creating request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, errors.New(string(bodyBytes))
	}

	return resp.StatusCode, nil
}

func makeRequestWithProxy(proxyURLStr, cookieStr, userAgentStr, apiLink string) ([]byte, int, error) {
	parsedProxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		return nil, 0, fmt.Errorf("error parsing proxy URL: %v", err)
	}

	httpTransport := &http.Transport{}
	httpClient := &http.Client{Transport: httpTransport}

	switch strings.ToLower(parsedProxyURL.Scheme) {
	case "http", "https":
		httpTransport.Proxy = http.ProxyURL(parsedProxyURL)
	case "socks":
		auth := &proxy.Auth{
			User:     parsedProxyURL.User.Username(),
			Password: getPasswordFromURL(parsedProxyURL),
		}
		dialer, err := proxy.SOCKS5("tcp", parsedProxyURL.Host, auth, proxy.Direct)
		if err != nil {
			return nil, 0, fmt.Errorf("error creating SOCKS5 dialer: %v", err)
		}
		httpTransport.Dial = dialer.Dial
	default:
		return nil, 0, fmt.Errorf("unsupported proxy type: %s", parsedProxyURL.Scheme)
	}

	request, err := http.NewRequest("GET", apiLink, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating the request: %v", err)
	}
	request.Header.Set("Accept-Language", "en-US,en;q=0.9,ru;q=0.8")
	// Set Cookie and User-Agent
	request.Header.Set("Cookie", cookieStr)
	request.Header.Set("User-Agent", userAgentStr)

	// Make the request
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("error reading response body: %v", err)
	}

	return bodyBytes, response.StatusCode, nil
}

func getPasswordFromURL(u *url.URL) string {
	if password, set := u.User.Password(); set {
		return password
	}
	return ""
}
