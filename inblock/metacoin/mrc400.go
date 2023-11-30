package metacoin

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/shopspring/decimal"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

/*
MRC400 변경사항

- Allow_token(판매시 거래 가능한 토큰) 정책 변경
  "0" 혹은 빈 값 : 메타코인만 허용 / 그외 : 메타코인과 지정된 토큰만 허용
  =>   "0" 혹은 빈 값 : 모든 토큰 허용 / 그외 : 메타코인과 지정된 토큰만 허용

- 파트너 추가("파트너" 명칭은 테스트넷 올라가기전까지 변경 가능합니다.)
  파트너는 3가지로 구분됩니다.

  partner : MRC401 생성, MRC401 업데이트, MRC401 생성하면서 판매, 성생후 거래/이체된적이 없는 MRC401 아이템의 판매/판매 취소, 경매/경매취소
  seller : MRC401 생성하면서 판매, 성생후 거래/이체된적이 없는 MRC401 아이템의 판매/판매 취소, 경매/경매취소
  creator : MRC401 생성, MRC401 업데이트


MRC401 변경 사항
- 한번에 최대 1024 개 를 생성할 수 있습니다.

- 생성하면서 판매 기능 추가
  MRC401 생성 + 거래체결이 합쳐진 프로토콜 입니다.

- 거래플렛폼이 수수료를 가져갈 수 있도록 변경됩니다.

- 최대 5명의 저작권자가 추가될 수 있습니다.
  각각의 저작권자는 거래시 0.0001 ~ 10.000 % 의 수수료를 가져갈 수 있습니다.

*/

type MRC401ActionType int

const (
	MRC401AT_Buy = iota
	MRC401AT_Sell
	MRC401AT_UnSell
	MRC401AT_Auction
	MRC401AT_Bidding
	MRC401AT_UnAuction
	MRC401AT_Create
	MRC401AT_Update
	MRC401AT_CreateTrade
	MRC401AT_Melt
	MRC401AT_Transfer
)

type MRC401Status int

const (
	MRC401STATUS_NORMALE        = iota // not sale or auction
	MRC401STATUS_SALE                  // sale
	MRC401STATUS_AUCTION_WAIT          // wait for auction start
	MRC401STATUS_AUCTION               // auction(biddable)
	MRC401STATUS_AUCTION_END           // auction end
	MRC401STATUS_AUCTION_FINISH        // auction finish
)

// TMRC400 for NFT Item project
type TMRC400 struct {
	Id         string `json:"id"`         // MRC 402 ID
	CreateDate int64  `json:"createdate"` // read only
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	ImageURL   string `json:"image_url"`

	// 2023. 9 policy change
	AllowToken string `json:"allow_token"`

	ItemURL      string `json:"item_url"`
	ItemImageURL string `json:"item_image_url"`
	Category     string `json:"category"`
	Description  string `json:"description"`

	// add 2023. 9
	SocialMedia string `json:"socialmedia"`
	// 수정가능 - {"twitter" : "https:// ...",  "We Schedule" : "https://domain.com/path" }
	// Icon with Description site : twitter, facebook, telegram,  instagram, youtube, tiktok, snapchat,
	//                              discord, twitch, pinterest, linkedin, wechat, qq, douyin, weibo, github

	// add 2023. 9
	Partner map[string]string `json:"partner"`
	// { MRC401 Create, Update, Sell Permission owner Address : Permission }
	// Permission(MAX 10 address)
	//	- partner : MRC401 create, MRC401 update, MRC401 create for trade, MRC401 first sell
	//	- seller : MRC401 create for trade, MRC401 first sell
	//	- creator : MRC401 create, MRC401 update
	// ex :  { "MT123" : "partnet", "MT456" : "seller", "MT789" : "creator" }

	Data string `json:"data"`

	JobType string `json:"job_type"`
	JobArgs string `json:"job_args"`
	JobDate int64  `json:"jobdate"`
}

// TMRC401 for NFT ITEM
type TMRC401 struct {
	Id                   string `json:"id"`              // MRC401 ID
	MRC400               string `json:"mrc400"`          // MRC400 ID
	Owner                string `json:"owner"`           // 소유자
	Creator              string `json:"creator"`         // 생성자
	ItemURL              string `json:"item_url"`        // item description URL
	ItemImageURL         string `json:"item_image_url"`  // image url
	GroupID              string `json:"groupid"`         // group id
	CreateDate           int64  `json:"createdate"`      // read only
	InititalReserve      string `json:"initial_reserve"` // 초기 판매 금액
	InititalToken        string `json:"initial_token"`   // 초기 판매 토큰
	MeltingFee           string `json:"melting_fee"`     // 멜팅 수수료(0.0001~ 99.9999%)
	MeltingDate          int64  `json:"melting_date"`    // Write Once 삭제 일시 0 이면 미 삭제,
	Transferable         string `json:"transferable"`    // 양도 가능 여부 : Permanent(가능), Bound(불가), Temporary(지금은 가능 - 불가능으로 변경 될 수 있음)
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
	LastTradeType        string `json:"last_trade_type"`        // "Auction" or "Sell" or "Transfer"
	LastWorker           string `json:"last_worker"`            // last sell, auction, creator, unsell, unauction
	// add 2023. 9
	ShareHolder        map[string]string `json:"shareholder"`   // { 저작권자 주소 : 판매/경매시 해당 주소가 가져가는 수수료 (0~10%)}, 최대 5명
	PlatformName       string            `json:"platform_name"` // 플렛폼 이름
	PlatformURL        string            `json:"platform_url"`
	PlatformAddress    string            `json:"platform_address"`    // 판매/경매시 수수료를 받을 플렛폼 주소
	PlatformCommission string            `json:"platform_commission"` // 판매/경매시 수수료를 받을 플렛폼가 가져가는 수수료 (0~10%)
}

// for NFT ITEM create
type TMRC401CreateUpdate struct {
	MRC400          string `json:"mrc400"`          // MRC400 ID
	ItemID          string `json:"item_id"`         // MRC401 Item ID
	ItemURL         string `json:"item_url"`        // item description URL
	ItemImageURL    string `json:"item_image_url"`  // image url
	GroupID         string `json:"groupid"`         // group id
	InititalReserve string `json:"initial_reserve"` // 초기 판매 금액
	InititalToken   string `json:"initial_token"`   // 초기 판매 토큰
	MeltingFee      string `json:"melting_fee"`     // 멜팅 수수료(0.0001~ 99.9999%)
	Transferable    string `json:"transferable"`    // 양도 가능 여부 : Permanent(가능), Bound(불가), Temporary(지금은 가능 - 불가능으로 변경 될 수 있음)
	SellFee         string `json:"sell_fee"`        // read only 이체 수수료 비율(0.0001~ 99.9999%)
	// add 2023. 9
	ShareHolder map[string]string `json:"shareholder"` // { 저작권자 주소 : 판매/경매시 해당 주소가 가져가는 수수료 (0~10%)}, 최대 5명
}

// TMRC401Sell for NFT ITEM sell
type TMRC401Sell struct {
	ItemID             string `json:"id"` // MRC401 Item ID
	SellPrice          string `json:"amount"`
	SellToken          string `json:"token"`         // read only 이체 수수료 비율(0.0001~ 99.9999%)
	PlatformName       string `json:"platform_name"` // 플렛폼 이름
	PlatformURL        string `json:"platform_url"`
	PlatformAddress    string `json:"platform_address"`    // 판매/경매시 수수료를 받을 플렛폼 주소
	PlatformCommission string `json:"platform_commission"` // 판매/경매시 수수료를 받을 플렛폼가 가져가는 수수료 (0~10%)
}

// TMRC401Auction for NFT ITEM auction
type TMRC401Auction struct {
	ItemID             string `json:"id"`                   // MRC401 Item ID
	AuctionToken       string `json:"token"`                // 경매 가능 토큰
	AuctionBiddingUnit string `json:"auction_bidding_unit"` // 경매 입찰 단위
	AuctionStartPrice  string `json:"auction_start_price"`  // 경매 시작 금액
	AuctionBuyNowPrice string `json:"auction_buynow_price"` // 경매 즉시 구매 금액
	AuctionStartDate   int64  `json:"auction_start_date"`
	AuctionEndDate     int64  `json:"auction_end_date"` // 경매 종료 일시
	PlatformName       string `json:"platform_name"`    // 플렛폼 이름
	PlatformURL        string `json:"platform_url"`
	PlatformAddress    string `json:"platform_address"`    // 판매/경매시 수수료를 받을 플렛폼 주소
	PlatformCommission string `json:"platform_commission"` // 판매/경매시 수수료를 받을 플렛폼가 가져가는 수수료 (0~10%)
}

/* Sell, Auction :
- actor is mrc401 owner
- actor is mrc400 partner/seller and mrc401 owner is mrc400 owner and `LastTradeType` is empty.

UnSell, UnAuction:
- actor is mrc401 owner
- actor is mrc400 partner/seller and mrc401 owner is mrc400 owner and lastworker

Create
- actor is mrc400 owner
- actor is mrc400 partner/creator

Update
- actor is mrc400 owner
- actor is mrc400 partner/creator and mrc401 creator

MRC401AT_CreateTrade
- actor is mrc400 owner
- actor is mrc400 partner/seller
*/

func Mrc401PermissionCheck(MRC400 TMRC400, MRC401 TMRC401, actor string, action MRC401ActionType) error {
	var permission string
	var isPartner bool

	if !util.IsAddress(actor) {
		return errors.New("1002," + actor + " is not Metacoin address")
	}
	// owner
	switch action {
	case MRC401AT_Transfer, MRC401AT_Melt:
		if MRC401.Owner == actor {
			return nil
		} else {
			return errors.New("3004,MRC401 [" + MRC401.Id + "] is not your item")
		}
	case MRC401AT_Sell, MRC401AT_UnSell, MRC401AT_Auction, MRC401AT_UnAuction:
		if MRC401.Owner == actor {
			return nil
		} else if MRC401.Owner != MRC400.Owner {
			return errors.New("3004,MRC401 [" + MRC401.Id + "] is not your item")
		}
	case MRC401AT_Create, MRC401AT_Update, MRC401AT_CreateTrade:
		if actor == MRC400.Owner {
			return nil
		}
	}

	if permission, isPartner = MRC400.Partner[actor]; isPartner == false {
		return errors.New("3007,MRC401 operation is only available to MRC400 partner")
	}
	switch action {
	case MRC401AT_Sell, MRC401AT_Auction:
		if permission != "partner" && permission != "seller" {
			return errors.New("3007,MRC401 [" + MRC401.Id + "] sell or auction is only available to MRC400 partner or seller")
		}
	case MRC401AT_UnSell, MRC401AT_UnAuction:
		if permission != "partner" && permission != "seller" {
			return errors.New("3007,MRC401 [" + MRC401.Id + "] unsell or unauction is only available to MRC400 partner or seller")
		}
		if MRC401.LastWorker != actor {
			return errors.New("3007,MRC401 [" + MRC401.Id + "] unsell or unauction is only available to last worker")
		}

	case MRC401AT_Create:
		if permission != "partner" && permission != "creator" {
			return errors.New("3007,MRC401 creation is only available to MRC400 partner or creator")
		}
	case MRC401AT_Update:
		if permission != "partner" && permission != "creator" {
			return errors.New("3007,MRC401 update is only available to MRC400 partner or creator")
		}
		if MRC401.Creator != actor {
			return errors.New("3007,MRC401 [" + MRC401.Id + "] update is only available to creator")
		}

	case MRC401AT_CreateTrade:
		if permission != "seller" && permission != "partner" {
			return errors.New("3007,MRC401 creation is only available to MRC400 partner or creator")
		}
	}
	return nil
}

