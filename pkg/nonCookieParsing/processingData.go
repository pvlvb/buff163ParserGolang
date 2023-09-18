package nonCookieParsing

import (
	"fmt"
	"strconv"
)

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

func pointerToString(s string) *string {
	return &s
}

func pointerToInt(i int) *int {
	return &i
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

func transformData(id string, responseData map[string]interface{}) *ProcessedItem {
	// Extracting data field from response
	data, ok := responseData["data"].(map[string]interface{})
	if !ok {
		nonCookieParsingLogger.Error("Error processing response data.")
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
