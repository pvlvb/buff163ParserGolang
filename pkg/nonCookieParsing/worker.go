package nonCookieParsing

import (
	"buff163Parser/pkg/logger"
	"buff163Parser/pkg/nonCookieParsing/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// TODO put this logger in other packages
var nonCookieParsingLogger = logger.Log.WithField("context", "nonCookieParsing")

func processAndSendItem(jwtToken, id string) {
	// Use proxy to make a request to the third-party API
	clientWithProxy, err := utils.GetHttpClientWithProxy()
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error getting HTTP client with proxy")
		return
	}

	thirdPartyURL := fmt.Sprintf("https://buff.163.com/api/market/goods/info?goods_id=%s&game=csgo", id)

	// Create a new request
	req1, err := http.NewRequest("GET", thirdPartyURL, nil)
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error creating new request")
		return
	}

	// Set the Accept-Language header for the request
	req1.Header.Set("Accept-Language", "en-US,en;q=0.9,ru;q=0.8")

	// Use clientWithProxy to execute the request
	resp, err := clientWithProxy.Do(req1)
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error making request to third-party API")
		return
	}
	defer resp.Body.Close()

	// Here you can process the response from the third-party API if needed
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error reading response body")
		return
	}

	// Assuming the response structure corresponds to the JSON you provided
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error unmarshalling response")
		return
	}
	if code, exists := responseData["code"].(string); !exists || code != "OK" {
		nonCookieParsingLogger.Error("Invalid response or response code is not 'OK'")
		return
	}

	// Here you can process the response from the third-party API
	_, ok := responseData["data"].(map[string]interface{})
	if !ok {
		nonCookieParsingLogger.Error("Error processing response data")
		return
	}
	transformedItem := transformData(id, responseData)
	formattedData, err := json.MarshalIndent(transformedItem, "", "  ")
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error marshaling transformed item for printing")
		return
	}

	// Send the processed ID to the localhost server without using a proxy.
	client := &http.Client{} // This client doesn't use a proxy

	req, err := http.NewRequest("POST", "http://localhost/items", bytes.NewBuffer(formattedData))
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error creating request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err = client.Do(req)
	if err != nil {
		nonCookieParsingLogger.WithError(err).Error("Error sending processed item")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		nonCookieParsingLogger.Errorf("Received non-OK response: %s", resp.Status)
	}
}

func StartNonCookieParsing() error {
	// Initializing nonCookieParsingLogger
	nonCookieParsingLogger.Info("nonCookie parsing started!")
	//// Step 1: Authenticate and get the JWT token.
	//jwtToken, err := Authenticate()
	jwtToken, err1 := "", []string(nil)
	if err1 != nil {
		//nonCookieParsingLogger.WithError(err1).Errorf('Authentication with backend failed')
		return fmt.Errorf("error authenticating %s", err1)
	}
	const N = 100 //TODO move to config. Number of items to notify if they were parsed
	processedCount := 0

	for {
		//Step 2: Fetch all missing buff IDs.
		allIDs, err := utils.FetchMissingBuffIDs(jwtToken)
		if err != nil {
			nonCookieParsingLogger.WithError(err).Errorf("Fetching of missing buffIds failed")
			return fmt.Errorf("error fetching missing buff IDs %s", err)
		}

		if len(allIDs) == 0 {
			nonCookieParsingLogger.Info("No missing buff IDs found. Sleeping for 10 minutes.")
			time.Sleep(10 * time.Minute)
			continue
		}

		src := rand.NewSource(time.Now().UnixNano())
		r := rand.New(src)
		r.Shuffle(len(allIDs), func(i, j int) {
			allIDs[i], allIDs[j] = allIDs[j], allIDs[i]
		})
		//allIDs := []string{"42917"}
		nonCookieParsingLogger.Infof("Total missing buff IDs fetched %d", len(allIDs))

		//Step 3: initializing proxies. Fetching them from backend and setting counter of usage
		numberOfProxies, err := utils.InitProxies(jwtToken)
		if err != nil {
			nonCookieParsingLogger.WithError(err).Errorf("Error fetching parsing proxies")
			return fmt.Errorf("error fetching parsing proxies: %s", err)
		}

		nonCookieParsingLogger.Infof("Total proxies fetched %d", numberOfProxies)

		// Step 3: Process the IDs in batches.
		for len(allIDs) > 0 {
			if len(allIDs) < numberOfProxies {
				numberOfProxies = len(allIDs)
			}

			// Process a batch of IDs.
			workerFunction(jwtToken, allIDs[:numberOfProxies])
			// Remove the processed IDs from the list.
			allIDs = allIDs[numberOfProxies:]
			processedCount += len(allIDs[:numberOfProxies])
			if processedCount >= N {
				nonCookieParsingLogger.Infof("%d buffIds have been processed, %d left", processedCount, len(allIDs))
				processedCount = 0 // Reset the counter
			}
			//TODO pass to config
			time.Sleep(3 * time.Second)
		}
	}
	return nil
}

func workerFunction(jwtToken string, ids []string) {
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
