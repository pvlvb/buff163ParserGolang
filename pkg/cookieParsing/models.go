package cookieParsing

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
	Range          []string `json:"range,omitempty"`
	Price          *string  `json:"price"`
	ApiLink        string   `json:"apiLink"`
	Name           *string  `json:"name,omitempty"`
	Value          *string  `json:"value,omitempty"`
	Parameter      *string  `json:"parameter,omitempty"`
	ListingsPrices []string `json:"listingsprices,omitempty"`
}

type BackendWaitingTimeResponse struct {
	Message     string `json:"message"`
	WaitingTime int    `json:"waitingTime"`
}

type Buff163SellOrdersResponse struct {
	Code string `json:"code"`
	Data struct {
		Items []struct {
			Price string `json:"price"`
		} `json:"items"`
	} `json:"data"`
}

type PriceHistoryData struct {
	Currency           string      `json:"currency"`
	CurrencySymbol     string      `json:"currency_symbol"`
	Days               int         `json:"days"`
	PriceHistory       [][]float64 `json:"price_history"`
	PriceType          string      `json:"price_type"`
	SteamPriceCurrency string      `json:"steam_price_currency"`
}

type PriceHistoryResponse struct {
	Code string `json:"code"`
	Data struct {
		Currency           string      `json:"currency"`
		CurrencySymbol     string      `json:"currency_symbol"`
		Days               int         `json:"days"`
		PriceHistory       [][]float64 `json:"price_history"`
		PriceType          string      `json:"price_type"`
		SteamPriceCurrency string      `json:"steam_price_currency"`
	} `json:"data"`
	Msg interface{} `json:"msg"`
}
type ResultData struct {
	GoodsID      string      `json:"goodsid"`
	PriceHistory [][]float64 `json:"price_history"`
}

type SaleRecordsApiResponse struct {
	Code string `json:"code"`
	Data struct {
		Items []struct {
			AssetInfo struct {
				Info struct {
					Stickers []struct {
						Category  string  `json:"category"`
						ImgURL    string  `json:"img_url"`
						Name      string  `json:"name"`
						Slot      int     `json:"slot"`
						StickerID int     `json:"sticker_id"`
						Wear      float64 `json:"wear"`
					} `json:"stickers"`
				} `json:"info"`
				GoodsID   int    `json:"goods_id"`
				Paintwear string `json:"paintwear"`
				ID        string `json:"id"`
			} `json:"asset_info"`
			Price        string `json:"price"`
			SellerID     string `json:"seller_id"`
			TransactTime int64  `json:"transact_time"`
		} `json:"items"`
	} `json:"data"`
}

type ProcessedSaleRecord struct {
	Stickers []struct {
		Category  string  `json:"category"`
		ImgURL    string  `json:"img_url"`
		Name      string  `json:"name"`
		Slot      int     `json:"slot"`
		StickerID int     `json:"sticker_id"`
		Wear      float64 `json:"wear"`
	} `json:"stickers"`
	Price    string `json:"price"`
	GoodsID  int    `json:"goodsid"`
	SaleID   string `json:"sale_id"`
	Date     int64  `json:"date"`
	Float    string `json:"floatvalue"`
	SellerID string `json:"seller_id"`
}
