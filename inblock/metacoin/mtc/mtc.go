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
type MetaWallet struct {
	Regdate  int64                 `json:"regdate"`
	Password string                `json:"password"`
	Addinfo  string                `json:"addinfo"`
	JobType  string                `json:"job_type"`
	JobArgs  string                `json:"job_args"`
	JobDate  int64                 `json:"jobdate"`
	Balance  []BalanceInfo         `json:"balance"`
	MRC402   map[string]NFTBalance `json:"mrc402"`
	MRC800   map[string]string     `json:"mrc800"`
	Pending  map[int]string        `json:"pending"`
	Nonce    string                `json:"nonce"`
}

type NFTBalance struct {
	Balance       string `json:"balance"`
	SaleAmount    string `json:"saleamount"`
	AuctionAmount string `json:"auctionamount"`
}

// BalanceInfo - token balance info with unlockdate
type BalanceInfo struct {
	Balance    string `json:"balance"`
	Token      int    `json:"token"`
	UnlockDate int64  `json:"unlockdate"`
}

// MRC100Reward - game reward user list
type MRC100Reward struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Tag     string `json:"tag"`
	Memo    string `json:"memo"`
}

// MRC100Payment - game payment user list
type MRC100Payment struct {
	Address   string `json:"address"`
	Amount    string `json:"amount"`
	Memo      string `json:"memo"`
	Signature string `json:"signature"`
	TKey      string `json:"tkey"`
}

// MRC100Log - game log
type MRC100Log struct {
	Regdate int64  `json:"regdate"`
	Token   int    `json:"token"`
	Logger  string `json:"logger"`
	JobType string `json:"job_type"`
	JobArgs string `json:"job_args"`
}

// MRC020 - Delaied open
type MRC020 struct {
	Owner        string `json:"owner"`
	Data         string `json:"data"`
	CreateDate   int64  `json:"createdate"` // read only
	OpenDate     int64  `json:"opendate"`   // read only
	PublicKey    string `json:"publickey"`
	Algorithm    string `json:"algorithm"`
	ReferenceKey string `json:"referencekey"`
	IsOpen       int    `json:"is_open"`
	JobType      string `json:"job_type"`
	JobDate      int64  `json:"jobdate"`
	Type         string `json:"type"`
}

// Token MRC010 - TOKEN
type Token struct {
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
	Reserve        []TokenReserve   `json:"reserve"`
	Tier           []TokenTier      `json:"tier"`
	Status         string           `json:"status"` // editable - wait, iter-n, pause,
	TargetToken    map[int]int64    `json:"targettoken"`
	BaseToken      int              `json:"basetoken"`
	Type           string           `json:"type"`
	Logger         map[string]int64 `json:"logger"`
	JobType        string           `json:"job_type"`
	JobArgs        string           `json:"job_args"`
	JobDate        int64            `json:"jobdate"`
}

// PricePair - 금액, 토큰
type PricePair struct {
	Amount string `json:"amount"`
	Token  string `json:"token"`
}

// logDataBuy multi payyment info
type PaymentInfo struct {
	FromAddr    string `json:"from_addr"`
	ToAddr      string `json:"to_addr"`
	Amount      string `json:"amount"`
	TokenID     string `json:"token"`
	TradeAmount string `json:"trade_amount"`
	TradeID     string `json:"trade_id"`
	PayType     string `json:"type"`
}

