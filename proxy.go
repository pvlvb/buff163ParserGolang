package main

import (
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
)

// Proxies list.
var proxies = []string{
	"http://AKQY21288:JMSU1459@149.51.5.33:5878",
	"http://AKQY21288:JMSU1459@149.51.4.237:6291",
	"http://AKQY21288:JMSU1459@149.51.2.203:5872",
	"http://AKQY21288:JMSU1459@149.51.6.111:5837",
	"http://AKQY21288:JMSU1459@149.51.1.8:5427",
	"http://AKQY21288:JMSU1459@149.51.7.212:5615",
	"http://AKQY21288:JMSU1459@149.51.2.197:5041",
	"http://AKQY21288:JMSU1459@149.51.7.222:5991",
	"http://AKQY21288:JMSU1459@149.51.3.42:6732",
	"http://AKQY21288:JMSU1459@149.51.7.166:6061",
	// ... other proxy addresses
}

var proxyIndex int32 = -1

// getNextProxy rotates through the proxies in a round-robin fashion.
func getNextProxy() (*url.URL, error) {
	// Increment the proxyIndex and get the next proxy
	nextIndex := atomic.AddInt32(&proxyIndex, 1) % int32(len(proxies))

	// Ensure the nextIndex is within bounds
	if nextIndex >= int32(len(proxies)) {
		return nil, errors.New("ran out of proxies")
	}

	// Parse the proxy URL
	return url.Parse(proxies[nextIndex])
}

// getHttpClientWithProxy gets an HTTP client that uses a proxy.
func getHttpClientWithProxy() (*http.Client, error) {
	proxyURL, err := getNextProxy()
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
