package nonCookieParsing

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

type MissingBuffIDsResponse struct {
	IDs []string `json:"ids"`
}