// Mrc400Create create MRC400 Item
func Mrc400Create(stub shim.ChaincodeStubInterface, args []string) (string, error) {
	var err error
	var MRC400Creator mtc.TWallet
	var MRC400 TMRC400
	var argdat []byte

	if len(args) < 14 {
		return "", errors.New("1000,mrc400create operation must include four arguments : " +
			"owner, name, url, imageurl, allowtoken, " +
			"itemurl, itemimageurl, category, description, socialmedia, " +
			"partner, data, sign, nonce")
	}
	MRC400 = TMRC400{
		CreateDate: time.Now().Unix(),
		AllowToken: "0",
	}

	if MRC400Creator, err = GetAddressInfo(stub, args[0]); err != nil {
		return "", err
	}

	if err = NonceCheck(&MRC400Creator, args[13],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[7],
			args[5], args[6], args[10], args[11], args[13]}, "|"),
		args[12]); err != nil {
		return "", err
	}

	// 0 owner
	if err = util.DataAssign(args[0], &MRC400.Owner, "address", 40, 40, false); err != nil {
		return "", errors.New("3005,Data must be 1 to 4096 characters long")
	}

	// 1 name
	if err = util.DataAssign(args[1], &MRC400.Name, "string", 1, 128, false); err != nil {
		return "", errors.New("3005,Name must be 1 to 128 characters long")
	}

	// 2 url
	if err = util.DataAssign(args[2], &MRC400.URL, "url", 1, 1024, false); err != nil {
		return "", errors.New("3005,Url must be 1 to 1024 characters long URL")
	}

	// 3 image url
	if err = util.DataAssign(args[3], &MRC400.ImageURL, "url", 1, 255, false); err != nil {
		return "", errors.New("3005,ImageURL must be 1 to 1024 characters long URL")
	}

	// 4 allow token
	if args[4] != "" {
		if _, _, err = GetMRC010(stub, args[4]); err != nil {
			return "", errors.New("3005,Token id " + args[4] + " error : " + err.Error())
		}
		if err = util.DataAssign(args[4], &MRC400.AllowToken, "string", 1, 128, false); err != nil {
			return "", errors.New("3005,AllowToken must be 1 to 128 characters long")
		}
	}

	// 5 item data url
	if err = util.DataAssign(args[5], &MRC400.ItemURL, "url", 1, 255, true); err != nil {
		return "", errors.New("3005,ItemURL must be 1 to 1024 characters long URL")
	}

	// 6 item image url
	if err = util.DataAssign(args[6], &MRC400.ItemImageURL, "url", 1, 255, true); err != nil {
		return "", errors.New("3005,ItemImageURL must be 1 to 1024 characters long URL")
	}

	// 7 category
	if err = util.DataAssign(args[7], &MRC400.Category, "string", 1, 64, true); err != nil {
		return "", errors.New("3005,Category must be 1 to 64 characters long")
	}

	// 8 description
	if err = util.DataAssign(args[8], &MRC400.Description, "string", 1, 40960, true); err != nil {
		return "", errors.New("3005,Description must be 1 to 40960 characters long")
	}

	// 9 social media
	if err = util.DataAssign(args[9], &MRC400.SocialMedia, "string", 0, 40960, true); err != nil {
		return "", errors.New("3005,SocialMedia must be 0 to 40960 characters long")
	}

	// 10 partner
	if args[10] != "" {
		if err = json.Unmarshal([]byte(args[10]), &MRC400.Partner); err != nil {
			return "", errors.New("3005,partner data value error : " + err.Error())
		}
		if len(MRC400.Partner) > 64 {
			return "", errors.New("3002,There must be 64 or fewer partner")
		}
		index := 0
		for partner_address, partner_permission := range MRC400.Partner {
			if _, err = GetAddressInfo(stub, partner_address); err != nil {
				return "", errors.New("3005," + util.GetOrdNumber(index) + " partner item error : " + err.Error())
			}
			switch partner_permission {
			case "partner", "seller", "creator":
				index++
			default:
				return "", errors.New("3005," + util.GetOrdNumber(index) + " partner item permission is partner or seller or creator")
			}
		}
	}

	// 11 data
	if err = util.DataAssign(args[11], &MRC400.Data, "string", 0, 40960, false); err != nil {
		return "", errors.New("3005,Data must be 0 to 40960 characters long")
	}

	var isSuccess = false
	temp := util.GenerateKey("MRC400_", args)
	for i := 0; i < 10; i++ {
		MRC400.Id = fmt.Sprintf("%39s%1d", temp, i)
		argdat, err = stub.GetState(MRC400.Id)
		if err != nil {
			return "", errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if argdat != nil { // key already exists
			continue
		} else {
			isSuccess = true
			break
		}
	}

	if !isSuccess {
		return "", errors.New("3005,Data generate error, retry again")
	}

	if err = setMRC400(stub, MRC400, "mrc400_create", []string{MRC400.Id,
		args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[13]}); err != nil {
		return "", err
	}

	if err = SetAddressInfo(stub, MRC400Creator, "mrc400create",
		[]string{MRC400.Id, args[0], args[1], args[2], args[3], args[4],
			args[5], args[6], args[7], args[8], args[9],
			args[10], args[11], args[12], args[13]}); err != nil {
		return "", err
	}
	return MRC400.Id, nil
}

// Mrc400Update for MTC110 token update.
func Mrc400Update(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC400Creator mtc.TWallet
	var MRC400 TMRC400
	if len(args) < 14 {
		return errors.New("1000,mrc400update operation must include four arguments : " +
			"mrc400id, name, url, imageurl, allowtoken, " +
			"itemurl, itemimageurl, category, description, socialmedia, " +
			"partner, data, signature, nonce")
	}

	if MRC400, _, err = GetMRC400(stub, args[0]); err != nil {
		return err
	}

	if MRC400Creator, err = GetAddressInfo(stub, MRC400.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&MRC400Creator, args[13],
		// mrc400id, name, url, imageurl, category
		// itemurl, itemimageurl, partner, data
		strings.Join([]string{args[0], args[1], args[2], args[3], args[7],
			args[5], args[6], args[10], args[11], args[13]}, "|"),
		args[12]); err != nil {
		return err
	}

	// 1 name
	if err = util.DataAssign(args[1], &MRC400.Name, "string", 1, 128, false); err != nil {
		return errors.New("3005,Name must be 1 to 128 characters long")
	}

	// 2 url
	if err = util.DataAssign(args[2], &MRC400.URL, "url", 1, 1024, false); err != nil {
		return errors.New("3005,Url must be 1 to 1024 characters long URL")
	}

	// 3 image url
	if err = util.DataAssign(args[3], &MRC400.ImageURL, "url", 1, 255, false); err != nil {
		return errors.New("3005,ImageURL must be 1 to 1024 characters long URL")
	}

	// 4 allow token
	if args[4] != "" {
		if _, _, err = GetMRC010(stub, args[4]); err != nil {
			return errors.New("3005,Token id " + args[4] + " error : " + err.Error())
		}
		if err = util.DataAssign(args[4], &MRC400.AllowToken, "string", 1, 128, false); err != nil {
			return errors.New("3005,AllowToken must be 1 to 128 characters long")
		}
	} else {
		MRC400.AllowToken = "0"
	}

	// 5 item data url
	if err = util.DataAssign(args[5], &MRC400.ItemURL, "url", 0, 255, false); err != nil {
		return errors.New("3005,ItemURL must be 1 to 1024 characters long URL")
	}

	// 6 item image url
	if err = util.DataAssign(args[6], &MRC400.ItemImageURL, "url", 0, 255, false); err != nil {
		return errors.New("3005,ItemImageURL must be 1 to 1024 characters long URL")
	}

	// 7 category
	if err = util.DataAssign(args[7], &MRC400.Category, "string", 0, 64, false); err != nil {
		return errors.New("3005,Category must be 1 to 64 characters long")
	}

	// 8 description
	if err = util.DataAssign(args[8], &MRC400.Description, "string", 0, 40960, false); err != nil {
		return errors.New("3005,Description must be 1 to 40960 characters long")
	}

	// 9 social media
	if err = util.DataAssign(args[9], &MRC400.SocialMedia, "string", 0, 40960, false); err != nil {
		return errors.New("3005,SocialMedia must be 0 to 40960 characters long")
	}

	// 10 partner
	if args[10] != "" {
		if err = json.Unmarshal([]byte(args[10]), &MRC400.Partner); err != nil {
			return errors.New("3005,partner data value error : " + err.Error())
		}
		if len(MRC400.Partner) > 64 {
			return errors.New("3002,There must be 64 or fewer partner")
		}
		index := 0
		for partner_address, partner_permission := range MRC400.Partner {
			if _, err = GetAddressInfo(stub, partner_address); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " partner item error : " + err.Error())
			}
			switch partner_permission {
			case "partner", "seller", "creator":
				index++
			default:
				return errors.New("3005," + util.GetOrdNumber(index) + " partner item permission is partner or seller or creator")
			}
		}
	}

	// 11 data
	if err = util.DataAssign(args[11], &MRC400.Data, "string", 0, 40960, false); err != nil {
		return errors.New("3005,Data must be 0 to 40960 characters long")
	}

	if err = setMRC400(stub, MRC400, "mrc400_update", []string{MRC400.Id,
		args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[13]}); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, MRC400Creator, "mrc400update", []string{MRC400.Id,
		args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[13]}); err != nil {
		return err
	}
	return nil
}

// GetMRC400 - get MRC400 token
func GetMRC400(stub shim.ChaincodeStubInterface, mrc400id string) (TMRC400, []byte, error) {
	var dat []byte
	var err error
	var mrc400 TMRC400

	if strings.Index(mrc400id, "MRC400_") != 0 || len(mrc400id) != 40 {
		return mrc400, dat, errors.New("6102,invalid MRC400 data address")
	}

	dat, err = stub.GetState(mrc400id)
	if err != nil {
		return mrc400, dat, errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return mrc400, dat, errors.New("6004,MRC400 [" + mrc400id + "] not exist")
	}
	if err = json.Unmarshal(dat, &mrc400); err != nil {
		return mrc400, dat, errors.New("3004,MRC400 [" + mrc400id + "] is in the wrong data")
	}
	if mrc400.Id == "" {
		mrc400.Id = mrc400id
	}
	return mrc400, dat, nil
}