// MRC400 for NFT Item project
type MRC400 struct {
	Owner        string `json:"owner"`
	CreateDate   int64  `json:"createdate"` // read only
	Name         string `json:"name"`
	URL          string `json:"url"`
	ImageURL     string `json:"image_url"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	ItemURL      string `json:"item_url"`
	ItemImageURL string `json:"item_image_url"`
	Data         string `json:"data"`
	AllowToken   string `json:"allow_token"`
	JobType      string `json:"job_type"`
	JobArgs      string `json:"job_args"`
	JobDate      int64  `json:"jobdate"`
}

// MRC401 for NFT ITEM
type MRC401 struct {
	MRC400               string `json:"mrc400"`          // MRC400 ID
	Owner                string `json:"owner"`           // 소유자
	ItemURL              string `json:"item_url"`        // item description URL
	ItemImageURL         string `json:"item_image_url"`  // image url
	GroupID              string `json:"groupid"`         // group id
	CreateDate           int64  `json:"createdate"`      // read only
	InititalReserve      string `json:"initial_reserve"` // 초기 판매 금액
	InititalToken        string `json:"initial_token"`   // 초기 판매 토큰
	MeltingFee           string `json:"melting_fee"`     // 멜팅 수수료(0.0001~ 99.9999%)
	MeltingDate          int64  `json:"melting_date"`    // Write Once 삭제 일시 0 이면 미 삭제,
	Transferable         string `json:"transferable"`    // 양도 가능 여부 : Permanent(가능), Bound(불가), Temprary(지금은 가능 - 불가능으로 변경 될 수 있음)
	SellDate             int64  `json:"sell_date"`       // 판매 시작 일시 0 이면 미 판매
	SellFee              string `json:"sell_fee"`        // read only 이체 수수료 비율(0.0001~ 99.9999%)
	SellPrice            string `json:"sell_price"`      // 판매 금액
	SellToken            string `json:"sell_token"`      // 판매 토큰
	JobType              string `json:"job_type"`
	JobArgs              string `json:"job_args"`
	JobDate              int64  `json:"jobdate"`
	AuctionDate          int64  `json:"auction_date"`           // 경매 시작 일시
	AuctionEnd           int64  `json:"auction_end"`            // 경매 종료 일시
	AuctionToken         string `json:"auction_token"`          // 경매 가능 토큰
	AuctionBiddingUnit   string `json:"auction_bidding_unit"`   // 경매 입찰 단위
	AuctionStartPrice    string `json:"auction_start_price"`    // 경매 시작 금액
	AuctionBuyNowPrice   string `json:"auction_buynow_price"`   // 경매 즉시 구매 금액
	AuctionCurrentPrice  string `json:"auction_current_price"`  // 경매 현 금액
	AuctionCurrentBidder string `json:"auction_current_bidder"` // 현재 입찰자
	LastTradeDate        int64  `json:"last_trade_date"`        // last buy or auction finish date
	LastTradeAmount      string `json:"last_trade_amount"`      // last buy or auction finish amount
	LastTradeToken       string `json:"last_trade_token"`       // last buy or auction finish token
	LastTradeType        string `json:"last_trade_type"`        // "Auction" or "Sell"
}

// MRC401job for NFT ITEM create
type MRC401job struct {
	MRC400          string `json:"mrc400"`          // MRC400 ID
	ItemID          string `json:"item_id"`         // MRC401 Item ID
	ItemURL         string `json:"item_url"`        // item description URL
	ItemImageURL    string `json:"item_image_url"`  // image url
	GroupID         string `json:"groupid"`         // group id
	CreateDate      int64  `json:"createdate"`      // read only
	InititalReserve string `json:"initial_reserve"` // 초기 판매 금액
	InititalToken   string `json:"initial_token"`   // 초기 판매 토큰
	MeltingFee      string `json:"melting_fee"`     // 멜팅 수수료(0.0001~ 99.9999%)
	Transferable    string `json:"transferable"`    // 양도 가능 여부 : Permanent(가능), Bound(불가), Temprary(지금은 가능 - 불가능으로 변경 될 수 있음)
	SellFee         string `json:"sell_fee"`        // read only 이체 수수료 비율(0.0001~ 99.9999%)
}

// MRC401Sell for NFT ITEM sell
type MRC401Sell struct {
	ItemID    string `json:"id"` // MRC401 Item ID
	SellPrice string `json:"amount"`
	SellToken string `json:"token"` // read only 이체 수수료 비율(0.0001~ 99.9999%)
}

// MRC401Auction for NFT ITEM auction
type MRC401Auction struct {
	ItemID             string `json:"id"`      // MRC401 Item ID
	AuctionEnd         int64  `json:"end"`     // 경매 종료 일시
	AuctionToken       string `json:"token"`   // 경매 가능 토큰
	AuctionBiddingUnit string `json:"bidding"` // 경매 입찰 단위
	AuctionStartPrice  string `json:"start"`   // 경매 시작 금액
	AuctionBuyNowPrice string `json:"buynow"`  // 경매 즉시 구매 금액
}

// Token MRC402 - NFT TOKEN
type MRC402 struct {
	Id                string `json:"id"`                // MRC 402 ID
	Creator           string `json:"creator"`           // 생성자
	CreatorCommission string `json:"creatorcommission"` // 판매/경매시 생성자가 가져가는 수수료 (0~10%)
	TotalSupply       string `json:"totalsupply"`       // 총 발행량 - 현재 유통량
	MeltedAmount      string `json:"meltedamount"`      // Melt된 수량

	Decimal     int    `json:"decimal"`     // 가능하면 0 으로 할것.
	Name        string `json:"name"`        // 외부에 노출되는 이름
	ExpireDate  int64  `json:"expiredate"`  // 만기 일자 - unix timestamp
	ImageURL    string `json:"image"`       // Image URL
	URL         string `json:"url"`         // 수정가능 - URL
	Data        string `json:"data"`        // 수정가능 - Data Non-human readable text may be json object ?
	Info        string `json:"info"`        // 수정가능 - Human readable text. aka Markdown
	SocialMedia string `json:"socialmedia"` // 수정가능 - {"twitter" : "https:// ...",  "We Schedule" : "https://domain.com/path" }
	// Icon with Description site : twitter, facebook, telegram,  instagram, youtube, tiktok, snapchat,
	//                              discord, twitch, pinterest, linkedin, wechat, qq, douyin, weibo, github

	CopyrightRegCountry string `json:"copyright_registration_country"`
	CopyrightRegistrar  string `json:"copyright_registrar"`
	CopyrightRegNumber  string `json:"copyright_registration_number"`

	ShareHolder    map[string]string `json:"shareholder"`    // { 저작권자 주소 : 판매/경매시 해당 주소가 가져가는 수수료 (0~10%)}
	InitialReserve map[string]string `json:"initialreserve"` // 토큰 1개 소각시 받을 수 있는 자산 목록   { tokenID : amount }

	JobType string `json:"job_type"`
	JobArgs string `json:"job_args"`
	JobDate int64  `json:"jobdate"`
}

type MRC402DEX struct {
	Id           string `json:"dexid"`
	MRC402       string `json:"mrc402"` // MRC402 ID
	Seller       string `json:"seller"` // 판매자
	Buyer        string `json:"buyer"`  // 판매자
	Amount       string `json:"amount"` // 총 판매 수량
	RemainAmount string `json:"remain_amount"`

	PlatformName       string `json:"platform_name"` // 플렛폼 이름
	PlatformURL        string `json:"platform_url"`
	PlatformAddress    string `json:"platform_address"`    // 판매/경매시 수수료를 받을 플렛폼 주소
	PlatformCommission string `json:"platform_commission"` // 판매/경매시 수수료를 받을 플렛폼가 가져가는 수수료 (0~10%)

	RegDate    int64 `json:"regdate"`     // 등록 일시		must 0 <
	SellDate   int64 `json:"sell_date"`   // 거래 완료 일시  0 : not sale or auction finish, 0 < : sale or auction win
	CancelDate int64 `json:"cancel_date"` // 취소 일시		0 : not cancel, 0 < : canceled

	SellPrice string `json:"sell_price"` // 판매 금액
	SellToken string `json:"sell_token"` // 거래 가능 토큰

	AuctionBidCount      int    `json:"auction_bid_count"`
	AuctionStartDate     int64  `json:"auction_start_date"`     // 경매 시작 일시	0 < : auction item, 0: not auction item
	AuctionEndDate       int64  `json:"auction_end_date"`       // 경매 종료 일시   0 < : auction item
	AuctionSettledDate   int64  `json:"auction_settle_date"`    // 경매 정산 일시   0 < : trade or auction settled, 0: not yet.
	AuctionBiddingUnit   string `json:"auction_bidding_unit"`   // 경매 최소 입찰 단위	0 : free bidding
	AuctionStartPrice    string `json:"auction_start_price"`    // 경매 시작 금액		0 < : auction item
	AuctionBuyNowPrice   string `json:"auction_buynow_price"`   // 경매 즉시 구매 금액
	AuctionCurrentPrice  string `json:"auction_current_price"`  // 경매 현 금액		"" : nothing bidder
	AuctionCurrentBidder string `json:"auction_current_bidder"` // 현재 입찰자		"" : nothing bidder

	JobType string `json:"job_type"`
	JobArgs string `json:"job_args"`
	JobDate int64  `json:"jobdate"`
}

// MRC400 for NFT Item project
type MRC800 struct {
	Owner        string `json:"owner"`
	CreateDate   int64  `json:"createdate"` // read only
	Name         string `json:"name"`
	URL          string `json:"url"`
	ImageURL     string `json:"image_url"`
	Transferable string `json:"transferable"`
	Description  string `json:"description"`
	JobType      string `json:"job_type"`
	JobArgs      string `json:"job_args"`
	JobDate      int64  `json:"jobdate"`
}

// TokenReserve token ico reserve
type TokenReserve struct {
	Address    string `json:"address"`
	Value      string `json:"value"`
	UnlockDate int64  `json:"unlockdate"`
}

// MultiTransferList for multi transfer struct
type MultiTransferList struct {
	Address    string `json:"address"`
	Amount     string `json:"amount"`
	TokenID    string `json:"tokenid"`
	UnlockDate string `json:"unlockdate"`
	Tag        string `json:"tag"`
	Memo       string `json:"memo"`
}

// TokenTier token ico tier
type TokenTier struct {
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

// ExchangeItem : MRC040 exchange request
type ExchangeItem struct {
	Owner        string `json:"owner"`
	Side         string `json:"side"`
	BaseToken    int    `json:"basetoken"`
	TargetToken  int    `json:"targettoken"`
	Price        string `json:"price"`
	Qtt          string `json:"qtt"`
	RemainQtt    string `json:"remainqtt"`
	Regdate      int64  `json:"regdate"`
	CompleteDate int64  `json:"complete_date"`
	CancelDate   int64  `json:"cancel_date"`
	Status       string `json:"status"`
	JobType      string `json:"job_type"`
	JobArgs      string `json:"job_args"`
	JobDate      int64  `json:"jobdate"`
	Type         string `json:"type"`
}

// ExchangeResult : MRC040 exchange result
type ExchangeResult struct {
	SellOwner  string `json:"sell_owner"`
	BuyOwner   string `json:"buy_owner"`
	SellItemTX string `json:"from_item_tx"`
	BuyItemTX  string `json:"to_item_tx"`
	SellToken  int    `json:"sell_token"`
	BuyToken   int    `json:"buy_token"`
	Price      string `json:"price"`
	Qtt        string `json:"qtt"`
	Regdate    int64  `json:"regdate"`
	JobType    string `json:"job_type"`
	JobArgs    string `json:"job_args"`
	JobDate    int64  `json:"jobdate"`
	Type       string `json:"type"`
}

// MRC410 : Coupon/Ticket base
type MRC410 struct {
	Creator    string `json:"creator"`
	Name       string `json:"name"`
	IsTransfer int    `json:"is_transfer"`
	StartDate  int64  `json:"start_date"`
	EndDate    int64  `json:"end_date"`
	Term       int    `json:"term"`
	Code       string `json:"code"`
	Data       string `json:"data"`
	JobType    string `json:"job_type"`
	JobArgs    string `json:"job_args"`
	JobDate    int64  `json:"jobdate"`
}

// MRC411 : Coupon/Ticket
type MRC411 struct {
	Creator    string `json:"creator"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	CreateDate int64  `json:"create_date"`
	ExpireDate int64  `json:"expire_date"`
	Code       string `json:"code"`
	Data       string `json:"data"`
	JobType    string `json:"job_type"`
	JobArgs    string `json:"job_args"`
	JobDate    int64  `json:"jobdate"`
}

