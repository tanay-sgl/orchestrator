package main

import (
	"reflect"
	"time"
)

func GetTableStruct(tableName string) reflect.Type {
	tableMap := map[string]reflect.Type{
		"collection":         reflect.TypeOf(Collection{}),
		"collection_dynamic": reflect.TypeOf(CollectionDynamic{}),
		"nft_listings":       reflect.TypeOf(NFTListing{}),
		"nft_ownership":      reflect.TypeOf(NFTOwnership{}),
		"nft_offers":         reflect.TypeOf(NFTOffer{}),
		"erc20_transfers":    reflect.TypeOf(ERC20Transfer{}),
		"fee":                reflect.TypeOf(Fee{}),
		"nft":                reflect.TypeOf(NFT{}),
		"contract":           reflect.TypeOf(Contract{}),
		"payment_tokens":     reflect.TypeOf(PaymentToken{}),
		"nft_dynamic":        reflect.TypeOf(NFTDynamic{}),
		"token_price":        reflect.TypeOf(TokenPrice{}),
		"nft_events":         reflect.TypeOf(NFTEvent{}),
		"documents":          reflect.TypeOf(Document{}),
	}
	return tableMap[tableName]
}

// All table names except document
var TableNames = []string{
	"collection",
	"collection_dynamic",
	"nft_listings",
	"nft_ownership",
	"nft_offers",
	"erc20_transfers",
	"fee",
	"nft",
	"contract",
	"payment_tokens",
	"nft_dynamic",
	"token_price",
	"nft_events",
}

type Collection struct {
	tableName    struct{}  `pg:"collection"`
	OpenSeaSlug  string    `pg:"opensea_slug,pk"`
	GameName     string    `pg:"game_name"`
	GameID       string    `pg:"game_id"`
	Name         string    `pg:"name"`
	Description  string    `pg:"description"`
	Owner        string    `pg:"owner"`
	Category     string    `pg:"category"`
	IsNSFW       bool      `pg:"is_nsfw"`
	OpenSeaURL   string    `pg:"opensea_url"`
	ProjectURL   string    `pg:"project_url"`
	WikiURL      string    `pg:"wiki_url"`
	DiscordURL   string    `pg:"discord_url"`
	TelegramURL  string    `pg:"telegram_url"`
	TwitterURL   string    `pg:"twitter_url"`
	InstagramURL string    `pg:"instagram_url"`
	CreatedDate  time.Time `pg:"created_date"`
	UpdatedAt    time.Time `pg:"updated_at"`
	Embedding    []float32 `pg:"embedding"`
}

type CollectionDynamic struct {
	tableName              struct{}  `pg:"collection_dynamic"`
	CollectionSlug         string    `pg:"collection_slug,pk"`
	GameID                 string    `pg:"game_id"`
	TotalAveragePrice      float64   `pg:"total_average_price"`
	TotalSupply            float64   `pg:"total_supply"`
	TotalVolume            float64   `pg:"total_volume"`
	TotalNumOwners         int       `pg:"total_num_owners"`
	TotalSales             float64   `pg:"total_sales"`
	TotalMarketCap         float64   `pg:"total_market_cap"`
	Sales                  float64   `pg:"sales"`
	Volume                 float64   `pg:"volume"`
	FloorPrice             float64   `pg:"floor_price"`
	FloorPriceCurrency     string    `pg:"floor_price_currency"`
	AveragePrice           float64   `pg:"average_price"`
	DailyUAW               int64     `pg:"daily_uaw"`
	MonthlyUAW             int64     `pg:"monthly_uaw"`
	TotalWallets           int64     `pg:"total_wallets"`
	TwitterFollowers       int64     `pg:"twitter_followers"`
	TwitterSentiment       float64   `pg:"twitter_sentiment"`
	FacebookFollowers      int64     `pg:"facebook_followers"`
	FacebookSentiment      float64   `pg:"facebook_sentiment"`
	InstagramFollowers     int64     `pg:"instagram_followers"`
	InstagramSentiment     float64   `pg:"instagram_sentiment"`
	RedditUsers            int64     `pg:"reddit_users"`
	RedditSentiment        float64   `pg:"reddit_sentiment"`
	DiscordServerSize      int64     `pg:"discord_server_size"`
	DiscordSentiment       float64   `pg:"discord_sentiment"`
	TelegramSupergroupSize int64     `pg:"telegram_supergroup_size"`
	TelegramSentiment      float64   `pg:"telegram_sentiment"`
	RRVal                  float64   `pg:"rr_val"`
	RRSymbol               string    `pg:"rr_symbol"`
	EventTimestamp         time.Time `pg:"event_timestamp,pk"`
	Embedding              []float32 `pg:"embedding"`
}