func setMRC400(stub shim.ChaincodeStubInterface, MRC400 TMRC400, jobType string, jobArgs []string) error {
	var err error
	var argdat []byte

	if strings.Index(MRC400.Id, "MRC400_") != 0 || len(MRC400.Id) != 40 {
		return errors.New("6102,invalid MRC401 data address")
	}

	MRC400.JobType = jobType
	MRC400.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal(jobArgs); err == nil {
		MRC400.JobArgs = string(argdat)
	}

	if argdat, err = json.Marshal(MRC400); err != nil {
		return errors.New("3209,Invalid MRC401ItemData data format")
	}

	if err := stub.PutState(MRC400.Id, argdat); err != nil {
		return errors.New("8600,Mrc401Create stub.PutState [" + MRC400.Id + "] Error " + err.Error())
	}
	return nil
}

// GetMRC401 - get MRC401 token
func GetMRC401(stub shim.ChaincodeStubInterface, mrc401id string) (TMRC401, []byte, error) {
	var dat []byte
	var err error
	var mrc401 TMRC401

	if strings.Index(mrc401id, "MRC400_") != 0 || len(mrc401id) != 81 {
		return mrc401, nil, errors.New("6102,invalid MRC401 data address")
	}

	dat, err = stub.GetState(mrc401id)
	if err != nil {
		return mrc401, nil, errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return mrc401, nil, errors.New("6004,MRC401 [" + mrc401id + "] not exist")
	}
	if err = json.Unmarshal(dat, &mrc401); err != nil {
		return mrc401, nil, errors.New("3004,MRC401 [" + mrc401id + "] is in the wrong data")
	}
	if mrc401.Id == "" {
		mrc401.Id = mrc401id
	}

	return mrc401, dat, nil
}

func setMRC401(stub shim.ChaincodeStubInterface, MRC401 TMRC401, jobType string, jobArgs []string) error {
	var err error
	var argdat []byte

	if strings.Index(MRC401.Id, "MRC400_") != 0 || len(MRC401.Id) != 81 {
		return errors.New("6102,invalid MRC401 data address")
	}

	if len(MRC401.MRC400) != 40 {
		return errors.New("6102,invalid MRC401 ID")
	}
	if strings.Index(MRC401.Id, MRC401.MRC400) != 0 {
		return errors.New("6102,invalid MRC401 ID")
	}

	MRC401.JobType = jobType
	MRC401.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal(jobArgs); err == nil {
		MRC401.JobArgs = string(argdat)
	}

	if argdat, err = json.Marshal(MRC401); err != nil {
		return errors.New("3209,Invalid MRC401ItemData data format")
	}

	if err := stub.PutState(MRC401.Id, argdat); err != nil {
		return errors.New("8600,Mrc401Create stub.PutState [" + MRC401.Id + "] Error " + err.Error())
	}
	return nil
}

// Mrc401Create MRC401 create
func Mrc401Create(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC401Creator mtc.TWallet
	var now int64
	var MRC400 TMRC400
	var createTotal map[string]decimal.Decimal
	var tempPrice decimal.Decimal
	var MRC401JobList []TMRC401CreateUpdate
	var MRC401ItemData TMRC401
	var data []byte
	var buf string
	var logData []TMRC401Sell
	var keyCheck map[string]int
	var MRC401 TMRC401

	if len(args) < 5 {
		return errors.New("1000,mrc401create operation must include four arguments : " +
			"mrc400id, creator, itemData, signature, nonce")
	}

	// get project(mrc400)
	if MRC400, _, err = GetMRC400(stub, args[0]); err != nil {
		return err
	}

	// permission check
	if err = Mrc401PermissionCheck(MRC400, MRC401, args[1], MRC401AT_Create); err != nil {
		return err
	}

	if MRC401Creator, err = GetAddressInfo(stub, args[1]); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&MRC401Creator, args[4],
		strings.Join([]string{args[0], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(args[2]), &MRC401JobList); err != nil {
		return errors.New("6205,MRC401 Data is in the wrong data - " + err.Error())
	}
	if len(MRC401JobList) > 1024 {
		return errors.New("3002,There must be 1024 or fewer create item")
	}
	if len(MRC401JobList) < 1 {
		return errors.New("3002,There is no item information")
	}
	createTotal = make(map[string]decimal.Decimal)
	now = time.Now().Unix()
	keyCheck = make(map[string]int)
	for index := range MRC401JobList {

		if _, exists := keyCheck[MRC401JobList[index].ItemID]; exists {
			return errors.New("3004,MRC401 [" + MRC401JobList[index].ItemID + "] is duplicate")
		}
		keyCheck[MRC401JobList[index].ItemID] = 0

		data, err = stub.GetState(MRC400.Id + "_" + MRC401JobList[index].ItemID)
		if err != nil {
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if data != nil {
			return errors.New("8600,Item ID " + MRC401JobList[index].ItemID + " already exists in project " + MRC400.Id)
		}

		// init data
		MRC401ItemData = TMRC401{
			Id:                   MRC400.Id + "_" + MRC401JobList[index].ItemID,
			MRC400:               MRC400.Id,
			Owner:                MRC400.Owner,
			Creator:              MRC401Creator.Id,
			ItemURL:              "",
			ItemImageURL:         "",
			GroupID:              "",
			CreateDate:           now,
			InititalReserve:      "0",
			InititalToken:        "0",
			MeltingFee:           "0.0",
			Transferable:         "Permanent",
			SellFee:              "0.0",
			SellPrice:            "0",
			SellToken:            "0",
			SellDate:             0,
			MeltingDate:          0,
			JobType:              "mrc401_create",
			JobArgs:              "",
			JobDate:              now,
			AuctionDate:          0,
			AuctionEnd:           0,
			AuctionToken:         "0",
			AuctionBiddingUnit:   "0",
			AuctionStartPrice:    "0",
			AuctionBuyNowPrice:   "0",
			AuctionCurrentPrice:  "0",
			AuctionCurrentBidder: "0",
			LastTradeDate:        0,
			LastTradeAmount:      "0",
			LastTradeToken:       "0",
			LastTradeType:        "",
			ShareHolder:          nil,
			PlatformName:         "",
			PlatformURL:          "",
			PlatformAddress:      "",
			PlatformCommission:   "",
		}

		// param check
		if err = util.DataAssign(MRC401JobList[index].ItemID, &MRC401JobList[index].ItemID, "id", 40, 40, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item ItemID error : " + err.Error())
		}

		if err = util.DataAssign(MRC401JobList[index].ItemURL, &MRC401ItemData.ItemURL, "url", 1, 255, true); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item ItemURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401JobList[index].ItemImageURL, &MRC401ItemData.ItemImageURL, "url", 1, 255, true); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item ItemImageURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401JobList[index].GroupID, &MRC401ItemData.GroupID, "string", 1, 40, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item GroupID error : " + err.Error())
		}

		if err = util.NumericDataCheck(MRC401JobList[index].InititalReserve, &MRC401ItemData.InititalReserve, "0", "9999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item InititalReserve error : " + err.Error())
		}

		if err = util.DataAssign(MRC401JobList[index].InititalToken, &MRC401ItemData.InititalToken, "string", 1, 40, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item InititalToken error : " + err.Error())
		}

		if err = util.NumericDataCheck(MRC401JobList[index].MeltingFee, &MRC401ItemData.MeltingFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item MeltingFee error : " + err.Error())
		}

		if err = util.DataAssign(MRC401JobList[index].Transferable, &MRC401ItemData.Transferable, "string", 1, 128, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item Transferable error : " + err.Error())
		}

		if MRC401ItemData.Transferable != "Permanent" && MRC401ItemData.Transferable != "Bound" && MRC401ItemData.Transferable != "Temporary" {
			return errors.New("3005," + util.GetOrdNumber(index) + " item Transferable value is Permanent, Bound, Temporary ")
		}

		if err = util.NumericDataCheck(MRC401JobList[index].SellFee, &MRC401ItemData.SellFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellFee error : " + err.Error())
		}

		// Initital token check
		if MRC400.AllowToken == "" {
			MRC400.AllowToken = "0"
		}
		if MRC400.AllowToken != "0" && MRC400.AllowToken != "" {
			if MRC401ItemData.InititalToken != MRC400.AllowToken && MRC401ItemData.InititalToken != "0" {
				return errors.New("3005," + util.GetOrdNumber(index) + " item Token is must " + MRC400.AllowToken)
			}
		}

		if tempPrice, err = decimal.NewFromString(MRC401ItemData.InititalReserve); err != nil {
			return errors.New("3209," + util.GetOrdNumber(index) + " item Invalid InititalReserve")
		}
		if tempPrice.IsPositive() {
			createTotal[MRC401ItemData.InititalToken] = createTotal[MRC401ItemData.InititalToken].Add(tempPrice).Truncate(0)
		}

		if len(MRC401JobList[index].ShareHolder) > 0 {
			if len(MRC401JobList[index].ShareHolder) > 5 {
				return errors.New("3002,There must be 5 or fewer copyrighter")
			}
			index := 0
			for shareholder, commission := range MRC401JobList[index].ShareHolder {
				if shareholder == MRC400.Owner {
					return errors.New("3005," + util.GetOrdNumber(index) + " item shareholder The owner's address and the copyright holder's address are the same")
				}
				if _, err = GetAddressInfo(stub, shareholder); err != nil {
					return errors.New("3005," + util.GetOrdNumber(index) + " item shareholder error : " + err.Error())
				}
				if err = util.NumericDataCheck(commission, &buf, "0", "10.0", 2, false); err != nil {
					return errors.New("3005," + util.GetOrdNumber(index) + " item shareholder commission error : " + err.Error())
				}
			}
			MRC401ItemData.ShareHolder = MRC401JobList[index].ShareHolder
		}

		if err = setMRC401(stub, MRC401ItemData, "mrc401_create", []string{MRC401ItemData.Id,
			args[1], MRC401ItemData.InititalReserve, MRC401ItemData.InititalToken, args[3], args[4]}); err != nil {
			return err
		}

		// MRC401Sell for NFT ITEM sell
		logData = append(logData, TMRC401Sell{ItemID: MRC401JobList[index].ItemID, SellPrice: MRC401ItemData.InititalReserve, SellToken: MRC401ItemData.InititalToken})
	}

	// subtract token for item initial price
	for token, totPrice := range createTotal {
		if totPrice.IsPositive() {
			if err = MRC010Subtract(stub, &MRC401Creator, token, totPrice.String(), MRC010MT_Normal); err != nil {
				return err
			}
		}
	}

	// save create info
	// - for update balance
	// - for nonce update
	if err = SetAddressInfo(stub, MRC401Creator, "mrc401create", []string{MRC400.Id, args[1],
		util.JSONEncode(logData), args[3], args[4]}); err != nil {
		return err
	}
	return nil
}

