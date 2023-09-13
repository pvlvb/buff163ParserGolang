package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Account struct {
	ID                       int    `json:"id"`
	Cookie                   string `json:"cookie"`
	SteamLinked              bool   `json:"steam_linked"`
	LastUsedAt               string `json:"last_used_at"`
	TotalReqsMade            int    `json:"total_reqs_made"`
	TotalRequestsMadePerHour int    `json:"total_requests_made_per_hour"`
	OuterItemDelay           int    `json:"outer_item_delay"`
	InterItemDelay           int    `json:"inter_item_delay"`
	IsLocked                 bool   `json:"is_locked"`
	LockedUntil              string `json:"locked_until"`
	Reqs429                  int    `json:"reqs_429"`
	BackoffCoeff             int    `json:"backoff_coeff"`
	Proxy                    string `json:"proxy"`
	UserAgent                string `json:"user_agent"`
}

type ErrorResponse struct {
	Message     string `json:"message"`
	WaitingTime int    `json:"waitingTime"`
}
type ProxyResponseData struct {
	Code string `json:"code"`
	Data struct {
		Items []struct {
			Price string `json:"price"`
		} `json:"items"`
	} `json:"data"`
}

type MissingBuffIDsResponse struct {
	IDs []string `json:"ids"`
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

func fetchMissingBuffIDs(jwtToken string) ([]string, error) {
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

func workerFunction(account *Account, goodsID string) {
	var successfulReqs, reqs429 int
	isBanned := false
	fmt.Println("Started processing item with goodsId", goodsID)
	defer func() {
		// Return the account to the backend once the work is completed
		data := map[string]interface{}{
			"account":         account,
			"successful_reqs": successfulReqs,
			"reqs_429":        reqs429,
			"is_banned":       isBanned,
		}
		fmt.Println(data)
		jsonData, _ := json.Marshal(data)
		http.Post("http://localhost/releaseaccount", "application/json", bytes.NewBuffer(jsonData))
		fmt.Println("Finished processing item with goodsId", goodsID)
	}()

	// Fetch the ProcessedItem from the backend
	resp, err := http.Get("http://localhost/items/" + goodsID) // Assuming the first ID for simplicity
	if err != nil {
		fmt.Println("Error fetching ProcessedItem:", err)
		return
	}
	defer resp.Body.Close()
	//TODO optimize this request so we will be updating other data without making non-cookie request
	_, initialStatusCode, err := makeRequestWithProxy(account.Proxy, account.Cookie, account.UserAgent, fmt.Sprintf("https://buff.163.com/goods/%s", goodsID))
	if err != nil {
		fmt.Printf("Error making initial request for goodsID %s: %s\n", goodsID, err)
		return
	}
	if initialStatusCode != http.StatusOK {
		fmt.Printf("Received unexpected status %d for initial request with goodsID %s\n", initialStatusCode, goodsID)
		return
	}

	// Wait for the inter_item request delay before proceeding
	randomFactor := 1.0 + rand.Float64()*0.5 // Random number between 1 and 1.5
	delay := time.Duration(float64(account.InterItemDelay)*randomFactor) * time.Second
	time.Sleep(delay)

	var item ProcessedItem
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&item); err != nil {
		fmt.Println("Error decoding ProcessedItem:", err)
		return
	}
	if len(item.FloatCategory) == 0 {
		return
	}

	maxCategories := 2
	if len(item.FloatCategory) < 2 {
		maxCategories = len(item.FloatCategory)
	}

	for idx, category := range item.FloatCategory[:maxCategories] {
		responseData, statusCode, err := makeRequestWithProxy(account.Proxy, account.Cookie, account.UserAgent, category.ApiLink)
		if err != nil {
			fmt.Printf("Error making request with proxy for category %s: %s\n", category.Name, err)
			return
		}

		// Check for response status codes
		switch {
		case statusCode == http.StatusOK:
			successfulReqs++
		case statusCode == http.StatusTooManyRequests:
			reqs429++
			return
		case statusCode == http.StatusForbidden: // Handling the 403 Forbidden status code
			fmt.Println("403 forbidden response ")
			// Pretty-printing the JSON
			if account != nil {
				fmt.Printf("for account ID: %d\n", account.ID)
			} else {
				fmt.Println("account is nil.")
			}
			isBanned = true
			return
		default:
			if account != nil {
				fmt.Printf("Received unexpected status %d for account ID: %d\n", statusCode, account.ID)
			} else {
				fmt.Printf("Received unexpected status %d, but account is nil\n", statusCode)
			}
			return
		}

		// Decode the response data
		var result ProxyResponseData
		if err := json.Unmarshal(responseData, &result); err != nil {
			fmt.Printf("Error decoding response data for category %s: %s\n", category.Name, err)
			return
		}

		// Check for "Action Forbidden" in the response code
		if result.Code == "Action Forbidden" {
			isBanned = true
			return
		}

		// Update the item's floatCategory price if there are items in the response
		if len(result.Data.Items) > 0 {
			price := result.Data.Items[0].Price
			item.FloatCategory[idx].Price = &price
		}

		// Delay between requests
		randomFactor := 1.0 + rand.Float64()*0.5 // Random number between 1 and 1.5
		delay := time.Duration(float64(account.InterItemDelay)*randomFactor) * time.Second
		time.Sleep(delay)
	}

	// Send the updated item back to localhost/items
	jsonItem, err := json.Marshal(item)
	if err != nil {
		fmt.Println("Error marshaling updated item:", err)
		return
	}

	resp, err = http.Post("http://localhost/items", "application/json", bytes.NewBuffer(jsonItem))
	if err != nil {
		fmt.Println("Error sending updated item to localhost/items:", err)
		return
	}
	defer resp.Body.Close()
}

var wg sync.WaitGroup

func main() {
	mode := "account_parsing"
	switch mode {
	case "proxy_parsing":
		//// Step 1: Authenticate and get the JWT token.
		//jwtToken, err := Authenticate()
		jwtToken, err1 := "", []int(nil)
		if err1 != nil {
			fmt.Println("Error authenticating:", err1)
			return
		}

		//Step 2: Fetch all missing buff IDs.
		allIDs, err := fetchMissingBuffIDs(jwtToken)
		if err != nil {
			fmt.Println("Error fetching missing buff IDs:", err)
			return
		}
		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		r.Shuffle(len(allIDs), func(i, j int) {
			allIDs[i], allIDs[j] = allIDs[j], allIDs[i]
		})
		//allIDs := []string{"42917"}
		fmt.Println("Total missing buff IDs fetched:", len(allIDs))

		// Determine the batch size (number of IDs processed in one round) based on available proxies.
		batchSize := len(proxies)
		fmt.Println("total amount of proxies:", len(proxies))
		// Step 3: Process the IDs in batches.
		for len(allIDs) > 0 {
			if len(allIDs) < batchSize {
				batchSize = len(allIDs)
			}

			// Process a batch of IDs.
			StartWorkers(jwtToken, allIDs[:batchSize])

			// Remove the processed IDs from the list.
			allIDs = allIDs[batchSize:]

			// Wait for a second before processing the next batch.
			time.Sleep(3 * time.Second)
			fmt.Println("Total missing buff IDs left:", len(allIDs))
		}
	case "account_parsing":
		// Step 1: Authenticate and get the JWT token.
		// jwtToken, err := Authenticate()
		fmt.Println("Account parsing started!")
		// Mocked data for demonstration purposes:
		buffIDs, err := fetchCookieParsingBuffIDs("jwtToken")
		if err != nil {
			fmt.Println("Error fetching missing buff IDs:", err)
			return
		}
		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		r.Shuffle(len(buffIDs), func(i, j int) {
			buffIDs[i], buffIDs[j] = buffIDs[j], buffIDs[i]
		})
		//allIDs := []string{"42917"}
		fmt.Println("Total missing buff IDs fetched:", len(buffIDs))

		for len(buffIDs) > 0 {
			//TODO Add reseting locks for accounts
			time.Sleep(100 * time.Millisecond)
			account, statusCode, responseBody, err := fetchAccount()
			if err != nil {
				fmt.Println("Error fetching account:", err)
				continue
			}

			if statusCode == 200 {
				//fmt.Println("Account with id is working", account.ID)
				wg.Add(1)
				go workerFunction(account, buffIDs[0])
				buffIDs = buffIDs[1:]
			} else if statusCode == 404 {
				errorResponse := &ErrorResponse{}
				err = json.Unmarshal(responseBody, errorResponse)
				if err != nil {
					//fmt.Println("Error decoding error response:", err)
					continue
				}
				fmt.Println(errorResponse.Message)
				fmt.Printf("Waiting for %d seconds for new accounts...\n", errorResponse.WaitingTime)
				time.Sleep(time.Duration(errorResponse.WaitingTime) * time.Second)

			} else {
				fmt.Println("Unexpected status code:", statusCode)
			}
		}
		wg.Wait()

	default:
		fmt.Println("Unknown mode.")
	}

}