type Contract struct {
	tableName       struct{}  `pg:"contract"`
	CollectionSlug  string    `pg:"collection_slug"`
	ContractAddress string    `pg:"contract_address,pk"`
	Chain           string    `pg:"chain,pk"`
	Embedding       []float32 `pg:"embedding"`
}

type ERC20Transfer struct {
	tableName       struct{}  `pg:"erc20_transfers"`
	Buyer           string    `pg:"buyer"`
	Seller          string    `pg:"seller"`
	ContractAddress string    `pg:"contract_address"`
	Price           float64   `pg:"price"`
	Symbol          string    `pg:"symbol"`
	Decimals        int       `pg:"decimals"`
	TransactionHash string    `pg:"transaction_hash,pk"`
	EventTimestamp  time.Time `pg:"event_timestamp,pk"`
	CollectionSlug  string    `pg:"collection_slug"`
	Embedding       []float32 `pg:"embedding"`
}

type Fee struct {
	tableName      struct{}  `pg:"fee"`
	CollectionSlug string    `pg:"collection_slug,pk"`
	Fee            float64   `pg:"fee"`
	Recipient      string    `pg:"recipient,pk"`
	Embedding      []float32 `pg:"embedding"`
}

type NFT struct {
	tableName       struct{}  `pg:"nft"`
	CollectionSlug  string    `pg:"collection_slug"`
	GameID          string    `pg:"game_id"`
	TokenID         string    `pg:"token_id,pk"`
	ContractAddress string    `pg:"contract_address,pk"`
	TokenStandard   string    `pg:"token_standard"`
	Name            string    `pg:"name"`
	Description     string    `pg:"description"`
	ImageURL        string    `pg:"image_url"`
	MetadataURL     string    `pg:"metadata_url"`
	OpenSeaURL      string    `pg:"opensea_url"`
	UpdatedAt       time.Time `pg:"updated_at"`
	IsNSFW          bool      `pg:"is_nsfw"`
	IsDisabled      bool      `pg:"is_disabled"`
	Traits          []byte    `pg:"traits"`
	Embedding       []float32 `pg:"embedding"`
}

type PaymentToken struct {
	tableName       struct{}  `pg:"payment_tokens"`
	CollectionSlug  string    `pg:"collection_slug,pk"`
	ContractAddress string    `pg:"contract_address,pk"`
	Symbol          string    `pg:"symbol"`
	Decimals        int       `pg:"decimals"`
	Chain           string    `pg:"chain"`
	Embedding       []float32 `pg:"embedding"`
}

type NFTEvent struct {
	tableName          struct{}  `pg:"nft_events"`
	TransactionHash    string    `pg:"transaction_hash"`
	Marketplace        string    `pg:"marketplace"`
	MarketplaceAddress string    `pg:"marketplace_address"`
	BlockNumber        int64     `pg:"block_number"`
	OrderHash          string    `pg:"order_hash"`
	EventType          string    `pg:"event_type"`
	TokenID            string    `pg:"token_id,pk"`
	ContractAddress    string    `pg:"contract_address,pk"`
	CollectionSlug     string    `pg:"collection_slug"`
	GameID             string    `pg:"game_id"`
	Seller             string    `pg:"seller"`
	Buyer              string    `pg:"buyer"`
	Quantity           int       `pg:"quantity"`
	PriceVal           string    `pg:"price_val"`
	PriceCurrency      string    `pg:"price_currency"`
	PriceDecimals      string    `pg:"price_decimals"`
	EventTimestamp     time.Time `pg:"event_timestamp,pk"`
	Embedding          []float32 `pg:"embedding"`
}

type TokenPrice struct {
	tableName           struct{}  `pg:"token_price"`
	ContractAddress     string    `pg:"contract_address,pk"`
	EthPrice            float64   `pg:"eth_price"`
	USDTPrice           float64   `pg:"usdt_price"`
	USDTConversionPrice float64   `pg:"usdt_conversion_price"`
	EthConversionPrice  float64   `pg:"eth_conversion_price"`
	EventTimestamp      time.Time `pg:"event_timestamp,pk"`
	Embedding           []float32 `pg:"embedding"`
}