// Mrc401Update MRC401 update
func Mrc401Update(stub shim.ChaincodeStubInterface, args []string) error {
	var MRC400 TMRC400
	var MRC400Partner mtc.TWallet
	var MRC401 TMRC401
	var MRC401Job []TMRC401CreateUpdate
	var logData []string
	var err error

	var keyCheck map[string]int

	if len(args) < 5 {
		return errors.New("1000,mrc401update operation must include four arguments : " +
			"mrc400id, updater, itemData, signature, nonce")
	}

	// get project(mrc400)
	if MRC400, _, err = GetMRC400(stub, args[0]); err != nil {
		return err
	}

	// get updator
	if MRC400Partner, err = GetAddressInfo(stub, args[1]); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&MRC400Partner, args[4],
		strings.Join([]string{args[0], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(args[2]), &MRC401Job); err != nil {
		return errors.New("6205,MRC401 Data is in the wrong data - " + err.Error())
	}
	if len(MRC401Job) > 100 {
		return errors.New("3002,There must be 100 or fewer update item")
	}
	if len(MRC401Job) < 1 {
		return errors.New("3002,There is no item information")
	}

	keyCheck = make(map[string]int)
	for index := range MRC401Job {
		if _, exists := keyCheck[MRC401Job[index].ItemID]; exists {
			return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is duplicate")
		}
		keyCheck[MRC401Job[index].ItemID] = 0

		if MRC401, _, err = GetMRC401(stub, MRC400.Id+"_"+MRC401Job[index].ItemID); err != nil {
			return err
		}
		// permission check
		if err = Mrc401PermissionCheck(MRC400, MRC401, args[1], MRC401AT_Update); err != nil {
			return err
		}

		if MRC401.MeltingDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already melted")
		}

		if MRC401.SellDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already sale")
		}

		if MRC401.AuctionDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already auction")
		}

		if err = util.DataAssign(MRC401Job[index].ItemURL, &MRC401.ItemURL, "url", 0, 255, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item ItemURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].ItemImageURL, &MRC401.ItemImageURL, "url", 0, 255, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item ItemImageURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].GroupID, &MRC401.GroupID, "string", 0, 40, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item GroupID error : " + err.Error())
		}

		if err = util.NumericDataCheck(MRC401Job[index].SellFee, &MRC401.SellFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item SellFee error : " + err.Error())
		}
		if err = util.NumericDataCheck(MRC401Job[index].MeltingFee, &MRC401.MeltingFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item MeltingFee error : " + err.Error())
		}

		if MRC401.Transferable != "Temporary" && MRC401.Transferable != "Temprary" {
			if MRC401Job[index].Transferable != MRC401.Transferable {
				return errors.New("3005," + MRC401Job[index].ItemID + " item Transferable value cannot be change")
			}
		} else {
			if err = util.DataAssign(MRC401Job[index].Transferable, &MRC401.Transferable, "string", 1, 128, false); err != nil {
				return errors.New("3005," + MRC401Job[index].ItemID + " item Transferable error : " + err.Error())
			}

			if MRC401.Transferable != "Permanent" && MRC401.Transferable != "Bound" && MRC401.Transferable != "Temporary" {
				return errors.New("3005," + MRC401Job[index].ItemID + " item Transferable value is Permanent, Bound, Temporary")
			}
		}

		if err = setMRC401(stub, MRC401, "mrc401_update", []string{MRC401.Id,
			args[1], MRC401.InititalReserve, MRC401.InititalToken, args[3], args[4]}); err != nil {
			return err
		}
		logData = append(logData, MRC401Job[index].ItemID)
	}

	// save create info
	// - for update balance
	// - for nonce update

	if err = SetAddressInfo(stub, MRC400Partner, "mrc401update", []string{MRC400.Id, MRC400.Owner,
		util.JSONEncode(logData),
		args[3], args[4]}); err != nil {
		return err
	}
	return nil
}

func Mrc401CreateTrade(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC401Creator mtc.TWallet
	var now int64
	var MRC400 TMRC400
	var MRC401 TMRC401

	var tempPrice decimal.Decimal
	var MRC401Job TMRC401CreateUpdate
	var buyerWallet mtc.TWallet
	var PaymentInfo []mtc.TDexPaymentInfo

	var data []byte
	var buf string

	if len(args) < 14 {
		return errors.New("1000,mrc401createtrade operation must include four arguments : " +
			"mrc400id, creator, itemData, buyer, price, " +
			"token, platform_name, platform_url, platform_address, platform_commission, " +
			"creatorSignature, creatorNonce, buyerSignature, buyerNonce")
	}

	// args 0
	if MRC400, _, err = GetMRC400(stub, args[0]); err != nil {
		return err
	}

	// permission check
	if err = Mrc401PermissionCheck(MRC400, MRC401, args[1], MRC401AT_CreateTrade); err != nil {
		return err
	}

	if MRC401Creator, err = GetAddressInfo(stub, args[1]); err != nil {
		return err
	}

	// args 3, 4 : sign check
	// "mrc400id, creator, itemData, creatorSignature, creatorNonce, " +
	// "buyer, price, token, buyerSignature, buyerNonce")
	if err = NonceCheck(&MRC401Creator, args[11],
		strings.Join([]string{args[0], args[2], args[3], args[4], args[5], args[11]}, "|"),
		args[10]); err != nil {
		return err
	}

	// args 2
	// MRC401 data parsing
	if err = json.Unmarshal([]byte(args[2]), &MRC401Job); err != nil {
		return errors.New("6205,MRC401 Data is in the wrong data - " + err.Error())
	}

	// args 5
	if buyerWallet, err = GetAddressInfo(stub, args[3]); err != nil {
		return err
	}
	// args 7, 8 : buyer sign check
	if err = NonceCheck(&buyerWallet, args[13],
		strings.Join([]string{args[0], args[2], args[3], args[4], args[5], args[13]}, "|"),
		args[12]); err != nil {
		return err
	}

	// args 6 trade price
	if err = util.NumericDataCheck(args[4], &buf, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
		return errors.New("3005,SellPrice error : " + err.Error())
	}

	// args 7
	if _, _, err = GetMRC010(stub, args[5]); err != nil {
		return errors.New("3005,Token id " + args[5] + " error : " + err.Error())
	}

	// block self trade
	if buyerWallet.Id == MRC400.Owner {
		return errors.New("3004,MRC400 owner cannot purchase items sold by yourself")
	}
	// block self trade
	if buyerWallet.Id == MRC401Creator.Id {
		return errors.New("3004,MRC401 creator cannot purchase items sold by yourself")
	}

	// ================================================
	// MRC401 Create
	// ================================================
	now = time.Now().Unix()

	// MRC401 exists check
	data, err = stub.GetState(MRC400.Id + "_" + MRC401Job.ItemID)
	if err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}

	if data != nil {
		return errors.New("8600,Item ID " + MRC401Job.ItemID + " already exists in project " + MRC400.Id)
	}

	// init data
	MRC401 = TMRC401{
		Id:                   MRC400.Id + "_" + MRC401Job.ItemID,
		MRC400:               MRC400.Id,
		Owner:                MRC400.Owner,
		Creator:              MRC401Creator.Id,
		ItemURL:              "",
		ItemImageURL:         "",
		GroupID:              "",
		CreateDate:           now,
		InititalReserve:      "0",
		InititalToken:        "0",
		MeltingFee:           "0.0",
		Transferable:         "Permanent",
		SellFee:              "0.0",
		SellPrice:            "0",
		SellToken:            "0",
		SellDate:             0,
		MeltingDate:          0,
		JobType:              "mrc401_createtrade",
		JobArgs:              "",
		JobDate:              now,
		AuctionDate:          0,
		AuctionEnd:           0,
		AuctionToken:         "0",
		AuctionBiddingUnit:   "0",
		AuctionStartPrice:    "0",
		AuctionBuyNowPrice:   "0",
		AuctionCurrentPrice:  "0",
		AuctionCurrentBidder: "0",
		LastTradeDate:        0,
		LastTradeAmount:      "0",
		LastTradeToken:       "0",
		LastTradeType:        "",
		ShareHolder:          nil,
		PlatformName:         "",
		PlatformURL:          "",
		PlatformAddress:      "",
		PlatformCommission:   "",
	}

	// param check
	if err = util.DataAssign(MRC401Job.ItemID, &MRC401Job.ItemID, "id", 40, 40, false); err != nil {
		return errors.New("3005,ItemID error : " + err.Error())
	}

	if err = util.DataAssign(MRC401Job.ItemURL, &MRC401.ItemURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,ItemURL error : " + err.Error())
	}

	if err = util.DataAssign(MRC401Job.ItemImageURL, &MRC401.ItemImageURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,ItemImageURL error : " + err.Error())
	}

	if err = util.DataAssign(MRC401Job.GroupID, &MRC401.GroupID, "string", 1, 40, false); err != nil {
		return errors.New("3005,GroupID error : " + err.Error())
	}

	if err = util.NumericDataCheck(MRC401Job.InititalReserve, &MRC401.InititalReserve, "0", "9999999999999999999999999999999999999999", 0, false); err != nil {
		return errors.New("3005,InititalReserve error : " + err.Error())
	}

	if err = util.DataAssign(MRC401Job.InititalToken, &MRC401.InititalToken, "string", 1, 40, false); err != nil {
		return errors.New("3005,InititalToken error : " + err.Error())
	}

	if err = util.NumericDataCheck(MRC401Job.MeltingFee, &MRC401.MeltingFee, "0", "99.9999", 4, false); err != nil {
		return errors.New("3005,MeltingFee error : " + err.Error())
	}

	if err = util.DataAssign(MRC401Job.Transferable, &MRC401.Transferable, "string", 1, 128, false); err != nil {
		return errors.New("3005,Transferable error : " + err.Error())
	}

	if MRC401.Transferable != "Permanent" && MRC401.Transferable != "Bound" && MRC401.Transferable != "Temporary" {
		return errors.New("3005,Transferable value is Permanent, Bound, Temporary ")
	}

	if err = util.NumericDataCheck(MRC401Job.SellFee, &MRC401.SellFee, "0", "99.9999", 4, false); err != nil {
		return errors.New("3005,SellFee error : " + err.Error())
	}

	// Initital token check
	if MRC400.AllowToken == "" {
		MRC400.AllowToken = "0"
	}

	// initial reserve check.
	if tempPrice, err = decimal.NewFromString(MRC401.InititalReserve); err != nil {
		return errors.New("3209,Invalid InititalReserve")
	}
	if tempPrice.IsPositive() {
		if MRC400.AllowToken != "0" && MRC400.AllowToken != "" {
			if MRC401.InititalToken != MRC400.AllowToken && MRC401.InititalToken != "0" {
				return errors.New("3005,Token is must " + MRC400.AllowToken)
			}
		}

		if err = MRC010Subtract(stub, &MRC401Creator, MRC401.InititalToken, tempPrice.String(), MRC010MT_Normal); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: MRC401Creator.Id, ToAddr: MRC401.Id,
			Amount: tempPrice.String(), TokenID: MRC401.InititalToken, TradeAmount: "", TradeID: "", PayType: "mrc401_createtrade"})
	}

	if len(MRC401Job.ShareHolder) > 0 {
		if len(MRC401Job.ShareHolder) > 5 {
			return errors.New("3002,There must be 5 or fewer copyrighter")
		}
		index := 0
		for shareholder, commission := range MRC401Job.ShareHolder {
			if shareholder == MRC400.Owner {
				return errors.New("3005," + util.GetOrdNumber(index) + " item shareholder The owner's address and the copyright holder's address are the same")
			}
			if _, err = GetAddressInfo(stub, shareholder); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " shareholder item error : " + err.Error())
			}
			if err = util.NumericDataCheck(commission, &buf, "0", "10.0", 2, false); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " shareholder item commission error : " + err.Error())
			}
			index++
		}
		MRC401.ShareHolder = MRC401Job.ShareHolder
	}

	// =======================
	// sell process
	// ===========================

	// item transferable ?
	if MRC401.Transferable == "Bound" {
		// allow owner sale.
		if MRC401.Owner != MRC400.Owner {
			return errors.New("5002,MRC401 [" + MRC401.Id + "] is cannot be sold")
		}
	}

	// sell price check
	if err = util.NumericDataCheck(args[4], &MRC401.SellPrice, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
		return errors.New("3005,SellPrice error : " + err.Error())
	}

	//  token check
	if MRC400.AllowToken != "0" && MRC400.AllowToken != "" {
		if args[5] != MRC400.AllowToken && args[5] != "0" {
			return errors.New("3005,item SellToken is must " + MRC400.AllowToken + " or metacoin")
		}
	}

	// Platform Name
	if err = util.DataAssign(args[6], &MRC401.PlatformName, "string", 1, 255, true); err != nil {
		return errors.New("3005,InititalToken error : " + err.Error())
	}

	// Platform URL
	if err = util.DataAssign(args[7], &MRC401.PlatformURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,InititalToken error : " + err.Error())
	}

	// Platform Address
	if err = util.DataAssign(args[8], &MRC401.PlatformAddress, "address", 40, 40, true); err != nil {
		return errors.New("3005,InititalToken error : " + err.Error())
	}
	if util.IsAddress(MRC401.PlatformAddress) {
		if _, err = GetAddressInfo(stub, MRC401.PlatformAddress); err != nil {
			return errors.New("3005," + "PlatformAddress not found : " + err.Error())
		}
	}
	// Platform Commission
	if err = util.NumericDataCheck(args[9], &MRC401.PlatformCommission, "0.00", "10.00", 2, true); err != nil {
		return errors.New("3005,InititalToken error : " + err.Error())
	}

	// set sell token
	MRC401.SellToken = args[5]
	MRC401.SellDate = now

	// save item
	MRC401.LastWorker = MRC401Creator.Id

	// =========================
	// buy process
	// =========================

	// set payment info 1st - buy(buyer => mrc401)
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{
		FromAddr: buyerWallet.Id, ToAddr: MRC401.Id,
		Amount: MRC401.SellPrice, TokenID: MRC401.SellToken, PayType: "mrc401_createtrade_buy"})

	return mrc401DexProcess(stub, MRC400, MRC401, MRC401Creator, buyerWallet, PaymentInfo, MRC401AT_CreateTrade, args[10], args[11])
}

