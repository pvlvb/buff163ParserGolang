package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

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

// Processed item structure
type ProcessedItem struct {
	GoodsID         string     `json:"goodsid"`
	MarketHashName  string     `json:"markethashname"`
	ListingPrice    string     `json:"listingprice"`
	Listings        int        `json:"listings"`
	BuyOrders       int        `json:"buyorders"`
	BuyOrderPrice   string     `json:"buyorderprice"`
	SteamMarketLink string     `json:"steammarketlink"`
	FadeCategory    []Category `json:"fadecategory"`
	StyleCategory   []Category `json:"stylecategory"`
	FloatCategory   []Category `json:"floatcategory"`
}

type Category struct {
	Range     []string `json:"range,omitempty"`
	Price     *string  `json:"price"`
	ApiLink   string   `json:"apiLink"`
	Name      *string  `json:"name,omitempty"`
	Value     *string  `json:"value,omitempty"`
	Parameter *string  `json:"parameter,omitempty"`
}

func convertToString(val interface{}) string {
	switch v := val.(type) {
	case float64:
		return fmt.Sprintf("%.0f", v)
	case string:
		return v
	default:
		return "" // or return some default value
	}
}

func extractCategories(data map[string]interface{}, apiUrl string) (floatCategory, fadeCategory, styleCategory, paintSeedCategory []Category) {

	// Extracting paintwear_choices
	if paintwearChoices, ok := data["paintwear_choices"].([]interface{}); ok {
		for _, item := range paintwearChoices {
			if choice, ok := item.([]interface{}); ok && len(choice) == 2 {
				cat := Category{
					Range:   []string{choice[0].(string), choice[1].(string)},
					ApiLink: apiUrl + "&min_paintwear=" + choice[0].(string) + "&max_paintwear=" + choice[1].(string),
				}
				floatCategory = append(floatCategory, cat)
			}
		}
	}

	// Extracting fade_choices
	if data["has_fade_name"] == true {
		if fadeChoices, ok := data["fade_choices"].([]interface{}); ok {
			for _, item := range fadeChoices {
				if choice, ok := item.([]interface{}); ok && len(choice) == 2 {
					cat := Category{
						Range:   []string{choice[0].(string), choice[1].(string)},
						ApiLink: apiUrl + "&min_fade=" + choice[0].(string) + "&max_fade=" + choice[1].(string),
					}
					fadeCategory = append(fadeCategory, cat)
				}
			}
		}
	}

	// Extracting asset_tags
	if assetTags, ok := data["asset_tags"].([]interface{}); ok && len(assetTags) > 0 {
		if tags, ok := assetTags[0].(map[string]interface{}); ok {
			if items, ok := tags["items"].([]interface{}); ok {
				for _, item := range items {
					if tagItem, ok := item.(map[string]interface{}); ok {
						cat := Category{
							Name:      pointerToString(tagItem["name"].(string)),
							Value:     pointerToString(fmt.Sprintf("%.0f", tagItem["id"].(float64))),
							Parameter: pointerToString("tag_ids"),
							ApiLink:   apiUrl + "&tag_ids=" + strconv.Itoa(int(tagItem["id"].(float64))),
						}
						styleCategory = append(styleCategory, cat)
					}
				}
			}
		}
	}

	// Extracting paintseed_filters
	if paintseedFilters, ok := data["paintseed_filters"].([]interface{}); ok {
		for _, filter := range paintseedFilters {
			if filterMap, ok := filter.(map[string]interface{}); ok {
				if filterMap["type"] != "paintseed" && filterMap["items"] != nil {
					items := filterMap["items"].([]interface{})
					for _, item := range items {
						if itemMap, ok := item.(map[string]interface{}); ok {
							cat := Category{
								Name:      pointerToString(convertToString(itemMap["name"])),
								Value:     pointerToString(convertToString(itemMap["value"])),
								Parameter: pointerToString(filterMap["type"].(string)),
								ApiLink:   apiUrl + "&" + convertToString(filterMap["type"]) + "=" + convertToString(itemMap["value"]),
							}
							paintSeedCategory = append(paintSeedCategory, cat)
						}
					}
				}
			}
		}
	}

	return
}