// MRC011 : Coupon/Ticket base
type MRC011 struct {
	Creator      string `json:"creator"`
	Name         string `json:"name"`
	TotalSupply  int    `json:"totalsupply"`
	UsedCount    int    `json:"used_count"`
	PublishCount int    `json:"publish_count"`
	RemainCount  int    `json:"remain_count"`
	ValidityYype string `json:"validity_type"`
	IsTransfer   int    `json:"is_transfer"`
	StartDate    int64  `json:"start_date"`
	EndDate      int64  `json:"end_date"`
	Term         int    `json:"term"`
	Code         string `json:"code"`
	Data         string `json:"data"`
	JobType      string `json:"job_type"`
	JobArgs      string `json:"job_args"`
	JobDate      int64  `json:"jobdate"`
}

// MRC012 : Coupon/Ticket
type MRC012 struct {
	Creator    string `json:"creator"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	CreateDate int64  `json:"create_date"`
	ExpireDate int64  `json:"expire_date"`
	Code       string `json:"code"`
	Data       string `json:"data"`
	JobType    string `json:"job_type"`
	JobArgs    string `json:"job_args"`
	JobDate    int64  `json:"jobdate"`
}

// MRC030 : Item
type MRC030 struct {
	Creator            string               `json:"creator"`
	Title              string               `json:"title"`
	URL                string               `json:"url"`
	Description        string               `json:"description"`
	StartDate          int64                `json:"start_date"`
	EndDate            int64                `json:"end_date"`
	Reward             string               `json:"reward"`
	TotalReward        string               `json:"total_reward"`
	RewardToken        int                  `json:"reward_token"`
	MaxRewardRecipient int                  `json:"max_reward_recipient"`
	RewardType         int                  `json:"reward_type"`
	Question           []MRC030Question     `json:"question"`
	QuestionCount      int                  `json:"question_count"`
	QuestionInfo       []MRC030QuestionInfo `json:"question_info"`
	Voter              map[string]int       `json:"voter"`
	IsFinish           int                  `json:"is_finish"`
	IsNeedSign         int                  `json:"is_need_sign"`
	JobType            string               `json:"job_type"`
	JobArgs            string               `json:"job_args"`
	JobDate            int64                `json:"jobdate"`
}

// MRC030QuestionInfo : MRC030 질문 수량
type MRC030QuestionInfo struct {
	AnswerCount    int   `json:"a"`
	SubAnswerCount []int `json:"s"`
}

// MRC030Question : MRC030 질문
type MRC030Question struct {
	Question string       `json:"question"`
	URL      string       `json:"url"`
	Item     []MRC030Item `json:"item"`
}

// MRC030Item : MRC030 답변
type MRC030Item struct {
	Answer   string          `json:"answer"`
	URL      string          `json:"url"`
	SubQuery string          `json:"subquery"`
	SubItem  []MRC030SubItem `json:"subitem"`
}

// MRC030SubItem : MRC030 답변
type MRC030SubItem struct {
	SubAnswer string `json:"subanswer"`
	URL       string `json:"url"`
}

// MRC031 : Answer
type MRC031 struct {
	Regdate int64          `json:"regdate"`
	Voter   string         `json:"voter"`
	Answer  []MRC031Answer `json:"answer"`
	JobType string         `json:"job_type"`
	JobArgs string         `json:"job_args"`
	JobDate int64          `json:"jobdate"`
}

// MRC031Answer : MRC031 답변
type MRC031Answer struct {
	Answer    int `json:"answer"`
	SubAnswer int `json:"subanswer"`
}