// Mrc401Transfer transfer
func Mrc401Transfer(stub shim.ChaincodeStubInterface, mrc401id, fromAddr, toAddr, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.TWallet
	var MRC401 TMRC401
	var MRC400 TMRC400

	// get item
	if MRC401, _, err = GetMRC401(stub, mrc401id); err != nil {
		return err
	}

	// item transferable ?
	if MRC401.Transferable == "Bound" {
		// get project
		if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
			return err
		}
		if MRC401.Owner != MRC400.Owner {
			return errors.New("5002,MRC401 [" + mrc401id + "] is not transferable")
		}
	}

	if MRC401.SellDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already sale")
	}

	if MRC401.AuctionDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already auction")
	}

	if MRC401.Owner != fromAddr {
		return errors.New("3004,MRC401 [" + mrc401id + "] is not your item")
	}

	if toAddr == fromAddr {
		return errors.New("3005,From address and to address must be different values")
	}

	// get owner info
	if ownerWallet, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}

	// get owner info
	if _, err = GetAddressInfo(stub, toAddr); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{fromAddr, toAddr, mrc401id, tkey}, "|"),
		signature); err != nil {
		return err
	}

	// item owner change
	MRC401.Owner = toAddr
	// Change LastTradeType
	if MRC401.LastTradeType == "" {
		MRC401.LastTradeType = "Transfer"
	}

	// save MRC401
	if err := setMRC401(stub, MRC401, "mrc401_transfer", args); err != nil {
		return err
	}

	// save prev owner info for nonce update
	if err = SetAddressInfo(stub, ownerWallet, "mrc401transfer", []string{mrc401id, fromAddr, toAddr, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401Sell Mrc401Sell
func Mrc401Sell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var now int64

	var MRC401Seller mtc.TWallet
	var MRC401 TMRC401
	var MRC400 TMRC400

	var MRC401SellData []TMRC401Sell
	var logData []TMRC401Sell
	var keyCheck map[string]int

	if len(args) < 4 {
		return errors.New("1000,mrc401sell operation must include four arguments : seller, itemData, sign, tkey")
	}

	// seller, itemData, signature, tkey string,
	if err = json.Unmarshal([]byte(args[1]), &MRC401SellData); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401SellData) > 1000 {
		return errors.New("3002,There must be 1000 or fewer sell item")
	}
	if len(MRC401SellData) < 1 {
		return errors.New("3002,There is no item information")
	}
	// get seller info
	if MRC401Seller, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&MRC401Seller, args[3],
		strings.Join([]string{args[0], args[1], args[3]}, "|"),
		args[2]); err != nil {
		return err
	}

	now = time.Now().Unix()

	keyCheck = make(map[string]int)
	for index := range MRC401SellData {
		// item duplication check
		if _, exists := keyCheck[MRC401SellData[index].ItemID]; exists {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is duplicate")
		}
		keyCheck[MRC401SellData[index].ItemID] = 0

		// get MRC401
		if MRC401, _, err = GetMRC401(stub, MRC401SellData[index].ItemID); err != nil {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] - " + err.Error())
		}

		// get MRC400
		if MRC400.Id != MRC401.MRC400 {
			if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
				return err
			}
		}

		// permission check
		if err = Mrc401PermissionCheck(MRC400, MRC401, args[0], MRC401AT_Sell); err != nil {
			return err
		}

		// item is sell or auction ?
		if MRC401.SellDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is already sale")
		}
		if MRC401.AuctionDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is already auction")
		}

		// item transferable ?
		if MRC401.Transferable == "Bound" {
			// allow owner sale.
			if MRC401.Owner != MRC400.Owner {
				return errors.New("5002,MRC401 [" + MRC401SellData[index].ItemID + "] is cannot be sold")
			}
		}

		// sell price check
		if err = util.NumericDataCheck(MRC401SellData[index].SellPrice, &MRC401.SellPrice, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellPrice error : " + err.Error())
		}

		//  token check
		if MRC401SellData[index].SellToken == "" {
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellToken not define")
		}
		if MRC400.AllowToken != "0" && MRC400.AllowToken != "" {
			if MRC401SellData[index].SellToken != MRC400.AllowToken && MRC401SellData[index].SellToken != "0" {
				return errors.New("3005," + util.GetOrdNumber(index) + " item SellToken is must " + MRC400.AllowToken + " or metacoin")
			}
		}

		// clear platform info
		MRC401.PlatformName = ""
		MRC401.PlatformURL = ""
		MRC401.PlatformAddress = ""
		MRC401.PlatformCommission = ""

		// Platform Name
		if err = util.DataAssign(MRC401SellData[index].PlatformName, &MRC401.PlatformName, "string", 1, 255, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}

		// Platform URL
		if err = util.DataAssign(MRC401SellData[index].PlatformURL, &MRC401.PlatformURL, "url", 1, 255, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}

		// Platform Address
		if err = util.DataAssign(MRC401SellData[index].PlatformAddress, &MRC401.PlatformAddress, "address", 40, 40, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}
		if util.IsAddress(MRC401.PlatformAddress) {
			if _, err = GetAddressInfo(stub, MRC401.PlatformAddress); err != nil {
				return errors.New("3005," + "PlatformAddress not found : " + err.Error())
			}
		}
		// Platform Commission
		if err = util.NumericDataCheck(MRC401SellData[index].PlatformCommission, &MRC401.PlatformCommission, "0.00", "10.00", 2, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}

		// set sell token
		MRC401.SellToken = MRC401SellData[index].SellToken
		MRC401.SellDate = now

		// save item
		MRC401.LastWorker = MRC401Seller.Id
		if err = setMRC401(stub, MRC401, "mrc401_sell", []string{MRC401SellData[index].ItemID,
			args[0], MRC401SellData[index].SellPrice, MRC401SellData[index].SellToken, args[2], args[3]}); err != nil {
			return err
		}
		logData = append(logData, TMRC401Sell{
			ItemID:             MRC401SellData[index].ItemID,
			SellPrice:          MRC401SellData[index].SellPrice,
			SellToken:          MRC401SellData[index].SellToken,
			PlatformName:       MRC401SellData[index].PlatformName,
			PlatformURL:        MRC401SellData[index].PlatformURL,
			PlatformAddress:    MRC401SellData[index].PlatformAddress,
			PlatformCommission: MRC401SellData[index].PlatformCommission,
		})
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, MRC401Seller, "mrc401sell", []string{util.JSONEncode(logData), args[0], args[2], args[3]}); err != nil {
		return err
	}
	return nil
}

// Mrc401UnSell Mrc401UnSell
func Mrc401UnSell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC401Seller mtc.TWallet
	var MRC401list []string
	var MRC400 TMRC400
	var MRC401 TMRC401
	var logData []string

	// seller, itemData, signature, tkey string,
	if len(args) < 4 {
		return errors.New("1000,mrc401unsell operation must include four arguments : seller, itemData, sign, tkey")
	}

	// get seller info
	if MRC401Seller, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&MRC401Seller, args[3],
		strings.Join([]string{args[0], args[1], args[3]}, "|"),
		args[2]); err != nil {
		return err
	}

	// 0 item data
	if err = json.Unmarshal([]byte(args[1]), &MRC401list); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401list) > 1000 {
		return errors.New("3002,There must be 1000 or fewer unsell item")
	}
	if len(MRC401list) < 1 {
		return errors.New("3002,There is no item information")
	}

	for index := range MRC401list {
		if MRC401, _, err = GetMRC401(stub, MRC401list[index]); err != nil {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] - " + err.Error())
		}
		if MRC400.Id != MRC401.MRC400 {
			if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
				return err
			}
		}

		if err = Mrc401PermissionCheck(MRC400, MRC401, MRC401Seller.Id, MRC401AT_UnSell); err != nil {
			return err
		}

		// item is sell ?
		if MRC401.SellDate == 0 {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not sale")
		}

		// sell info clear
		MRC401.SellDate = 0
		MRC401.SellPrice = "0"
		MRC401.SellToken = "0"

		// platform info clear
		MRC401.PlatformAddress = ""
		MRC401.PlatformCommission = ""
		MRC401.PlatformName = ""
		MRC401.PlatformURL = ""

		// set Last worker.
		MRC401.LastWorker = MRC401Seller.Id

		// save MRC401
		if err = setMRC401(stub, MRC401, "mrc401_unsell",
			[]string{MRC401list[index], args[0], args[2], args[3]}); err != nil {
			return err
		}
		logData = append(logData, MRC401list[index])
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, MRC401Seller, "mrc401unsell", []string{util.JSONEncode(logData), args[0], args[2], args[3]}); err != nil {
		return err
	}
	return nil
}

