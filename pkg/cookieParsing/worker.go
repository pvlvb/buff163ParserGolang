package cookieParsing

import (
	"buff163Parser/pkg/logger"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var cookieParsingLogger = logger.Log.WithField("context", "cookieParsing")

// TODO implement counting of goroutines(?)
var wg sync.WaitGroup

func StartCookieParsing() error {
	// Step 1: Authenticate and get the JWT token.
	// jwtToken, err := Authenticate()
	const N = 10 // Change this to your desired logging interval
	parsedCount := 0
	cookieParsingLogger.Info("Cookie parsing started!")

	_, err := resetBuff163Accounts("jwtToken")
	if err != nil {
		cookieParsingLogger.Error("Error resetting buff163 accounts:", err)
		return fmt.Errorf("error resetting buff163 accounts %s", err)
	} else {
		cookieParsingLogger.Info("Accounts reset successfully")
	}

	buffIDs, err := fetchCookieParsingBuffIDs("jwtToken")
	if err != nil {
		cookieParsingLogger.Error("Error fetching missing buff IDs:", err)
		return fmt.Errorf("error fetching missing buff IDs %s", err)
	}
	//var buffIDs []string = []string{"35213"}
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)
	r.Shuffle(len(buffIDs), func(i, j int) {
		buffIDs[i], buffIDs[j] = buffIDs[j], buffIDs[i]
	})

	cookieParsingLogger.Infof("Total missing buff IDs fetched: %d", len(buffIDs))

	for {
		if len(buffIDs) == 0 {
			cookieParsingLogger.Info("No more buff IDs to process. Waiting for 10 minutes...")
			time.Sleep(10 * time.Minute)
			buffIDs, err = fetchCookieParsingBuffIDs("jwtToken")
			if err != nil {
				cookieParsingLogger.Error("Error fetching missing buff IDs after waiting:", err)
			}
			//TODO check this refetching logic, seems sketchy
			if len(buffIDs) != 0 {
				src := rand.NewSource(time.Now().UnixNano())
				r := rand.New(src)
				r.Shuffle(len(buffIDs), func(i, j int) {
					buffIDs[i], buffIDs[j] = buffIDs[j], buffIDs[i]
				})
			}
			continue
		}

		// TODO Add resetting locks for accounts
		time.Sleep(100 * time.Millisecond)
		account, statusCode, responseBody, err := fetchAccount()
		if err != nil {
			cookieParsingLogger.Error("Error fetching account:", err)
			continue
		}

		if statusCode == 200 {
			wg.Add(1)
			go workerFunction(account, buffIDs[0])
			buffIDs = buffIDs[1:]
			parsedCount++
			if parsedCount%N == 0 {
				cookieParsingLogger.Infof("%d buffIds have been processed, %d left", parsedCount, len(buffIDs))
			}
		} else if statusCode == 404 {
			errorResponse := &BackendWaitingTimeResponse{}
			err = json.Unmarshal(responseBody, errorResponse)
			if err != nil {
				cookieParsingLogger.Error("Error decoding error response:", err)
				continue
			}
			cookieParsingLogger.Info(errorResponse.Message)
			cookieParsingLogger.Infof("Waiting for %d seconds for new accounts...", errorResponse.WaitingTime)
			time.Sleep(time.Duration(errorResponse.WaitingTime) * time.Second)
		} else {
			cookieParsingLogger.Error("Unexpected status code:", statusCode)
			wg.Wait()
			return nil
		}
	}
}

