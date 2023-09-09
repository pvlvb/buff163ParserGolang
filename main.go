package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type MissingBuffIDsResponse struct {
	IDs []string `json:"ids"`
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

func main() {
	// Step 1: Authenticate and get the JWT token.
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
}