// Mrc401Buy Mrc401Buy
func Mrc401Buy(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC400 TMRC400
	var MRC401 TMRC401

	var buyerWallet mtc.TWallet
	var PaymentInfo []mtc.TDexPaymentInfo

	if len(args) < 4 {
		return errors.New("1000,mrc401buy operation must include four arguments : " +
			"buyer, mrc401id, sign, tkey")
	}

	// buyer, mrc401id, signature, tkey string,
	// sign check
	if buyerWallet, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}
	if err = NonceCheck(&buyerWallet, args[3],
		strings.Join([]string{args[1], args[3]}, "|"),
		args[2]); err != nil {
		return err
	}

	// get item info
	if MRC401, _, err = GetMRC401(stub, args[1]); err != nil {
		return err
	}
	// item is sell ??
	if MRC401.SellDate == 0 {
		return errors.New("3004,MRC401 [" + MRC401.Id + "] is not for sale")
	}
	// block self trade
	if buyerWallet.Id == MRC401.Owner {
		return errors.New("3004,You cannot purchase items sold by yourself")
	}
	// get Project
	if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
		return err
	}

	// set payment info 1st - buy(buyer => mrc401)
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{
		FromAddr: buyerWallet.Id, ToAddr: MRC401.Id,
		Amount: MRC401.SellPrice, TokenID: MRC401.SellToken, PayType: "mrc401_buy"})

	return mrc401DexProcess(stub, MRC400, MRC401, mtc.TWallet{}, buyerWallet, PaymentInfo, MRC401AT_Buy, args[2], args[3])
}

// Mrc401Melt Mrc401Melt
func Mrc401Melt(stub shim.ChaincodeStubInterface, mrc401id, signature, tkey string, args []string) error {
	var err error
	var itemOwnerWallet mtc.TWallet
	var projectOwnerWallet mtc.TWallet
	var MRC401ItemData TMRC401
	var MRC400ProjectData TMRC400
	var itemOwner string

	var PaymentInfo []mtc.TDexPaymentInfo

	var InititalPrice decimal.Decimal //  The amount given by the creator when creating an item
	var feeRate decimal.Decimal       // Melting Fee(percents)    100% == 100
	var Percent decimal.Decimal       // "100"  (Price * feeRate / Percent)
	var receivePrice decimal.Decimal  // The amount the owner will receive
	var feePrice decimal.Decimal      // The amount the creator will receive

	// get item info
	if MRC401ItemData, _, err = GetMRC401(stub, mrc401id); err != nil {
		return err
	}

	if MRC401ItemData.SellDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already sale")
	}

	if MRC401ItemData.AuctionDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already auction")
	}

	itemOwner = MRC401ItemData.Owner
	if itemOwner == "MELTED" {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already melted")
	}

	// get item owner info
	if itemOwnerWallet, err = GetAddressInfo(stub, itemOwner); err != nil {
		return err
	}

	// sign check.
	if err = NonceCheck(&itemOwnerWallet, tkey,
		strings.Join([]string{mrc401id, tkey}, "|"),
		signature); err != nil {
		return err
	}

	InititalPrice, _ = decimal.NewFromString(MRC401ItemData.InititalReserve)
	if InititalPrice.IsPositive() {
		// get Project
		if MRC400ProjectData, _, err = GetMRC400(stub, MRC401ItemData.MRC400); err != nil {
			return err
		}

		feeRate, _ = decimal.NewFromString(MRC401ItemData.MeltingFee)
		Percent, _ = decimal.NewFromString("100")
		feePrice = InititalPrice.Mul(feeRate).Div(Percent).Floor()
		receivePrice = InititalPrice.Sub(feePrice)

		if feePrice.IsPositive() {
			// set paymentinfo info - 1st melt fee(mrc401 -> project owner)
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: mrc401id, ToAddr: MRC400ProjectData.Owner,
				Amount: feePrice.String(), TokenID: MRC401ItemData.InititalToken, PayType: "mrc401_recv_meltfee"})
			if MRC400ProjectData.Owner != itemOwner {
				// get Proejct Owner
				if projectOwnerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
					return err
				}

				// Add melt fee
				if err = MRC010Add(stub, &projectOwnerWallet, MRC401ItemData.InititalToken, feePrice.String(), 0); err != nil {
					return err
				}
				// Save Project Owner
				if err = SetAddressInfo(stub, projectOwnerWallet, "receive_meltfee", []string{mrc401id, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.InititalToken, signature, "0", "", mrc401id, tkey}); err != nil {
					return err
				}
			}
		}

		if receivePrice.IsPositive() {
			// set paymentinfo info - 2nd recv initial amount(mrc401 -> item owner)
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: mrc401id, ToAddr: itemOwner,
				Amount: receivePrice.String(), TokenID: MRC401ItemData.InititalToken, PayType: "mrc401_recv_melt"})

			// save to item owner
			if MRC400ProjectData.Owner == itemOwner {
				// add remain price
				if err = MRC010Add(stub, &itemOwnerWallet, MRC401ItemData.InititalToken, InititalPrice.String(), 0); err != nil {
					return err
				}

				if err = SetAddressInfo(stub, itemOwnerWallet, "receive_melt",
					[]string{mrc401id, itemOwner, InititalPrice.String(), MRC401ItemData.InititalToken, signature, "0", "", mrc401id, tkey}); err != nil {
					return err
				}
			} else {
				// add remain price
				if err = MRC010Add(stub, &itemOwnerWallet, MRC401ItemData.InititalToken, receivePrice.String(), 0); err != nil {
					return err
				}

				if err = SetAddressInfo(stub, itemOwnerWallet, "receive_melt",
					[]string{mrc401id, itemOwner, receivePrice.String(), MRC401ItemData.InititalToken, signature, "0", "", mrc401id, tkey}); err != nil {
					return err
				}
			}
		}
	}

	// item owner change for MELTED
	MRC401ItemData.LastWorker = MRC401ItemData.Owner

	//
	MRC401ItemData.Owner = "MELTED"
	MRC401ItemData.MeltingDate = time.Now().Unix()
	if err = setMRC401(stub, MRC401ItemData, "mrc401_melt",
		[]string{mrc401id, itemOwner, util.JSONEncode(PaymentInfo), signature, tkey}); err != nil {
		return err
	}

	return nil
}

// Mrc401Auction Mrc401Sell
func Mrc401Auction(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var now int64

	var MRC401Seller mtc.TWallet
	var MRC400 TMRC400
	var MRC401 TMRC401

	var MRC401AuctionData []TMRC401Auction

	var logData []TMRC401Sell
	var keyCheck map[string]int

	// seller, itemData, signature, tkey string,
	if len(args) < 4 {
		return errors.New("1000,mrc401auction operation must include four arguments : " +
			"seller, itemData, sign, tkey")
	}

	if err = json.Unmarshal([]byte(args[1]), &MRC401AuctionData); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401AuctionData) > 1000 {
		return errors.New("3002,There must be 1000 or fewer sell item")
	}
	if len(MRC401AuctionData) < 1 {
		return errors.New("3002,There is no item information")
	}
	// get seller info
	if MRC401Seller, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&MRC401Seller, args[3],
		strings.Join([]string{args[0], args[1], args[3]}, "|"),
		args[2]); err != nil {
		return err
	}

	now = time.Now().Unix()

	keyCheck = make(map[string]int)
	for index := range MRC401AuctionData {
		// item duplication check
		if _, exists := keyCheck[MRC401AuctionData[index].ItemID]; exists {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is duplicate")
		}
		keyCheck[MRC401AuctionData[index].ItemID] = 0

		// get MRC401
		if MRC401, _, err = GetMRC401(stub, MRC401AuctionData[index].ItemID); err != nil {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] - " + err.Error())
		}

		// get MRC400
		if MRC400.Id != MRC401.MRC400 {
			if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
				return err
			}
		}

		// Permission check
		if err = Mrc401PermissionCheck(MRC400, MRC401, MRC401Seller.Id, MRC401AT_Auction); err != nil {
			return err
		}

		// sale or auction item ?
		if MRC401.SellDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is already sale")
		}
		if MRC401.AuctionDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is already auction")
		}

		// Platform Name

		if err = util.DataAssign(MRC401AuctionData[index].PlatformName, &MRC401.PlatformName, "string", 1, 255, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}

		// Platform URL
		if err = util.DataAssign(MRC401AuctionData[index].PlatformURL, &MRC401.PlatformURL, "url", 1, 255, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}

		// Platform Address
		if err = util.DataAssign(MRC401AuctionData[index].PlatformAddress, &MRC401.PlatformAddress, "address", 40, 40, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}
		if util.IsAddress(MRC401.PlatformAddress) {
			if _, err = GetAddressInfo(stub, MRC401.PlatformAddress); err != nil {
				return errors.New("3005," + "PlatformAddress not found : " + err.Error())
			}
		}
		// Platform Commission
		if err = util.NumericDataCheck(MRC401AuctionData[index].PlatformCommission, &MRC401.PlatformCommission, "0.00", "10.00", 2, true); err != nil {
			return errors.New("3005,InititalToken error : " + err.Error())
		}

		// item transferable ?
		if MRC401.Transferable == "Bound" {
			if MRC401.Owner != MRC400.Owner {
				return errors.New("5002,MRC401 [" + MRC401AuctionData[index].ItemID + "] is cannot be sold")
			}
		}

		// clear platform info
		MRC401.PlatformName = ""
		MRC401.PlatformURL = ""
		MRC401.PlatformAddress = ""
		MRC401.PlatformCommission = ""

		// start price check
		if err = util.NumericDataCheck(MRC401AuctionData[index].AuctionStartPrice, &MRC401.AuctionStartPrice, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_start_price error : " + err.Error())
		}

		// buynow price check
		if err = util.NumericDataCheck(MRC401AuctionData[index].AuctionBuyNowPrice, &MRC401.AuctionBuyNowPrice, "0", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, true); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_buynow_price error : " + err.Error())
		}

		// bidding unit price check
		if err = util.NumericDataCheck(MRC401AuctionData[index].AuctionBiddingUnit, &MRC401.AuctionBiddingUnit, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_bidding_unit error : " + err.Error())
		}

		auctionStart, _ := decimal.NewFromString(MRC401.AuctionStartPrice)
		auctionBuynow, _ := decimal.NewFromString(MRC401.AuctionBuyNowPrice)
		if !auctionBuynow.IsZero() && auctionBuynow.Cmp(auctionStart) < 0 {
			return errors.New("3005," + util.GetOrdNumber(index) + " item buynow price is must be greater then auction start price")
		}

		if MRC401AuctionData[index].AuctionToken == "" {
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellToken not define")
		}

		//  token check
		if MRC400.AllowToken != "0" && MRC400.AllowToken != "" {
			if MRC401AuctionData[index].AuctionToken != MRC400.AllowToken && MRC401AuctionData[index].AuctionToken != "0" {
				return errors.New("3005," + util.GetOrdNumber(index) + " item SellToken is must " + MRC400.AllowToken + " or metacoin")
			}
		}
		MRC401.AuctionToken = MRC401AuctionData[index].AuctionToken
		MRC401.AuctionCurrentPrice = "0"
		MRC401.AuctionCurrentBidder = ""

		MRC401.AuctionDate = MRC401AuctionData[index].AuctionStartDate
		if MRC401.AuctionDate <= 0 {
			MRC401.AuctionDate = now
		} else if MRC401.AuctionDate < now {
			return errors.New("3005,The auction start time is in the past")
		} else if (MRC401.AuctionDate - now) > 1814400 {
			return errors.New("3005,Auction start time must be within 7 days")
		}

		MRC401.AuctionEnd = MRC401AuctionData[index].AuctionEndDate
		if MRC401.AuctionEnd <= 0 {
			MRC401.AuctionEnd = MRC401.AuctionDate + 86400
		} else if (MRC401.AuctionEnd - MRC401.AuctionDate) < 3600 {
			return errors.New("3005,Auction duration is at least 1 hour")
		} else if (MRC401.AuctionEnd - MRC401.AuctionDate) > 1814400 {
			return errors.New("3005,The auction period is up to 7 days")
		}

		MRC401.LastWorker = MRC401Seller.Id
		// save item
		if err = setMRC401(stub, MRC401, "mrc401_auction", []string{
			MRC401AuctionData[index].ItemID, MRC401Seller.Id, MRC401.AuctionStartPrice, MRC401.AuctionToken, MRC401.AuctionBuyNowPrice,
			MRC401.AuctionBiddingUnit, args[2], args[3]}); err != nil {
			return err
		}
		logData = append(logData, TMRC401Sell{
			ItemID:    MRC401AuctionData[index].ItemID,
			SellPrice: MRC401AuctionData[index].AuctionStartPrice,
			SellToken: MRC401AuctionData[index].AuctionToken})
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, MRC401Seller, "mrc401auction", []string{args[1], args[0], args[2], args[3]}); err != nil {
		return err
	}
	return nil
}

