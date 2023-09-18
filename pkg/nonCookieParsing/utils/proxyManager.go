package utils

import (
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
)

var proxies []string
var proxyIndex int32 = -1

func InitProxies(jwtToken string) (int, error) {
	var err error
	proxies, err = fetchParsingProxies(jwtToken)
	if err != nil {
		return 0, err
	}

	if len(proxies) == 0 {
		return 0, errors.New("no proxies returned from fetch")
	}

	// Reset index
	proxyIndex = -1

	return len(proxies), nil
}
func GetNextProxy() (*url.URL, error) {
	nextIndex := atomic.AddInt32(&proxyIndex, 1) % int32(len(proxies))
	return url.Parse(proxies[nextIndex])
}

func GetHttpClientWithProxy() (*http.Client, error) {
	proxyURL, err := GetNextProxy()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	return &http.Client{
		Transport: transport,
	}, nil
}
