package mtc

// CoinName - Base coin name
const CoinName = "MetaCoinICO"

// CoinSymbol - Base coin symbol
const CoinSymbol = "MTC"

// CoinDecimals - Base coin max percimal
const CoinDecimals = 18

// InitSupply - Metacoin init supply
const InitSupply = 800000000

// MetaWallet - wallet data.
type TWallet struct {
	Id       string                   `json:"id"`
	Regdate  int64                    `json:"regdate"`
	Password string                   `json:"password"`
	Addinfo  string                   `json:"addinfo"`
	JobType  string                   `json:"job_type"`
	JobArgs  string                   `json:"job_args"`
	JobDate  int64                    `json:"jobdate"`
	Balance  []TMRC010Balance         `json:"balance"`
	MRC010   map[int]TMRC010BalanceV2 `json:"mrc010"`
	MRC402   map[string]NFTBalance    `json:"mrc402"`
	MRC800   map[string]string        `json:"mrc800"`
	Pending  map[int]string           `json:"pending"`
	Nonce    string                   `json:"nonce"`
}

type NFTBalance struct {
	Balance       string `json:"balance"`
	SaleAmount    string `json:"saleamount"`
	AuctionAmount string `json:"auctionamount"`
}

// BalanceInfo - token balance info with unlockdate
type TMRC010Balance struct {
	Balance       string `json:"balance"`
	Token         int    `json:"token"`
	UnlockDate    int64  `json:"unlockdate"`
	SaleAmount    string `json:"saleamount"`
	AuctionAmount string `json:"auctionamount"`
}

// BalanceInfo - token balance info with unlockdate
type TMRC010BalanceV2 struct {
	Balance       string         `json:"balance"`
	SaleAmount    string         `json:"saleamount"`
	AuctionAmount string         `json:"auctionamount"`
	LockedList    map[int]string `json:"locked_list"`
}

// Token MRC010 - TOKEN
type TMRC010 struct {
	Id             string           `json:"id"`
	Owner          string           `json:"owner"`
	Symbol         string           `json:"symbol"`
	CreateDate     int64            `json:"createdate"` // read only
	TotalSupply    string           `json:"totalsupply"`
	ReservedAmount string           `json:"reservedamount"`
	RemainAmount   string           `json:"remainamount"`
	BurnningAmount string           `json:"burnningamount"`
	SoftCap        string           `json:"softcap"`
	HardCap        string           `json:"hardcap"`
	Token          int              `json:"token"` // read only
	Name           string           `json:"name"`
	Information    string           `json:"information"` // writable
	URL            string           `json:"url"`         // writable
	Image          string           `json:"image"`       // writable
	Decimal        int              `json:"decimal"`
	Reserve        []TMRC010Reserve `json:"reserve"`
	Tier           []TMTC010ICOTier `json:"tier"`
	Status         string           `json:"status"` // editable - wait, iter-n, pause,
	TargetToken    map[int]int64    `json:"targettoken"`
	BaseToken      int              `json:"basetoken"`
	Type           string           `json:"type"`
	Logger         map[string]int64 `json:"logger"`
	JobType        string           `json:"job_type"`
	JobArgs        string           `json:"job_args"`
	JobDate        int64            `json:"jobdate"`
}

// TMTC010ICOTier token ico tier
type TMTC010ICOTier struct {
	StartDate    int64  `json:"startdate"`
	EndDate      int64  `json:"enddate"`
	Supply       string `json:"supply"`
	Rate         int    `json:"rate"`
	TierSN       int    `json:"tiersn"`
	Name         string `json:"name"`
	InvestorMin  string `json:"investormin"`
	RemainAmount string `json:"remainamount"`
	UnlockDate   int64  `json:"unlockdate"`
	ExpirePolicy string `json:"expirepolicy"` // move to next tier, burnin, move to owner
}

// PricePair - 금액, 토큰
type PricePair struct {
	Amount string `json:"amount"`
	Token  string `json:"token"`
}

// logDataBuy multi payyment info
type TDexPaymentInfo struct {
	FromAddr    string `json:"from_addr"`
	ToAddr      string `json:"to_addr"`
	Amount      string `json:"amount"`
	TokenID     string `json:"token"`
	TradeAmount string `json:"trade_amount"`
	TradeID     string `json:"trade_id"`
	PayType     string `json:"type"`
}

// TokenReserve token ico reserve
type TMRC010Reserve struct {
	Address    string `json:"address"`
	Value      string `json:"value"`
	UnlockDate int64  `json:"unlockdate"`
}

// MultiTransferList for multi transfer struct
type TMRC010TransferList struct {
	Address    string `json:"address"`
	Amount     string `json:"amount"`
	TokenID    string `json:"tokenid"`
	UnlockDate string `json:"unlockdate"`
	Tag        string `json:"tag"`
	Memo       string `json:"memo"`
}