// Mrc401UnAuction Mrc401UnAuction
func Mrc401UnAuction(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC401Seller mtc.TWallet
	var MRC401list []string
	var MRC400 TMRC400
	var MRC401 TMRC401
	var logData []string

	// seller, itemData, signature, tkey string,
	if len(args) < 4 {
		return errors.New("1000,mrc401unsell operation must include four arguments : " +
			"seller, itemData, sign, tkey")
	}

	// get seller info
	if MRC401Seller, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}
	// sign check
	if err = NonceCheck(&MRC401Seller, args[3],
		strings.Join([]string{args[0], args[1], args[3]}, "|"),
		args[2]); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(args[1]), &MRC401list); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401list) > 1000 {
		return errors.New("3002,There must be 1000 or fewer unauction item")
	}
	if len(MRC401list) < 1 {
		return errors.New("3002,There is no item information")
	}

	for index := range MRC401list {
		if MRC401, _, err = GetMRC401(stub, MRC401list[index]); err != nil {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] - " + err.Error())
		}
		if MRC400.Id != MRC401.MRC400 {
			if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
				return err
			}
		}

		if err = Mrc401PermissionCheck(MRC400, MRC401, MRC401Seller.Id, MRC401AT_UnAuction); err != nil {
			return err
		}

		// is auction item ?
		if MRC401.AuctionDate == 0 {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not auction item")
		}

		// bidder exists ?
		if MRC401.AuctionCurrentBidder != "" {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] there is a bidder, so the auction cannot be canceled")
		}

		// clear auction data
		MRC401.AuctionDate = 0
		MRC401.AuctionEnd = 0
		MRC401.AuctionToken = "0"
		MRC401.AuctionBiddingUnit = "0"
		MRC401.AuctionStartPrice = "0"
		MRC401.AuctionBuyNowPrice = "0"
		MRC401.AuctionCurrentPrice = "0"
		MRC401.AuctionCurrentBidder = ""

		// platform info clear
		MRC401.PlatformAddress = ""
		MRC401.PlatformCommission = ""
		MRC401.PlatformName = ""
		MRC401.PlatformURL = ""

		// set Last worker.
		MRC401.LastWorker = MRC401Seller.Id

		// save MRC401
		if err = setMRC401(stub, MRC401, "mrc401_unauction", []string{MRC401list[index], args[0], args[2], args[3]}); err != nil {
			return err
		}
		logData = append(logData, MRC401list[index])
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, MRC401Seller, "mrc401unauction", []string{util.JSONEncode(logData), args[0], args[2], args[3]}); err != nil {
		return err
	}
	return nil
}

// Mrc401AuctionBid Mrc401AuctionBid
func Mrc401AuctionBid(stub shim.ChaincodeStubInterface, args []string) error {
	var now = time.Now().Unix()
	var err error
	var buyerWallet mtc.TWallet
	var MRC401 TMRC401
	var MRC400 TMRC400
	var buyerAddress string
	var refunderWallet mtc.TWallet
	var refunderAddress string

	var PaymentInfo []mtc.TDexPaymentInfo

	var buyNow, oldBidPrice, newBidPrice, bidUnit decimal.Decimal
	var isBuynow bool

	// buyer, mrc401id, amount, signature, tkey string,
	if len(args) < 5 {
		return errors.New("1000,mrc402bid operation must include four arguments : " +
			"buyer, mrc401id, amount, signature, nonce")
	}
	buyerAddress = args[0]

	// get item info
	if MRC401, _, err = GetMRC401(stub, args[1]); err != nil {
		return err
	}

	// status check.
	if MRC401.AuctionDate == 0 {
		return errors.New("3004,MRC401 [" + MRC401.Id + "] is not for auction")
	}
	if MRC401.AuctionEnd < now {
		return errors.New("3004,MRC401 [" + MRC401.Id + "] has completed auction")
	}

	// buyer is seller ?
	if MRC401.Owner == buyerAddress {
		return errors.New("3004,Owners cannot bid on auctions")
	}

	// sign check
	if buyerWallet, err = GetAddressInfo(stub, buyerAddress); err != nil {
		return err
	}
	if err = NonceCheck(&buyerWallet, args[4],
		strings.Join([]string{args[1], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	// sell price check
	if _, err = util.ParsePositive(args[2]); err != nil {
		return err
	}
	if newBidPrice, err = decimal.NewFromString(args[2]); err != nil {
		return errors.New("3004,The bid amount is incorrect. " + err.Error())
	}

	// get price info
	buyNow, _ = decimal.NewFromString(MRC401.AuctionBuyNowPrice)
	bidUnit, _ = decimal.NewFromString(MRC401.AuctionBiddingUnit)
	isBuynow = false
	if !buyNow.IsZero() {
		if newBidPrice.Cmp(buyNow) == 0 {
			isBuynow = true
		} else if newBidPrice.Cmp(buyNow) > 0 {
			return errors.New("3004,The bid amount must be less than or equal to the purchase buynow price")
		}
	}

	if !isBuynow {
		if MRC401.AuctionCurrentBidder == buyerAddress {
			return errors.New("3004,You are already the highest bidder")
		}
	}

	if util.IsAddress(MRC401.AuctionCurrentBidder) {
		oldBidPrice, _ = decimal.NewFromString(MRC401.AuctionCurrentPrice)
		if !isBuynow {
			if newBidPrice.Sub(bidUnit).Cmp(oldBidPrice) < 0 {
				return errors.New("3004,The bid amount must be greater than the current bid amount plus the bid unit")
			}
		}

		refunderAddress = MRC401.AuctionCurrentBidder
		// set payment info 2nd - Refund of previous bidder
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: MRC401.Id, ToAddr: refunderAddress,
			Amount: MRC401.AuctionCurrentPrice, TokenID: MRC401.AuctionToken, PayType: "mrc401_recv_refund"})
	} else {
		oldBidPrice, _ = decimal.NewFromString(MRC401.AuctionStartPrice)
		if newBidPrice.Cmp(oldBidPrice) < 0 {
			return errors.New("3004,The bid amount must be equal to or greater than the starting price")
		}
		refunderAddress = ""
	}

	// set payment info 1st - auction bid (buyer => mrc401)
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: buyerAddress, ToAddr: MRC401.Id,
		Amount: newBidPrice.String(), TokenID: MRC401.AuctionToken, PayType: "mrc401_bid"})

	// set new bidder
	MRC401.AuctionCurrentPrice = newBidPrice.String()
	MRC401.AuctionCurrentBidder = buyerAddress

	// buynow
	if isBuynow {
		// get item info
		if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
			return err
		}
		return mrc401DexProcess(stub, MRC400, MRC401, mtc.TWallet{}, buyerWallet,
			PaymentInfo, MRC401AT_Auction, args[3], args[4])
	}

	// not buynow
	if err = MRC010Subtract(stub, &buyerWallet, MRC401.AuctionToken, newBidPrice.String(), MRC010MT_Normal); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, buyerWallet, "transfer_mrc401bid",
		[]string{buyerAddress, MRC401.Id, newBidPrice.String(), MRC401.AuctionToken, args[3], "0", "", MRC401.Id, args[4]}); err != nil {
		return err
	}

	if util.IsAddress(refunderAddress) {
		if refunderWallet, err = GetAddressInfo(stub, refunderAddress); err != nil {
			return err
		}
		if err = MRC010Add(stub, &refunderWallet, MRC401.AuctionToken, oldBidPrice.String(), 0); err != nil {
			return err
		}
		if err = SetAddressInfo(stub, refunderWallet, "receive_mrc401refund",
			[]string{MRC401.Id, refunderAddress, oldBidPrice.String(), MRC401.AuctionToken, args[3],
				"0", "", MRC401.Id, args[4]}); err != nil {
			return err
		}
	}

	if err = setMRC401(stub, MRC401, "mrc401_auctionbid",
		[]string{MRC401.Id, buyerAddress, util.JSONEncode(PaymentInfo), args[3], args[4]}); err != nil {
		return err
	}
	return nil
}

// Mrc401AuctionFinish Mrc401AuctionFinish
func Mrc401AuctionFinish(stub shim.ChaincodeStubInterface, mrc401id string) error {
	var err error
	var now = time.Now().Unix()

	var MRC400 TMRC400
	var MRC401 TMRC401

	var buyerWallet mtc.TWallet
	var PaymentInfo []mtc.TDexPaymentInfo

	// get item info
	if MRC401, _, err = GetMRC401(stub, mrc401id); err != nil {
		return err
	}
	if MRC401.AuctionDate == 0 || MRC401.AuctionEnd == 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is not auction")
	}
	if MRC401.AuctionEnd > now {
		return errors.New("3004,It cannot be closed while the auction is pending")
	} else if MRC401.AuctionDate == 0 {
		return errors.New("3004,MRC401 is not auction item")
	}

	if MRC401.AuctionCurrentBidder == "" {
		// clear auction data
		MRC401.AuctionDate = 0
		MRC401.AuctionEnd = 0
		MRC401.AuctionToken = "0"
		MRC401.AuctionBiddingUnit = "0"
		MRC401.AuctionStartPrice = "0"
		MRC401.AuctionBuyNowPrice = "0"
		MRC401.AuctionCurrentPrice = "0"
		MRC401.AuctionCurrentBidder = ""
		if err = setMRC401(stub, MRC401, "mrc401_auctionfailure", []string{
			MRC401.Id, MRC401.Owner, ""}); err != nil {
			return err
		}
		return nil
	} else {
		if buyerWallet, err = GetAddressInfo(stub, MRC401.AuctionCurrentBidder); err != nil {
			return err
		}
		if MRC400, _, err = GetMRC400(stub, MRC401.MRC400); err != nil {
			return err
		}
		return mrc401DexProcess(stub, MRC400, MRC401, mtc.TWallet{}, buyerWallet, PaymentInfo, MRC401AT_Auction, "", "")
	}
}