type NFTOwnership struct {
	tableName       struct{}  `pg:"nft_ownership"`
	Buyer           string    `pg:"buyer"`
	Seller          string    `pg:"seller"`
	TokenID         string    `pg:"token_id,pk"`
	ContractAddress string    `pg:"contract_address,pk"`
	TransactionHash string    `pg:"transaction_hash"`
	BuyTime         time.Time `pg:"buy_time,pk"`
	Quantity        int       `pg:"quantity"`
	SellTime        time.Time `pg:"sell_time"`
	CollectionSlug  string    `pg:"collection_slug"`
	GameID          string    `pg:"game_id"`
	Embedding       []float32 `pg:"embedding"`
}

type NFTDynamic struct {
	tableName       struct{}  `pg:"nft_dynamic"`
	CollectionSlug  string    `pg:"collection_slug"`
	TokenID         string    `pg:"token_id,pk"`
	ContractAddress string    `pg:"contract_address,pk"`
	RRVal           float64   `pg:"rr_val"`
	RRSymbol        string    `pg:"rr_symbol"`
	EventTimestamp  time.Time `pg:"event_timestamp,pk"`
	Embedding       []float32 `pg:"embedding"`
}

type NFTOffer struct {
	tableName       struct{}  `pg:"nft_offers"`
	OrderHash       string    `pg:"order_hash"`
	EventType       string    `pg:"event_type"`
	TokenID         string    `pg:"token_id,pk"`
	ContractAddress string    `pg:"contract_address,pk"`
	CollectionSlug  string    `pg:"collection_slug"`
	GameID          string    `pg:"game_id"`
	Seller          string    `pg:"seller"`
	Quantity        int       `pg:"quantity"`
	PriceVal        string    `pg:"price_val"`
	PriceCurrency   string    `pg:"price_currency"`
	PriceDecimals   string    `pg:"price_decimals"`
	StartDate       time.Time `pg:"start_date"`
	ExpirationDate  time.Time `pg:"expiration_date"`
	EventTimestamp  time.Time `pg:"event_timestamp,pk"`
	Embedding       []float32 `pg:"embedding"`
}

type NFTListing struct {
	tableName       struct{}  `pg:"nft_listings"`
	OrderHash       string    `pg:"order_hash"`
	TokenID         string    `pg:"token_id,pk"`
	ContractAddress string    `pg:"contract_address,pk"`
	CollectionSlug  string    `pg:"collection_slug"`
	GameID          string    `pg:"game_id"`
	Seller          string    `pg:"seller"`
	PriceVal        string    `pg:"price_val"`
	PriceCurrency   string    `pg:"price_currency"`
	PriceDecimals   string    `pg:"price_decimals"`
	StartDate       time.Time `pg:"start_date"`
	ExpirationDate  time.Time `pg:"expiration_date"`
	EventTimestamp  time.Time `pg:"event_timestamp,pk"`
	Embedding       []float32 `pg:"embedding"`
}

type Document struct {
	tableName      struct{}  `pg:"documents"`
	CollectionSlug string    `pg:"collection_slug,notnull"`
	CID            string    `pg:"cid,pk"`
	Content        string    `pg:"content"`
	Embedding      []float32 `pg:"embedding,type:vector"`
	EventTimestamp time.Time `pg:"event_timestamp,pk,type:timestamptz"`
}

type Conversation struct {
	tableName struct{}  `pg:"conversations"`
	ID        int64     `pg:"id,pk"`
	Title     string    `pg:"title,notnull"`
	CreatedAt time.Time `pg:"created_at,default:current_timestamp"`
	Messages  []Message `pg:"rel:has-many"`
}

type Message struct {
	tableName      struct{}      `pg:"messages"`
	ID             int64         `pg:"id,pk"`
	ConversationID int64         `pg:"conversation_id"`
	Role           string        `pg:"role,notnull"`
	Content        string        `pg:"content,notnull"`
	CreatedAt      time.Time     `pg:"created_at,default:current_timestamp"`
	Conversation   *Conversation `pg:"rel:has-one"`
	IsSummary      bool          `pg:"is_summary,notnull,default:false"`
}