func pointerToString(s string) *string {
	return &s
}

func pointerToInt(i int) *int {
	return &i
}

func transformData(id string, responseData map[string]interface{}) *ProcessedItem {
	// Extracting data field from response
	data, ok := responseData["data"].(map[string]interface{})
	if !ok {
		fmt.Println("Error processing response data.")
		return nil
	}

	apiUrl := fmt.Sprintf("https://buff.163.com/api/market/goods/sell_order?game=csgo&goods_id=%s&page_num=1&sort_by=default&mode=&allow_tradable_cooldown=1", id) // You might need to define your API URL here

	// Extracting categories
	floatCategory, fadeCategory, styleCategory, paintSeedCategory := extractCategories(data, apiUrl)

	// Extracting and transforming required fields
	item := &ProcessedItem{
		GoodsID:         id,
		MarketHashName:  data["market_hash_name"].(string),
		ListingPrice:    data["sell_min_price"].(string),
		Listings:        int(data["sell_num"].(float64)), // JSON numbers are float64 by default in Go
		BuyOrders:       int(data["buy_num"].(float64)),
		BuyOrderPrice:   data["buy_max_price"].(string),
		SteamMarketLink: data["steam_market_url"].(string),
		FadeCategory:    fadeCategory,
		StyleCategory:   append(styleCategory, paintSeedCategory...), // Merging two slices here
		FloatCategory:   floatCategory,
	}

	return item
}

// processAndSendItem processes the provided ID (mocked for now) and sends it to the endpoint.
func processAndSendItem(jwtToken, id string) {
	// Use proxy to make a request to the third-party API
	clientWithProxy, err := getHttpClientWithProxy()
	if err != nil {
		fmt.Println("Error getting HTTP client with proxy:", err)
		return
	}

	thirdPartyURL := fmt.Sprintf("https://buff.163.com/api/market/goods/info?goods_id=%s&game=csgo", id)

	// Create a new request
	req1, err := http.NewRequest("GET", thirdPartyURL, nil)
	if err != nil {
		fmt.Println("Error creating new request:", err)
		return
	}

	// Set the Accept-Language header for the request
	req1.Header.Set("Accept-Language", "en-US,en;q=0.9,ru;q=0.8")

	// Use clientWithProxy to execute the request
	resp, err := clientWithProxy.Do(req1)
	if err != nil {
		fmt.Println("Error making request to third-party API:", err)
		return
	}
	defer resp.Body.Close()

	// Here you can process the response from the third-party API if needed
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Assuming the response structure corresponds to the JSON you provided
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		fmt.Println("Error unmarshalling response:", err)
		return
	}
	//fmt.Println(responseData["data"])
	if code, exists := responseData["code"].(string); !exists || code != "OK" {
		fmt.Println("Invalid response or response code is not 'OK'.")
		return
	}

	// Here you can process the response from the third-party API
	_, ok := responseData["data"].(map[string]interface{})
	if !ok {
		fmt.Println("Error processing response data.")
		return
	}
	transformedItem := transformData(id, responseData)
	formattedData, err := json.MarshalIndent(transformedItem, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling transformed item for printing:", err)
		return
	}

	//fmt.Println(string(formattedData))

	// Mocked: Process the ID (no real processing for now, just a print statement).
	//fmt.Println("Processed ID:", id)

	// Send the processed ID to the localhost server without using a proxy.
	client := &http.Client{} // This client doesn't use a proxy

	//item := map[string]string{"id": id}
	//data1, err := json.Marshal(item)
	//if err != nil {
	//	fmt.Println("Error marshaling item:", err)
	//	return
	//}

	req, err := http.NewRequest("POST", "http://localhost/items", bytes.NewBuffer(formattedData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("Error sending processed item:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Received non-OK response:", resp.Status)
	}
}

// StartWorkers initiates the goroutines to process a batch of IDs.
func StartWorkers(jwtToken string, ids []string) {
	var wg sync.WaitGroup

	// For each ID in the current batch, launch a separate goroutine.
	for i := 0; i < len(ids); i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			processAndSendItem(jwtToken, id)
		}(ids[i])
	}

	// Wait for all goroutines to complete.
	wg.Wait()
}