// 1. 지급 내역 계산
// 2. 주소별 지급 내역 저장
// 3. DEx 저장.
func mrc401DexProcess(stub shim.ChaincodeStubInterface, MRC400 TMRC400, MRC401 TMRC401, creatorWallet, buyerWallet mtc.TWallet,
	PaymentInfo []mtc.TDexPaymentInfo, tradeType MRC401ActionType, sign, tkey string) error {
	var receiveAmount, paymentAmount decimal.Decimal
	var paymentToken string
	var commission decimal.Decimal
	var buyerAddress string
	var walletData mtc.TWallet
	var addrParams []string
	var err error
	var checkAddr string
	var jobType, sellerType, mrc401JobType string
	type RecvMapType struct {
		fromAddr    string
		toAddr      string
		amount      decimal.Decimal
		tradeAmount string
		jobType     string
		tokenID     string
	}
	var RecvMap map[string]RecvMapType
	SubtractJob := []string{"mrc401_bid", "mrc401_buy", "mrc401_create", "mrc401_createtrade", "mrc401_createtrade_buy"}

	if tradeType == MRC401AT_Auction {
		buyerAddress = MRC401.AuctionCurrentBidder
		sellerType = "mrc401_recv_auction"                           // set seller receive amount
		if MRC401.AuctionCurrentPrice == MRC401.AuctionBuyNowPrice { // set mrc401
			mrc401JobType = "mrc401_auctionbuynow"
		} else {
			mrc401JobType = "mrc401_auctionwinning"
		}
		paymentToken = MRC401.AuctionToken
		paymentAmount, _ = decimal.NewFromString(MRC401.AuctionCurrentPrice)

		// set last trade info
		MRC401.LastTradeDate = time.Now().Unix()
		MRC401.LastTradeAmount = MRC401.AuctionCurrentPrice
		MRC401.LastTradeToken = MRC401.AuctionToken
		MRC401.LastTradeType = "Auction"
	} else if tradeType == MRC401AT_Buy || tradeType == MRC401AT_CreateTrade {
		buyerAddress = buyerWallet.Id
		sellerType = "mrc401_recv_sell" // set seller receive amount
		if tradeType == MRC401AT_Buy {  // set mrc401
			mrc401JobType = "mrc401_buy"
		} else {
			mrc401JobType = "mrc401_createtrade"
		}
		paymentToken = MRC401.SellToken
		paymentAmount, _ = decimal.NewFromString(MRC401.SellPrice)

		// set last trade info
		MRC401.LastTradeDate = time.Now().Unix()
		MRC401.LastTradeAmount = MRC401.SellPrice
		MRC401.LastTradeToken = MRC401.SellToken
		MRC401.LastTradeType = "Sell"
	}

	receiveAmount = paymentAmount // seller receive amount

	// 3. creator commission calc
	if commission, err = DexFeeCalc(paymentAmount, MRC401.SellFee); err == nil {
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: MRC401.Id, ToAddr: MRC400.Owner,
			Amount: commission.String(), TokenID: paymentToken, PayType: "mrc401_recv_fee_owner"})
		// subtract seller receive amount
		receiveAmount = receiveAmount.Sub(commission)
	}

	// 4. platform commission calc
	if util.IsAddress(MRC401.PlatformAddress) {
		if commission, err = DexFeeCalc(paymentAmount, MRC401.PlatformCommission); err == nil {
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: MRC401.Id, ToAddr: MRC401.PlatformAddress,
				Amount: commission.String(), TokenID: paymentToken, PayType: "mrc401_recv_fee_platform"})
			// subtract seller receive amount
			receiveAmount = receiveAmount.Sub(commission)
		}
	}

	// 5. shareholder commission calc
	for shareholderAddress, shcomm := range MRC401.ShareHolder {
		if commission, err = DexFeeCalc(paymentAmount, shcomm); err == nil {
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: MRC401.Id, ToAddr: shareholderAddress,
				Amount: commission.String(), TokenID: paymentToken, PayType: "mrc401_recv_fee_shareholder"})
			// subtract seller receive amount
			receiveAmount = receiveAmount.Sub(commission)
		}
	}

	// set seller receive amount
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: MRC401.Id, ToAddr: MRC401.Owner,
		Amount: receiveAmount.String(), TokenID: paymentToken, PayType: sellerType})

	// payinfo grouping
	RecvMap = make(map[string]RecvMapType)
	Nev, _ := decimal.NewFromString("-1")

	for _, pi := range PaymentInfo {
		// mrc401_create may have a different coin, and has already been processed in the createtrade function.
		tAmount, _ := decimal.NewFromString(pi.Amount)
		if util.Contains(SubtractJob, pi.PayType) {
			tAmount = tAmount.Mul(Nev)
			checkAddr = pi.FromAddr
		} else {
			checkAddr = pi.ToAddr
		}

		if dt, exists := RecvMap[checkAddr]; !exists {
			// create initial reserve
			if pi.PayType == "mrc401_create" || pi.PayType == "mrc401_createtrade" {
				RecvMap[checkAddr] = RecvMapType{
					fromAddr:    pi.FromAddr,
					toAddr:      pi.ToAddr,
					jobType:     pi.PayType,
					tradeAmount: "",
					amount:      decimal.Zero,
					tokenID:     pi.TokenID,
				}
			} else {
				RecvMap[checkAddr] = RecvMapType{
					fromAddr:    pi.FromAddr,
					toAddr:      pi.ToAddr,
					jobType:     pi.PayType,
					tradeAmount: pi.TradeAmount,
					amount:      tAmount,
					tokenID:     paymentToken,
				}
			}
		} else {
			if dt.jobType == "mrc401_create" || dt.jobType == "mrc401_createtrade" {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr
			}
			// overwrite mrc401_recv_fee_* type
			if strings.Index(dt.jobType, "mrc401_recv_fee_") == 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

			} else if strings.Index(pi.PayType, "mrc401_recv_fee_") != 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr
				// overwrite mrc401_recv_refund type
			} else if dt.jobType == "mrc401_recv_refund" {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr
			}
			if pi.TradeAmount != "" && dt.tradeAmount == "" {
				dt.tradeAmount = pi.TradeAmount
			}
			dt.amount = dt.amount.Add(tAmount)
			RecvMap[checkAddr] = dt
		}
	}

	// payinfo save
	for _, v := range RecvMap {
		switch v.jobType {
		case "mrc401_create": // buyer => dex	구매비용 지불(MRC401)
			jobType = "mrc401create"
			checkAddr = v.fromAddr
		case "mrc401_buy": // buyer => dex	구매비용 지불(MRC401)
			jobType = "transfer_mrc401buy"
			checkAddr = v.fromAddr
		case "mrc401_createtrade_buy": // buyer => dex	구매비용 지불(MRC401) (mrc401 createtrade)
			jobType = "transfer_mrc401createtrade"
			checkAddr = v.fromAddr
		case "mrc401_createtrade": // create 초기 금액
			jobType = "transfer_mrc401create"
			checkAddr = v.fromAddr
		case "mrc401_bid": // buyer => dex	입찰 비용 지불(MRC401)
			jobType = "transfer_mrc401bid"
			checkAddr = v.fromAddr

		// checkaddr == to addr
		case "mrc401_recv_sell": // dex => seller	판매자 대금 받음(MRC401)
			jobType = "receive_mrc401sell"
			checkAddr = v.toAddr
		case "mrc401_recv_auction": // dex => seller	경매 낙찰금액 받음(MRC401)
			jobType = "receive_mrc401auction"
			checkAddr = v.toAddr
		case "mrc401_recv_refund": // dex=>refund	입찰 대금 환불(MRC401)
			jobType = "receive_mrc401refund"
			checkAddr = v.toAddr
		case "mrc401_recv_fee_owner": // dex => creator(MRC401)
			jobType = "receive_mrc401fee"
			checkAddr = v.toAddr
		case "mrc401_recv_fee_platform": // dex => platform(MRC401)
			jobType = "receive_mrc401fee"
			checkAddr = v.toAddr
		case "mrc401_recv_fee_shareholder": // dex => shareholder(MRC401)
			jobType = "receive_mrc401fee"
			checkAddr = v.toAddr
		}

		if (checkAddr == buyerWallet.Id) || (checkAddr == MRC401.AuctionCurrentBidder) {
			walletData = buyerWallet
		} else if checkAddr == creatorWallet.Id {
			walletData = creatorWallet
			if tradeType == MRC401AT_CreateTrade {
				jobType = "mrc401createtrade"
			}
		} else {
			if walletData, err = GetAddressInfo(stub, checkAddr); err != nil {
				return err
			}
		}

		if v.amount.Cmp(decimal.Zero) < 0 {
			if err = MRC010Subtract(stub, &walletData, paymentToken, v.amount.Abs().String(), MRC010MT_Normal); err != nil {
				return err
			}
		} else if v.amount.Cmp(decimal.Zero) > 0 {
			if err = MRC010Add(stub, &walletData, paymentToken, v.amount.Abs().String(), 0); err != nil {
				return err
			}
		}

		if (v.jobType == "mrc401_create") || (v.jobType == "mrc401_createtrade") {
			addrParams = []string{v.toAddr, v.fromAddr, v.amount.Abs().String(), v.tokenID, sign,
				"0", v.tradeAmount, MRC401.Id, tkey}

		} else {
			addrParams = []string{v.fromAddr, v.toAddr, v.amount.Abs().String(), paymentToken, sign,
				"0", v.tradeAmount, MRC401.Id, tkey}
		}
		if err = SetAddressInfo(stub, walletData, jobType, addrParams); err != nil {
			return err
		}
	}

	// clear auction data
	MRC401.AuctionDate = 0
	MRC401.AuctionEnd = 0
	MRC401.AuctionToken = "0"
	MRC401.AuctionBiddingUnit = "0"
	MRC401.AuctionStartPrice = "0"
	MRC401.AuctionBuyNowPrice = "0"
	MRC401.AuctionCurrentPrice = "0"
	MRC401.AuctionCurrentBidder = ""

	// clear sell data
	MRC401.SellDate = 0
	MRC401.SellPrice = "0"
	MRC401.SellToken = "0"

	// dex save
	addrParams = []string{MRC401.Id, MRC401.Owner, buyerAddress, util.JSONEncode(PaymentInfo)}
	// mrc401 owner change
	MRC401.Owner = buyerAddress
	setMRC401(stub, MRC401, mrc401JobType, addrParams)

	return nil
}