func workerFunction(account *Account, goodsID string) {
	var accountCookieParsingLogger = cookieParsingLogger.WithFields(logrus.Fields{"account": account.ID, "goodsId": goodsID})
	defer wg.Done() //ensure that the goroutine is closed after executing all of this stuff
	var successfulReqs, reqs429 int
	isBanned := false
	defer func() {
		// Return the account to the backend once the work is completed
		data := map[string]interface{}{
			"account":         account,
			"successful_reqs": successfulReqs,
			"reqs_429":        reqs429,
			"is_banned":       isBanned,
		}
		accountCookieParsingLogger.Debug(data)
		jsonData, _ := json.Marshal(data)
		_, err := http.Post("http://localhost/releaseaccount", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			accountCookieParsingLogger.WithError(err).Error("error releasing account")
			return
		}
		accountCookieParsingLogger.Debugf("Finished processing item with goodsId %s", goodsID)
	}()

	// Fetch the ProcessedItem from the backend
	resp, err := http.Get("http://localhost/items/" + goodsID) // Assuming the first ID for simplicity
	if err != nil {
		accountCookieParsingLogger.WithError(err).Errorf("Error fetching itemData from backend, goodsId %s", goodsID)
		return
	}
	defer resp.Body.Close()
	//TODO optimize this request so we will be updating other data without making non-cookie request
	//TODO antisybil request
	_, initialStatusCode, err := makeRequestWithProxy(account.Proxy, account.Cookie, account.UserAgent, fmt.Sprintf("https://buff.163.com/goods/%s", goodsID))
	if err != nil {
		accountCookieParsingLogger.WithError(err).Errorf("Error making initial request")
		return
	}
	switch {
	case initialStatusCode == http.StatusOK:
		successfulReqs++
		interItemSleepDelay(account)
	case initialStatusCode == http.StatusTooManyRequests:
		reqs429++
		accountCookieParsingLogger.Errorf("Account got %d code with initial request", initialStatusCode)
		return
	default:
		accountCookieParsingLogger.Errorf("Received unexpected status %d for initial request", initialStatusCode)
		return
	}

	var item ProcessedItem
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&item); err != nil {
		accountCookieParsingLogger.WithError(err).Errorf("Error decoding ProcessedItem")
		return
	}

	//TODO pass it to config(prob)
	maxCategories := 6
	if len(item.FloatCategory) < 6 {
		maxCategories = len(item.FloatCategory)
	}

	for idx, category := range item.FloatCategory[:maxCategories] {
		responseData, statusCode, err := makeRequestWithProxy(account.Proxy, account.Cookie, account.UserAgent, category.ApiLink)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error making request with proxy for category %s\n", category.ApiLink)
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
			accountCookieParsingLogger.Errorf("403 forbidden response, now this account banned")
			// Pretty-printing the JSON
			isBanned = true
			return
		default:
			accountCookieParsingLogger.Errorf("Received unexpected status %d", statusCode)
			return
		}

		// Decode the response data
		var result Buff163SellOrdersResponse
		if err := json.Unmarshal(responseData, &result); err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error decoding response data for category %s", *category.Name)
			return
		}

		// Check for "Action Forbidden" in the response code
		if result.Code == "Action Forbidden" {
			accountCookieParsingLogger.Errorf("Error with categories request, response data: %s", string(responseData))
			isBanned = true
			return
		}
		if result.Code != "OK" {
			accountCookieParsingLogger.Errorf("Error with categories request, response data: %s", string(responseData))
			isBanned = true
			return
		}
		var fvRangePrices []string
		// Update the item's floatCategory price if there are items in the response
		if len(result.Data.Items) > 0 {
			for _, rangeItem := range result.Data.Items {
				fvRangePrices = append(fvRangePrices, rangeItem.Price)
			}
			item.FloatCategory[idx].ListingsPrices = fvRangePrices
			price := result.Data.Items[0].Price
			if &price == nil {
				accountCookieParsingLogger.Errorf("Price wasn't updated for goodsId %s with code %d. \n Response data: %s\n", goodsID, statusCode, responseData)
			}
			item.FloatCategory[idx].Price = &price
		}

		// Delay between requests
		interItemSleepDelay(account)
	}
	//Fetching price history(graph) from buff163
	if account.SteamLinked {
		//fetching graph
		priceHistoryApiLink := fmt.Sprintf("https://buff.163.com/api/market/goods/price_history/buff?game=csgo&goods_id=%s&currency=USD&days=7&buff_price_type=2&with_sell_num=true", item.GoodsID)
		responseData, statusCode, err := makeRequestWithProxy(account.Proxy, account.Cookie, account.UserAgent, priceHistoryApiLink)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error making request with proxy for price history with account Id %d", account.ID)
			return
		}
		switch {
		case statusCode == http.StatusOK:
			successfulReqs++
		case statusCode == http.StatusTooManyRequests:
			reqs429++
			return
		case statusCode == http.StatusForbidden: // Handling the 403 Forbidden status code
			accountCookieParsingLogger.Errorf("403 forbidden response, now this account banned")
			isBanned = true
			return
		default:
			accountCookieParsingLogger.Errorf("Received unexpected status %d", statusCode)
			return
		}

		var priceHistoryResponse PriceHistoryResponse
		err = json.Unmarshal(responseData, &priceHistoryResponse)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error unmarshalling response data for goodsId %s\n", goodsID)
			return
		}
		processedPriceHistory := ResultData{
			GoodsID:      item.GoodsID,
			PriceHistory: priceHistoryResponse.Data.PriceHistory,
		}

		priceHistoryJSON, err := json.Marshal(processedPriceHistory)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error marshalling price history to JSON")
			return
		}

		resp, err = http.Post("http://localhost/historicalprices", "application/json", bytes.NewBuffer(priceHistoryJSON))
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error sending updated item to localhost/historicalprices")
			return
		}
		defer resp.Body.Close()
	}

	interItemSleepDelay(account)
	//Fetching sales from buff163
	if account.SteamLinked {
		//fetching graph
		salesRecordsApiLink := fmt.Sprintf("https://buff.163.com/api/market/goods/bill_order?game=csgo&goods_id=%s", item.GoodsID)
		responseData, statusCode, err := makeRequestWithProxy(account.Proxy, account.Cookie, account.UserAgent, salesRecordsApiLink)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error making request with proxy for sale records with account Id %d", account.ID)
			return
		}
		switch {
		case statusCode == http.StatusOK:
			successfulReqs++
		case statusCode == http.StatusTooManyRequests:
			reqs429++
			return
		case statusCode == http.StatusForbidden: // Handling the 403 Forbidden status code
			accountCookieParsingLogger.Errorf("403 forbidden response, now this account banned")
			isBanned = true
			return
		default:
			accountCookieParsingLogger.Errorf("Received unexpected status %d", statusCode)
			return
		}

		var saleRecordsResponsense SaleRecordsApiResponse
		err = json.Unmarshal(responseData, &saleRecordsResponsense)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error unmarshalling sale records response data")
			return
		}
		var processedSaleRecords []ProcessedSaleRecord
		for _, saleRecord := range saleRecordsResponsense.Data.Items {
			pItem := ProcessedSaleRecord{
				Stickers: saleRecord.AssetInfo.Info.Stickers,
				Price:    saleRecord.Price,
				GoodsID:  saleRecord.AssetInfo.GoodsID,
				SaleID:   saleRecord.AssetInfo.ID,
				Date:     saleRecord.TransactTime,
				Float:    saleRecord.AssetInfo.Paintwear,
				SellerID: saleRecord.SellerID,
			}

			processedSaleRecords = append(processedSaleRecords, pItem)
		}

		saleRecordsJson, err := json.Marshal(processedSaleRecords)
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error marshalling sale records to JSON")
			return
		}
		accountCookieParsingLogger.Debug("Sale records were processed")
		resp, err = http.Post("http://localhost/sales", "application/json", bytes.NewBuffer(saleRecordsJson))
		if err != nil {
			accountCookieParsingLogger.WithError(err).Errorf("Error sending updated item to localhost/sales")
			return
		}
		defer resp.Body.Close()
	}

	// Send the updated item back to localhost/items
	jsonItem, err := json.Marshal(item)
	if err != nil {
		accountCookieParsingLogger.WithError(err).Errorf("Error marshaling updated item")
		return
	}

	resp, err = http.Post("http://localhost/items", "application/json", bytes.NewBuffer(jsonItem))
	if err != nil {
		accountCookieParsingLogger.WithError(err).Errorf("Error sending updated item to localhost/items")
		return
	}
	defer resp.Body.Close()
}
