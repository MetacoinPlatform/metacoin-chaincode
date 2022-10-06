// Package Metacoin MRC402
// function for NFT token
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

type MRC402ModifyType int

const (
	MRC402MT_Normal = iota
	MRC402MT_Sell
	MRC402MT_Auction
)

type MRC402DexStatus int

const (
	MRC402DS_SALE           = iota // on sale
	MRC402DS_CANCLED               // cancled
	MRC402DS_SOLDOUT               // saled
	MRC402DS_AUCTION_WAIT          // wait for auction start
	MRC402DS_AUCTION               // auction(biddable)
	MRC402DS_AUCTION_END           // auction end
	MRC402DS_AUCTION_FINISH        // auction finish
)

// Token TMRC402 - NFT TOKEN
type TMRC402 struct {
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

type TMRC402DEX struct {
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

// GetMRC402 get MRC402 token
//
// Example :
//
//		mtc.MRC402, err := GetMRC402(stub, "MRC402Key")
//
func GetMRC402(stub shim.ChaincodeStubInterface, mrc402id string) (TMRC402, []byte, error) {
	var byte_data []byte
	var err error
	var mrc402 TMRC402

	if strings.Index(mrc402id, "MRC402_") != 0 || len(mrc402id) != 40 {
		return mrc402, nil, errors.New("6102,invalid MRC402 ID")
	}

	byte_data, err = stub.GetState(mrc402id)
	if err != nil {
		return mrc402, nil, errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if byte_data == nil {
		return mrc402, nil, errors.New("6004,MRC402 [" + mrc402id + "] not exist")
	}
	if err = json.Unmarshal(byte_data, &mrc402); err != nil {
		return mrc402, nil, err
	}
	return mrc402, byte_data, nil
}

// Mrc402set set MRC402 token
//
// Example :
//
//		err := Mrc402set(stub, mtc.MRC402, "MRC402Key", "jobtype", arguments)
//
func setMRC402(stub shim.ChaincodeStubInterface, MRC402ItemData TMRC402, jobType string, jobArgs []string) error {
	var err error
	var byte_data []byte

	if strings.Index(MRC402ItemData.Id, "MRC402_") != 0 || len(MRC402ItemData.Id) != 40 {
		return errors.New("6102,invalid MRC402 data address")
	}

	MRC402ItemData.JobType = jobType
	MRC402ItemData.JobDate = time.Now().Unix()
	if byte_data, err = json.Marshal(jobArgs); err == nil {
		MRC402ItemData.JobArgs = string(byte_data)
	}

	if byte_data, err = json.Marshal(MRC402ItemData); err != nil {
		return errors.New("3209,Invalid MRC402ItemData data format")
	}

	if err := stub.PutState(MRC402ItemData.Id, byte_data); err != nil {
		return errors.New("8600,Mrc402Set stub.PutState [" + MRC402ItemData.Id + "] Error " + err.Error())
	}
	return nil
}

// GetDEX402 get MRC402 Dex item
//
// Example :
//
//		mtc.MRC402DEX, err := GetDEX402(stub, "MRC402Dex ITEM ID")
//
func GetDEX402(stub shim.ChaincodeStubInterface, dexid string) (TMRC402DEX, []byte, error) {
	var byte_data []byte
	var err error
	var mrc402dex TMRC402DEX

	if strings.Index(dexid, "DEX402_") != 0 || len(dexid) != 40 {
		return mrc402dex, nil, errors.New("6102,invalid DEX ID")
	}

	byte_data, err = stub.GetState(dexid)
	if err != nil {
		return mrc402dex, nil, errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if byte_data == nil {
		return mrc402dex, nil, errors.New("6004,MRC402 [" + dexid + "] not exist")
	}
	if err = json.Unmarshal(byte_data, &mrc402dex); err != nil {
		return mrc402dex, nil, err
	}
	return mrc402dex, byte_data, nil
}

// Mrc4dex402set02set set MRC402 Dex item
//
// Example :
//
//		err := setDEX402(stub, mtc.MRC402DEX, "MRC402Dex ITEM ID", "jobtype", arguments)
//
func setDEX402(stub shim.ChaincodeStubInterface, MRC402DexItem TMRC402DEX, jobType string, jobArgs []string) error {
	var err error
	var byte_data []byte

	if strings.Index(MRC402DexItem.Id, "DEX402_") != 0 || len(MRC402DexItem.Id) != 40 {
		return errors.New("6102,invalid DEX402 data address")
	}

	MRC402DexItem.JobType = jobType
	MRC402DexItem.JobDate = time.Now().Unix()
	if byte_data, err = json.Marshal(jobArgs); err == nil {
		MRC402DexItem.JobArgs = string(byte_data)
	}

	if MRC402DexItem.RegDate == 0 {
		MRC402DexItem.RegDate = MRC402DexItem.JobDate
	}

	switch jobType {
	case "mrc402_unsell", "mrc402_unauction":
		if MRC402DexItem.CancelDate == 0 {
			MRC402DexItem.CancelDate = MRC402DexItem.JobDate
		}

	case "mrc402_buy":
		if (MRC402DexItem.SellDate == 0) && (MRC402DexItem.RemainAmount == "0") {
			MRC402DexItem.SellDate = MRC402DexItem.JobDate
		}

	case "mrc402_auctionwinning", "mrc402_auctionbuynow":
		if MRC402DexItem.AuctionSettledDate == 0 {
			MRC402DexItem.SellDate = MRC402DexItem.JobDate
			MRC402DexItem.AuctionSettledDate = MRC402DexItem.JobDate
		}

	case "mrc010_auctionfailure":
		if MRC402DexItem.AuctionSettledDate == 0 {
			MRC402DexItem.AuctionSettledDate = MRC402DexItem.JobDate
		}
	}

	if byte_data, err = json.Marshal(MRC402DexItem); err != nil {
		return errors.New("3209,Invalid MRC402DexItem data format")
	}

	if err := stub.PutState(MRC402DexItem.Id, byte_data); err != nil {
		return errors.New("8600,dex402set stub.PutState [" + MRC402DexItem.Id + "] Error " + err.Error())
	}
	return nil
}

// 잔액 추가
//
// Example:
//
//     wallet.MRC402["MRC402ID"].Balance = 5
//     err := mrc402Add(stub, wallet, "MRC402ID", "10")
//     wallet.MRC402["MRC402ID"].Balance // output: "15"
//
func mrc402Add(stub shim.ChaincodeStubInterface, wallet *mtc.TWallet, mrc402id string,
	amount string, AddType MRC402ModifyType) error {
	var err error
	var toCoin, addAmount decimal.Decimal
	var balance mtc.NFTBalance
	var exists bool

	if addAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1102," + amount + " is not positive integer")
	}

	if _, _, err = GetMRC402(stub, mrc402id); err != nil {
		return err
	}
	if wallet.MRC402 == nil {
		wallet.MRC402 = make(map[string]mtc.NFTBalance)
	}
	if balance, exists = wallet.MRC402[mrc402id]; !exists {
		wallet.MRC402[mrc402id] = mtc.NFTBalance{
			Balance:       amount,
			SaleAmount:    "0",
			AuctionAmount: "0",
		}
		balance = wallet.MRC402[mrc402id]
	} else {
		toCoin, _ = decimal.NewFromString(balance.Balance)
		toCoin = toCoin.Add(addAmount).Truncate(0)
		balance.Balance = toCoin.String()
	}

	if AddType == MRC402MT_Sell {
		sa, _ := decimal.NewFromString(balance.SaleAmount)
		balance.SaleAmount = sa.Sub(addAmount).String()
	} else if AddType == MRC402MT_Auction {
		sa, _ := decimal.NewFromString(balance.AuctionAmount)
		balance.AuctionAmount = sa.Sub(addAmount).String()
	}
	wallet.MRC402[mrc402id] = balance

	return nil
}

// 잔액 감소
//
// Example:
//
//     wallet.MRC402["MRC402ID"].Balance = 15
//     err := mrc402Subtract(stub, wallet, "MRC402ID", "10")
//     wallet.MRC402["MRC402ID"].Balance // output: "5"
//
func mrc402Subtract(stub shim.ChaincodeStubInterface,
	wallet *mtc.TWallet, mrc402id string, amount string,
	SubtractType MRC402ModifyType) error {
	var err error
	var toCoin, subtractAmount decimal.Decimal
	var balance mtc.NFTBalance
	var exists bool

	if subtractAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1103," + amount + " is not positive integer")
	}

	if _, _, err = GetMRC402(stub, mrc402id); err != nil {
		return err
	}

	if wallet.MRC402 == nil {
		return errors.New("5000,Not enough balance")
	}
	if balance, exists = wallet.MRC402[mrc402id]; !exists {
		return errors.New("5000,Not enough balance")
	}

	toCoin, _ = decimal.NewFromString(balance.Balance)
	toCoin = toCoin.Sub(subtractAmount).Truncate(0)
	if toCoin.IsNegative() {
		return errors.New("5000,Not enough balance")
	}
	balance.Balance = toCoin.String()
	if SubtractType == MRC402MT_Sell {
		sa, _ := decimal.NewFromString(balance.SaleAmount)
		balance.SaleAmount = sa.Add(subtractAmount).String()
	} else if SubtractType == MRC402MT_Auction {
		sa, _ := decimal.NewFromString(balance.AuctionAmount)
		balance.AuctionAmount = sa.Add(subtractAmount).String()
	}

	if balance.Balance == "0" && balance.SaleAmount == "0" && balance.AuctionAmount == "0" {
		delete(wallet.MRC402, mrc402id)
	} else {
		wallet.MRC402[mrc402id] = balance
	}
	return nil
}

// 잔액 감소
//
// Example:
//
//     wallet.MRC402["MRC402ID"].Balance = 15
//     err := mrc402Subtract(stub, wallet, "MRC402ID", "10")
//     wallet.MRC402["MRC402ID"].Balance // output: "5"
//
func mrc402SubtractSubBalance(stub shim.ChaincodeStubInterface, wallet *mtc.TWallet, mrc402id string, amount string, SubtractType MRC402ModifyType) error {
	var err error
	var subtractAmount decimal.Decimal
	var balance mtc.NFTBalance
	var exists bool

	if subtractAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1104," + amount + " is not positive integer")
	}

	if _, _, err = GetMRC402(stub, mrc402id); err != nil {
		return err
	}

	if wallet.MRC402 == nil {
		return errors.New("5000,Not enough balance")
	}
	if balance, exists = wallet.MRC402[mrc402id]; !exists {
		return errors.New("5000,Not enough balance")
	}

	if SubtractType == MRC402MT_Sell {
		sa, _ := decimal.NewFromString(balance.SaleAmount)
		balance.SaleAmount = sa.Sub(subtractAmount).String()
	} else if SubtractType == MRC402MT_Auction {
		sa, _ := decimal.NewFromString(balance.AuctionAmount)
		balance.AuctionAmount = sa.Sub(subtractAmount).String()
	}
	wallet.MRC402[mrc402id] = balance
	return nil
}

// MRC402 잔액을 다른 Wallet 로 이동
//
// from, to wallet data의 잔액을 변경함.
func mrc402Move(stub shim.ChaincodeStubInterface, fromwallet *mtc.TWallet, towallet *mtc.TWallet,
	amount string, mrc402id string) error {
	var err error
	var balanceCheck, moveAmount decimal.Decimal
	var balance mtc.NFTBalance
	var exists bool

	if moveAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1105," + amount + " is not positive integer")
	}

	if _, _, err = GetMRC402(stub, mrc402id); err != nil {
		return err
	}
	if fromwallet.MRC402 == nil {
		return errors.New("5000,Not enough balance")
	}
	if balance, exists = fromwallet.MRC402[mrc402id]; !exists {
		return errors.New("5000,Not enough balance")
	}

	balanceCheck, _ = decimal.NewFromString(balance.Balance)
	balanceCheck = balanceCheck.Sub(moveAmount).Truncate(0)
	if balanceCheck.IsNegative() {
		return errors.New("5000,Not enough balance")
	}
	balance.Balance = balanceCheck.String()

	if balance.Balance == "0" && balance.SaleAmount == "0" && balance.AuctionAmount == "0" {
		delete(fromwallet.MRC402, mrc402id)
	} else {
		fromwallet.MRC402[mrc402id] = balance
	}

	if towallet.MRC402 == nil {
		towallet.MRC402 = make(map[string]mtc.NFTBalance)
	}
	if balance, exists = towallet.MRC402[mrc402id]; !exists {
		towallet.MRC402[mrc402id] = mtc.NFTBalance{
			Balance:       amount,
			SaleAmount:    "0",
			AuctionAmount: "0",
		}
	} else {
		balanceCheck, _ = decimal.NewFromString(balance.Balance)
		balanceCheck = balanceCheck.Add(moveAmount).Truncate(0)
		balance.Balance = balanceCheck.String()
		towallet.MRC402[mrc402id] = balance
	}

	return nil
}

// MRC402 Token create
//
// *Name, *Creator, *CreatorCommission, *TotalSupply, *Decimal,
// *URL, *ImageUrl, *shareholder, *InitialReserve, *ExpireDate,
// Data, Signature, Nonce
func Mrc402Create(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var MRC402Creator mtc.TWallet
	var MRC402 TMRC402
	var buf string
	var reserveAmount decimal.Decimal
	var totalSupply decimal.Decimal

	var argdat []byte

	if len(args) < 18 {
		return errors.New("1000,mrc402create operation must include four arguments : " +
			"creator, name, creatorcommission, totalsupply, decimal, " +
			"url, imageurl, shareholder, initialreserve, expiredate, " +
			"data, information, socialmedia, copyright_registration_country, copyright_registrar, " +
			"copyright_registration_number, signature, nonce")
	}

	MRC402 = TMRC402{
		MeltedAmount: "0",
		JobType:      "",
		JobArgs:      "",
		JobDate:      time.Now().Unix(),
	}

	// 0 Creator
	if err = util.DataAssign(args[0], &MRC402.Creator, "string", 1, 128, true); err != nil {
		return errors.New("3005,Name must be 1 to 128 characters long")
	}
	if MRC402Creator, err = GetAddressInfo(stub, MRC402.Creator); err != nil {
		return err
	}
	// sign check
	if err = NonceCheck(&MRC402Creator, args[17],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[4],
			args[5], args[6], args[7], args[8], args[9],
			args[10], args[11], args[12], args[13], args[14],
			args[15], args[17]}, "|"),
		args[16]); err != nil {
		return err
	}

	// 1 Name
	if err = util.DataAssign(args[1], &MRC402.Name, "string", 1, 128, false); err != nil {
		return errors.New("3005,Name value error : " + err.Error())
	}

	// 2 CreatorCommission
	if err = util.NumericDataCheck(args[2], &MRC402.CreatorCommission, "0", "10.0", 2, false); err != nil {
		return errors.New("3005,Creatorcommission value error : " + err.Error())
	}

	// 3 TotalSupply
	if err = util.NumericDataCheck(args[3], &MRC402.TotalSupply, "1", "99999999", 0, false); err != nil {
		return errors.New("3005,TotalSupply value error : " + err.Error())
	}
	if totalSupply, err = util.ParsePositive(MRC402.TotalSupply); err != nil {
		return errors.New("3005,TotalSupply value error : " + err.Error())
	}

	// 4 Decimal
	if args[4] != "0" {
		return errors.New("3005,Decimal is must be zero")
	}
	if MRC402.Decimal, err = util.Strtoint(args[4]); err != nil {
		return errors.New("3005,Decimal value error : " + err.Error())
	}
	if MRC402.Decimal < 0 {
		return errors.New("3005,The decimal number must be bigger then 0")
	}

	if MRC402.Decimal > 8 {
		return errors.New("3005,The decimal number must be less than 18")
	}

	// 5 URL
	if err = util.DataAssign(args[5], &MRC402.URL, "url", 1, 255, false); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 6 ImageURL
	if err = util.DataAssign(args[6], &MRC402.ImageURL, "url", 1, 255, false); err != nil {
		return errors.New("3005,ImageUril value error : " + err.Error())
	}

	// 7 shareholder
	if args[7] != "" {
		if err = json.Unmarshal([]byte(args[7]), &MRC402.ShareHolder); err != nil {
			return errors.New("3005,shareholder Data value error : " + err.Error())
		}
		if len(MRC402.ShareHolder) > 5 {
			return errors.New("3002,There must be 5 or fewer copyrighter")
		}
		index := 0
		for shareholder, commission := range MRC402.ShareHolder {
			if _, err = GetAddressInfo(stub, shareholder); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " shareholder item error : " + err.Error())
			}
			if err = util.NumericDataCheck(commission, &buf, "0", "10.0", 2, false); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " shareholder item commission error : " + err.Error())
			}
			index++
		}
	}

	// 8 InitialReserve
	if args[8] != "" {
		if err = json.Unmarshal([]byte(args[8]), &MRC402.InitialReserve); err != nil {
			return errors.New("3005,InitialReserve Data value error : " + err.Error())
		}
		if len(MRC402.InitialReserve) > 5 {
			return errors.New("3002,There must be 5 or fewer InitialReserve")
		}
		index := 0
		for initTokenID, initTokenAmount := range MRC402.InitialReserve {
			if _, _, err = GetMRC010(stub, initTokenID); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item token error : " + err.Error())
			}

			if reserveAmount, err = util.ParsePositive(initTokenAmount); err != nil {
				return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item amount error : " + err.Error())
			}

			reserveAmount = totalSupply.Mul(reserveAmount)
			if err = MRC010Subtract(stub, &MRC402Creator, initTokenID, reserveAmount.String(), MRC010MT_Normal); err != nil {
				return err
			}
			index++
		}
	}

	// 9 ExpireDate
	if MRC402.ExpireDate, err = util.Strtoint64(args[9]); err != nil {
		return errors.New("3005,ExpireDate value error : " + err.Error())
	}

	// 10 Data
	if err = util.DataAssign(args[10], &MRC402.Data, "string", 0, 40960, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// 11 Info
	if err = util.DataAssign(args[11], &MRC402.Info, "string", 0, 40960, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// 12 SocialMedia
	if err = util.DataAssign(args[12], &MRC402.SocialMedia, "string", 0, 40960, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// 13 Coyright Registration country
	if err = util.DataAssign(args[13], &MRC402.CopyrightRegCountry, "string", 0, 2, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	if err = util.ISO3166Check(MRC402.CopyrightRegCountry); err != nil {
		return err
	}

	// 14 Copyright Registrar
	if err = util.DataAssign(args[14], &MRC402.CopyrightRegistrar, "string", 0, 128, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// 15 Copyright Registration number
	if err = util.DataAssign(args[15], &MRC402.CopyrightRegNumber, "string", 0, 64, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// generate MRC402 ID
	var isSuccess = false
	temp := util.GenerateKey("MRC402_", args)
	for i := 0; i < 10; i++ {
		MRC402.Id = fmt.Sprintf("%39s%1d", temp, i)
		argdat, err = stub.GetState(MRC402.Id)
		if err != nil {
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if argdat != nil { // key already exists
			continue
		} else {
			isSuccess = true
			break
		}
	}
	if !isSuccess {
		return errors.New("3005,Data generate error, retry again")
	}

	setMRC402(stub, MRC402, "mrc402_create", []string{MRC402.Id,
		args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[13], args[14],
		args[15], args[16], args[17]})

	if MRC402Creator.MRC402 == nil {
		MRC402Creator.MRC402 = make(map[string]mtc.NFTBalance)
	}
	MRC402Creator.MRC402[MRC402.Id] = mtc.NFTBalance{
		Balance:       MRC402.TotalSupply,
		SaleAmount:    "0",
		AuctionAmount: "0",
	}

	// save create info
	// - for update balance
	// - for nonce update
	if err = SetAddressInfo(stub, MRC402Creator, "mrc402create", []string{MRC402.Id,
		args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[13], args[14],
		args[15], args[16], args[17]}); err != nil {
		return err
	}
	return nil
}

// Token Information update.
//
// *MRC402ID, *Url, Signature, Nonce
func Mrc402Update(stub shim.ChaincodeStubInterface, args []string) error {
	var MRC402 TMRC402
	var err error
	var creatorWallet mtc.TWallet
	var isUpdate bool

	if len(args) < 7 {
		return errors.New("1000,mrc402update operation must include four arguments : " +
			"MRC402ID, Url, Data, Info, SocialMedia, " +
			"copyright_registration_country, copyright_registrar, copyright_registration_number, Signature, Nonce")
	}

	if MRC402, _, err = GetMRC402(stub, args[0]); err != nil {
		return err
	}

	if creatorWallet, err = GetAddressInfo(stub, MRC402.Creator); err != nil {
		return err
	}

	if err = NonceCheck(&creatorWallet, args[9],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[9]}, "|"),
		args[8]); err != nil {
		return err
	}

	isUpdate = false

	if len(args[1]) > 0 && MRC402.URL != args[1] {
		isUpdate = true
	}
	if len(args[2]) > 0 && MRC402.Data != args[2] {
		isUpdate = true
	}
	if len(args[3]) > 0 && MRC402.Info != args[3] {
		isUpdate = true
	}
	if len(args[4]) > 0 && MRC402.SocialMedia != args[4] {
		isUpdate = true
	}
	// 5 Coyright Registration country
	if len(args[5]) > 0 {
		if len(MRC402.CopyrightRegCountry) == 0 {
			isUpdate = true
			if err = util.DataAssign(args[5], &MRC402.CopyrightRegCountry, "string", 0, 2, true); err != nil {
				return errors.New("3005,Data value error : " + err.Error())
			}
			if err = util.ISO3166Check(MRC402.CopyrightRegCountry); err != nil {
				return err
			}
		} else {
			return errors.New("3005,Copyright registration country cannot be changed")
		}
	}

	// 6 Copyright Registrar
	if len(args[6]) > 0 {
		if len(MRC402.CopyrightRegistrar) == 0 {
			isUpdate = true
			if err = util.DataAssign(args[6], &MRC402.CopyrightRegistrar, "string", 0, 128, true); err != nil {
				return errors.New("3005,Data value error : " + err.Error())
			}
		} else {
			return errors.New("3005,Copyright registrar cannot be changed")
		}
	}

	// 7 Copyright Registration number
	if len(args[7]) > 0 {
		if len(MRC402.CopyrightRegNumber) == 0 {
			isUpdate = true
			if err = util.DataAssign(args[7], &MRC402.CopyrightRegNumber, "string", 0, 64, true); err != nil {
				return errors.New("3005,Data value error : " + err.Error())
			}
		} else {
			return errors.New("3005,Copyright registration number cannot be changed")
		}
	}

	if !isUpdate {
		return errors.New("4900,No data change")
	}

	// 1 URL
	if err = util.DataAssign(args[1], &MRC402.URL, "url", 1, 255, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 2 Data
	if err = util.DataAssign(args[2], &MRC402.Data, "string", 0, 40960, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// 3 Info
	if err = util.DataAssign(args[3], &MRC402.Info, "string", 0, 40960, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	// 4 SocialMedia
	if err = util.DataAssign(args[4], &MRC402.SocialMedia, "string", 0, 40960, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	params := []string{MRC402.Id, MRC402.Creator, args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9]}
	if err = setMRC402(stub, MRC402, "mrc402_update", params); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, creatorWallet, "mrc402update", params); err != nil {
		return err
	}
	return nil
}

// MRC402 Token burning
//
// MRC402ID, amount, Signature, Nonce
func Mrc402Burn(stub shim.ChaincodeStubInterface, args []string) error {
	var mrc402 TMRC402
	var err error
	var TotalSupply, BurnAmount, BurnableAmount, MeltedAmount decimal.Decimal

	var MRC402Creator mtc.TWallet
	var reserveAmount decimal.Decimal
	var buf string

	if len(args) < 5 {
		return errors.New("1000,mrc402burning operation must include four arguments : " +
			"MRC402ID, amount, memo, Signature, Nonce")
	}

	if mrc402, _, err = GetMRC402(stub, args[0]); err != nil {
		return err
	}

	if BurnAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1206,The amount must be an integer")
	}

	if MRC402Creator, err = GetAddressInfo(stub, mrc402.Creator); err != nil {
		return err
	}

	if err = util.DataAssign(args[2], &buf, "string", 0, 1024, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	if err = NonceCheck(&MRC402Creator, args[4],
		strings.Join([]string{args[0], args[1], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = mrc402Subtract(stub, &MRC402Creator, mrc402.Id, BurnAmount.String(), MRC402MT_Normal); err != nil {
		return err
	}

	TotalSupply, _ = decimal.NewFromString(mrc402.TotalSupply)
	MeltedAmount, _ = decimal.NewFromString(mrc402.MeltedAmount)
	BurnableAmount = TotalSupply.Sub(MeltedAmount) // total supply - melted amount = BurnableAmount
	TotalSupply = BurnableAmount.Sub(BurnAmount)   // BurnableAmount - burn amount = new total supply
	if err = util.NumericDataCheck(TotalSupply.String(), &mrc402.TotalSupply, "0", "99999999", 0, false); err != nil {
		return errors.New("3005,The maximum amount that can be burn is " + BurnableAmount.String())
	}

	// 8 InitialReserve
	index := 0
	for initTokenID, initTokenAmount := range mrc402.InitialReserve {
		if _, _, err = GetMRC010(stub, initTokenID); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item token error : " + err.Error())
		}

		if reserveAmount, err = util.ParsePositive(initTokenAmount); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item amount error : " + err.Error())
		}

		reserveAmount = BurnAmount.Mul(reserveAmount)
		if err = MRC010Add(stub, &MRC402Creator, initTokenID, reserveAmount.String(), 0); err != nil {
			return err
		}
		index++
	}

	if err = setMRC402(stub, mrc402, "mrc402_burn",
		[]string{mrc402.Id, mrc402.Creator, args[1], args[2], args[3], args[4]}); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, MRC402Creator, "mrc402burn",
		[]string{mrc402.Id, mrc402.Creator, args[1], args[2], args[3], args[4]}); err != nil {
		return err
	}
	return nil
}

// Token Increase
//
// MRC402ID, amount, Signature, Nonce
func Mrc402Mint(stub shim.ChaincodeStubInterface, args []string) error {
	var MRC402 TMRC402
	var err error
	var TotalSupply, MintAmount decimal.Decimal

	var MRC402Creator mtc.TWallet
	var reserveAmount decimal.Decimal
	var buf string

	if len(args) < 5 {
		return errors.New("1000,mrc402burning operation must include four arguments : " +
			"MRC402ID, amount, memo, Signature, Nonce")
	}

	if MRC402, _, err = GetMRC402(stub, args[0]); err != nil {
		return err
	}

	if MintAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1206,The amount must be an integer")
	}

	if MRC402Creator, err = GetAddressInfo(stub, MRC402.Creator); err != nil {
		return err
	}

	if err = util.DataAssign(args[2], &buf, "string", 0, 1024, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	if err = NonceCheck(&MRC402Creator, args[4],
		strings.Join([]string{args[0], args[1], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = mrc402Add(stub, &MRC402Creator, MRC402.Id, MintAmount.String(), MRC402MT_Normal); err != nil {
		return err
	}

	TotalSupply, _ = decimal.NewFromString(MRC402.TotalSupply)
	TotalSupply = TotalSupply.Add(MintAmount)
	if err = util.NumericDataCheck(TotalSupply.String(), &MRC402.TotalSupply, "1", "99999999", 0, false); err != nil {
		return errors.New("3005,After the mint, the value of TotalSupply is less than 99999999")
	}

	index := 0
	for initTokenID, initTokenAmount := range MRC402.InitialReserve {
		if _, _, err = GetMRC010(stub, initTokenID); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item token error : " + err.Error())
		}

		if reserveAmount, err = util.ParsePositive(initTokenAmount); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item amount error : " + err.Error())
		}

		reserveAmount = MintAmount.Mul(reserveAmount)
		if err = MRC010Subtract(stub, &MRC402Creator, initTokenID, reserveAmount.String(), MRC010MT_Normal); err != nil {
			return err
		}
		index++
	}

	if err = setMRC402(stub, MRC402, "mrc402_mint",
		[]string{MRC402.Id, MRC402.Creator, args[1], args[2], args[3], args[4]}); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, MRC402Creator, "mrc402mint",
		[]string{MRC402.Id, MRC402.Creator, args[1], args[2], args[3], args[4]}); err != nil {
		return err
	}

	return nil
}

// Token Information update
//
// *fromAddress, *toAddress, *amount, *MRC402ID, tag, memo, Signature, Nonce
func Mrc402Transfer(stub shim.ChaincodeStubInterface, args []string) error {
	var MRC402 TMRC402
	var err error
	var fromWallet, toWallet mtc.TWallet
	var TransferAmount decimal.Decimal

	if len(args) < 8 {
		return errors.New("1000,mrc402burning operation must include four arguments : " +
			"fromAddr, toAddr, amount, MRC402ID, tag, memo, signature, Nonce")

	}

	if fromWallet, err = GetAddressInfo(stub, args[0]); err != nil {
		return errors.New("3001,Invalid from address")
	}

	if toWallet, err = GetAddressInfo(stub, args[1]); err != nil {
		return errors.New("3001,Invalid to address")
	}

	if args[0] == args[1] {
		return errors.New("3201,From address and to address must be different values")
	}

	if TransferAmount, err = util.ParsePositive(args[2]); err != nil {
		return errors.New("1206,The amount must be an integer")
	}

	if MRC402, _, err = GetMRC402(stub, args[3]); err != nil {
		return err
	}

	if err = NonceCheck(&fromWallet, args[7],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[7]}, "|"),
		args[6]); err != nil {
		return err
	}

	if err = mrc402Move(stub, &fromWallet, &toWallet, TransferAmount.String(), MRC402.Id); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, fromWallet, "transfer_mrc402", args); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, toWallet, "receive_mrc402", args); err != nil {
		return err
	}

	return nil
}

// Mrc402Melt Mrc402Melt
func Mrc402Melt(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var melterWallet mtc.TWallet
	var mrc402 TMRC402
	var meltAmount, MeltedAmount decimal.Decimal

	var reserveAmount decimal.Decimal

	if len(args) < 5 {
		return errors.New("1000,mrc402melt operation must include four arguments : " +
			"mrc402id, address, amount, signature, nonce")
	}

	// 0 mrc402
	if mrc402, _, err = GetMRC402(stub, args[0]); err != nil {
		return err
	}

	if mrc402.ExpireDate != 0 && mrc402.ExpireDate > time.Now().Unix() {
		return errors.New("3004,It is not a meltable date")
	}

	// 1 Melter
	if melterWallet, err = GetAddressInfo(stub, args[1]); err != nil {
		return err
	}

	// 2 amount
	if meltAmount, err = util.ParsePositive(args[2]); err != nil {
		return errors.New("1106," + args[2] + " is not positive integer")
	}

	if err = NonceCheck(&melterWallet, args[4],
		strings.Join([]string{args[0], args[1], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = mrc402Subtract(stub, &melterWallet, mrc402.Id, meltAmount.String(), MRC402MT_Normal); err != nil {
		return err
	}

	MeltedAmount, _ = decimal.NewFromString(mrc402.MeltedAmount)
	MeltedAmount = MeltedAmount.Add(meltAmount)
	mrc402.MeltedAmount = MeltedAmount.String()

	// 8 InitialReserve
	index := 0
	for initTokenID, initTokenAmount := range mrc402.InitialReserve {
		if _, _, err = GetMRC010(stub, initTokenID); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item token error : " + err.Error())
		}

		if reserveAmount, err = util.ParsePositive(initTokenAmount); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " InitialReserve item amount error : " + err.Error())
		}

		reserveAmount = meltAmount.Mul(reserveAmount)
		if err = MRC010Add(stub, &melterWallet, initTokenID, reserveAmount.String(), 0); err != nil {
			return err
		}
		index++
	}

	if err = setMRC402(stub, mrc402, "mrc402_melt", args); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, melterWallet, "mrc402melt", args); err != nil {
		return err
	}
	return nil
}

func dex402Status(dex TMRC402DEX) MRC402DexStatus {
	var now = time.Now().Unix()
	if dex.CancelDate > 0 {
		return MRC402DS_CANCLED // Sale or auction canceled
	}

	if dex.AuctionStartDate == 0 {
		if dex.SellDate == 0 {
			return MRC402DS_SALE // on sale
		} else {
			return MRC402DS_SOLDOUT // solded
		}
	} else {
		if dex.AuctionStartDate > now {
			return MRC402DS_AUCTION_WAIT // wait for auction
		}
		if dex.AuctionEndDate > now {
			return MRC402DS_AUCTION // in auction
		}
		if dex.AuctionSettledDate == 0 {
			return MRC402DS_AUCTION_END // auction finish but not yet price calc
		} else {
			return MRC402DS_AUCTION_FINISH // auction finish && price complete.
		}
	}
}

// Mrc402Sell Mrc402Sell
func Mrc402Sell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var mrc402 TMRC402
	var sellAmount, unitPrice, totalPrice decimal.Decimal
	var dex TMRC402DEX
	var argdat []byte

	if len(args) < 11 {
		return errors.New("1000,mrc402sell operation must include four arguments : " +
			"seller, amount, mrc402id, sellPrice, selltoken, " +
			"platformName, platformURL, platformAddress, platformCommission, " +
			"signature, nonce")
	}

	// 0 seller
	if sellerWallet, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}

	// 1 amount
	if sellAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1107," + args[1] + " is not positive integer")
	}

	// 2 mrc402id
	if mrc402, _, err = GetMRC402(stub, args[2]); err != nil {
		return err
	}

	dex = TMRC402DEX{
		MRC402:           mrc402.Id,
		Seller:           args[0],
		Amount:           sellAmount.String(),
		RemainAmount:     sellAmount.String(),
		SellToken:        args[4],
		RegDate:          0,
		SellDate:         0,
		CancelDate:       0,
		AuctionStartDate: 0,
		AuctionEndDate:   0,
	}

	// 3 sellPrice
	if err = util.NumericDataCheck(args[3], &dex.SellPrice, "1", "", 0, false); err != nil {
		return err
	}
	if unitPrice, err = util.ParsePositive(args[3]); err != nil {
		return err
	}
	totalPrice = sellAmount.Mul(unitPrice)
	if err = util.NumericDataCheck(totalPrice.String(), nil, "1", "", 0, false); err != nil {
		return err
	}

	// 4 selltoken
	if _, _, err = GetMRC010(stub, dex.SellToken); err != nil {
		return err
	}

	// 5 platformname
	if err = util.DataAssign(args[5], &dex.PlatformName, "", 1, 255, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 6 PlatformURL
	if err = util.DataAssign(args[6], &dex.PlatformURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 7 PlatformAddress
	if err = util.DataAssign(args[7], &dex.PlatformAddress, "address", 40, 40, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}
	if util.IsAddress(dex.PlatformAddress) {
		if _, err = GetAddressInfo(stub, dex.PlatformAddress); err != nil {
			return errors.New("3005," + "PlatformAddress not found : " + err.Error())
		}
	}

	// 8 PlatformCommission
	if err = util.NumericDataCheck(args[8], &dex.PlatformCommission, "0.00", "10.00", 2, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	if err = NonceCheck(&sellerWallet, args[10],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[4],
			args[5], args[6], args[7], args[8],
			args[10]}, "|"),
		args[9]); err != nil {
		return err
	}

	if err = mrc402Subtract(stub, &sellerWallet, mrc402.Id, sellAmount.String(), MRC402MT_Sell); err != nil {
		return err
	}

	// generate MRC402 ID
	var isSuccess = false
	temp := util.GenerateKey("DEX402_", args)
	for i := 0; i < 10; i++ {
		dex.Id = fmt.Sprintf("%39s%1d", temp, i)
		argdat, err = stub.GetState(dex.Id)
		if err != nil {
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if argdat != nil { // key already exists
			continue
		} else {
			isSuccess = true
			break
		}
	}
	if !isSuccess {
		return errors.New("3005,Data generate error, retry again")
	}

	params := []string{dex.Id, args[0], args[1], args[2], args[3],
		args[4], args[5], args[6], args[7], args[8],
		args[9], args[10]}

	if err = setDEX402(stub, dex, "mrc402_sell", params); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, sellerWallet, "mrc402sell", params); err != nil {
		return err
	}
	return nil
}

// Mrc402Melt Mrc402Melt
func Mrc402UnSell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var dex TMRC402DEX

	if len(args) < 3 {
		return errors.New("1000,mrc402unsell operation must include four arguments : " +
			"dexid, signature, nonce")
	}

	// 0 mrc402id
	if dex, _, err = GetDEX402(stub, args[0]); err != nil {
		return err
	}
	if dex.AuctionStartDate > 0 {
		return errors.New("3004,DEX Item is not sell item")
	}
	switch dex402Status(dex) {
	case MRC402DS_SALE:
		// OK
	case MRC402DS_AUCTION_WAIT, MRC402DS_AUCTION, MRC402DS_AUCTION_END, MRC402DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is not sell item")
	case MRC402DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC402DS_SOLDOUT:
		return errors.New("3004,DEX Item is already traded")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	// 1 seller
	if sellerWallet, err = GetAddressInfo(stub, dex.Seller); err != nil {
		return err
	}

	if err = NonceCheck(&sellerWallet, args[2],
		strings.Join([]string{args[0], args[2]}, "|"),
		args[1]); err != nil {
		return err
	}

	if err = mrc402Add(stub, &sellerWallet, dex.MRC402, dex.RemainAmount, MRC402MT_Sell); err != nil {
		return err
	}

	params := []string{dex.Id, dex.Seller, dex.RemainAmount, dex.MRC402, dex.SellPrice,
		dex.SellToken, args[1], args[2]}

	if err = setDEX402(stub, dex, "mrc402_unsell", params); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, sellerWallet, "mrc402unsell", params); err != nil {
		return err
	}
	return nil
}

/*
MRC402DEX item buy.

DEX status is must be MRC402DS_SALE
*/
func Mrc402Buy(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var buyerWallet mtc.TWallet
	var buyAmount, unitPrice, remainAmount decimal.Decimal

	var dex TMRC402DEX
	var PaymentInfo []mtc.TDexPaymentInfo

	// argument check
	if len(args) < 5 {
		return errors.New("1000,mrc402unsell operation must include four arguments : " +
			"dexid, buyer, amount, signature, nonce")
	}

	// 0 mrc402id
	if dex, _, err = GetDEX402(stub, args[0]); err != nil {
		return err
	}

	if dex.Seller == args[1] {
		return errors.New("3004,Seller is don't buy")
	}

	switch dex402Status(dex) {
	case MRC402DS_SALE:
		// OK
	case MRC402DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC402DS_SOLDOUT:
		return errors.New("3004,DEX Item is already traded")
	case MRC402DS_AUCTION_WAIT, MRC402DS_AUCTION, MRC402DS_AUCTION_END, MRC402DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is not sell item")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	// 1 buyer & sign check
	if buyerWallet, err = GetAddressInfo(stub, args[1]); err != nil {
		return err
	}
	if err = NonceCheck(&buyerWallet, args[4],
		strings.Join([]string{args[0], args[1], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	// 2 buy amount check.
	if buyAmount, err = util.ParsePositive(args[2]); err != nil {
		return err
	}
	if remainAmount, err = util.ParsePositive(dex.RemainAmount); err != nil {
		return err
	}
	if remainAmount.Cmp(buyAmount) < 0 {
		return errors.New("3004,The quantity available for purchase is " + remainAmount.String() + " pieces.")
	}
	dex.RemainAmount = remainAmount.Sub(buyAmount).String()

	if unitPrice, err = util.ParsePositive(dex.SellPrice); err != nil {
		return err
	}

	dex.Buyer = args[1]
	PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Buyer, ToAddr: dex.Id,
		Amount: buyAmount.Mul(unitPrice).String(), TokenID: dex.SellToken, TradeAmount: "", TradeID: "", PayType: "mrc402_buy"})
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.Buyer,
		Amount: "", TokenID: "", TradeAmount: buyAmount.String(), TradeID: dex.MRC402, PayType: "mrc402_recv_item"})

	return mrc402DexProcess(stub, dex, buyerWallet, PaymentInfo, MRC402MT_Sell, buyAmount.String(), args[3], args[4])
}

/* Mrc402Auction

args arguments
	seller, amount, mrc402id, auction_start_price, selltoken,
	auction_bidding_unit, auction_buynow_price, auction_start_date, auction_end_date, platformName,
	platformURL, platformAddress, platformCommission, signature, nonce
*/
func Mrc402Auction(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var mrc402 TMRC402
	var sellAmount, startPrice, buyNowPrice decimal.Decimal
	var dex TMRC402DEX
	var argdat []byte
	var now int64

	if len(args) < 15 {
		return errors.New("1000,mrc402auction operation must include four arguments : " +
			"address, amount, mrc402id, auction_start_price, selltoken, " +
			"auction_bidding_unit, auction_buynow_price, auction_start_date, auction_end_date, platformName, " +
			"platformURL, platformAddress, platformCommission, signature, nonce")
	}
	now = time.Now().Unix()

	// 0 seller
	if sellerWallet, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}

	// 1 amount
	if sellAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1108," + args[1] + " is not positive integer")
	}

	// 2 mrc402id
	if mrc402, _, err = GetMRC402(stub, args[2]); err != nil {
		return err
	}
	dex = TMRC402DEX{
		MRC402:               mrc402.Id,
		Seller:               args[0],
		Amount:               sellAmount.String(),
		SellToken:            strings.Trim(args[4], " \t\n\r"),
		Buyer:                "",
		RegDate:              0,
		SellDate:             0,
		CancelDate:           0,
		AuctionBidCount:      0,
		AuctionStartDate:     0,
		AuctionEndDate:       0,
		AuctionBiddingUnit:   "0", //
		AuctionStartPrice:    "0", //
		AuctionBuyNowPrice:   "",  //
		AuctionCurrentPrice:  "",
		AuctionCurrentBidder: "",
	}

	// 3 auction start price
	if err = util.NumericDataCheck(args[3], &dex.AuctionStartPrice, "1", "", 0, false); err != nil {
		return err
	}
	// 4 selltoken
	if _, _, err = GetMRC010(stub, args[4]); err != nil {
		return err
	}

	// 5 auction bidding unit
	if err = util.NumericDataCheck(args[5], &dex.AuctionBiddingUnit, "1", "", 0, true); err != nil {
		return err
	}

	// 6 auction buy now price
	if err = util.NumericDataCheck(args[6], &dex.AuctionBuyNowPrice, "0", "", 0, true); err != nil {
		return err
	}

	if startPrice, err = decimal.NewFromString(dex.AuctionStartPrice); err != nil {
		return err
	}

	if buyNowPrice, err = decimal.NewFromString(dex.AuctionBuyNowPrice); err != nil {
		return err
	}

	if buyNowPrice.Cmp(decimal.Zero) > 0 {
		if buyNowPrice.Cmp(startPrice) < 1 {
			return errors.New("3005,Auction buynow is too small")
		}
	}

	// 7 auction start date
	if len(args[7]) == 0 {
		dex.AuctionStartDate = 0
	} else if dex.AuctionStartDate, err = util.Strtoint64(args[7]); err != nil {
		return err
	}

	if dex.AuctionStartDate <= 0 {
		dex.AuctionStartDate = now
	} else if dex.AuctionStartDate < now {
		return errors.New("3005,The auction start time is in the past")
	} else if (dex.AuctionStartDate - now) > 1814400 {
		return errors.New("3005,Auction start time must be within 7 days")
	}

	if len(args[8]) == 0 {
		dex.AuctionEndDate = 0
	} else if dex.AuctionEndDate, err = util.Strtoint64(args[8]); err != nil {
		return err
	}
	if dex.AuctionEndDate <= 0 {
		dex.AuctionEndDate = dex.AuctionStartDate + 86400
	} else if (dex.AuctionEndDate - now) < 3600 {
		return errors.New("3005,Auction duration is at least 1 hour")
	} else if (dex.AuctionEndDate - now) > 1814400 {
		return errors.New("3005,The auction period is up to 7 days")
	}

	// 8 platformname
	if err = util.DataAssign(args[9], &dex.PlatformName, "", 1, 255, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 9 PlatformURL
	if err = util.DataAssign(args[10], &dex.PlatformURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 10 PlatformAddress
	if err = util.DataAssign(args[11], &dex.PlatformAddress, "address", 40, 40, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}
	if util.IsAddress(dex.PlatformAddress) {
		if _, err = GetAddressInfo(stub, dex.PlatformAddress); err != nil {
			return errors.New("3005," + "PlatformAddress not found : " + err.Error())
		}
	}

	// 11 PlatformCommission
	if err = util.NumericDataCheck(args[12], &dex.PlatformCommission, "0.00", "10.00", 2, true); err != nil {
		return errors.New("3005,Data value error : " + err.Error())
	}

	if err = NonceCheck(&sellerWallet, args[14],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[4],
			args[5], args[6], args[7], args[8], args[9],
			args[10], args[11], args[12], args[14]}, "|"),
		args[13]); err != nil {
		return err
	}

	if err = mrc402Subtract(stub, &sellerWallet, mrc402.Id, sellAmount.String(), MRC402MT_Auction); err != nil {
		return err
	}

	// generate MRC402 ID
	var isSuccess = false
	temp := util.GenerateKey("DEX402_", args)
	for i := 0; i < 10; i++ {
		dex.Id = fmt.Sprintf("%39s%1d", temp, i)
		argdat, err = stub.GetState(dex.Id)
		if err != nil {
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if argdat != nil { // key already exists
			continue
		} else {
			isSuccess = true
			break
		}
	}
	if !isSuccess {
		return errors.New("3005,Data generate error, retry again")
	}

	// 	"seller, amount, mrc402id, auction_start_price, selltoken, " +
	//	"auction_bidding_unit, auction_buynow_price, auction_start_date, auction_end_date, platformName, " +
	//	"platformURL, platformAddress, platformCommission, signature, nonce")
	params := []string{dex.Id, args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[14]}

	if err = setDEX402(stub, dex, "mrc402_auction", params); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, sellerWallet, "mrc402auction", params); err != nil {
		return err
	}
	return nil
}

/* Mrc402UnAuction

args arguments
	dexid, signature, nonce
*/
func Mrc402UnAuction(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var dex TMRC402DEX

	if len(args) < 3 {
		return errors.New("1000,mrc402unauction operation must include four arguments : " +
			"mrc402dexid, signature, nonce")
	}

	// 0 mrc402id
	if dex, _, err = GetDEX402(stub, args[0]); err != nil {
		return err
	}
	if dex.AuctionStartDate == 0 {
		return errors.New("3004,DEX Item is not sell item")
	}

	switch dex402Status(dex) {
	case MRC402DS_AUCTION, MRC402DS_AUCTION_WAIT:
		// ok
	case MRC402DS_SALE, MRC402DS_SOLDOUT:
		return errors.New("3004,DEX Item is not auction item")
	case MRC402DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC402DS_AUCTION_END:
		return errors.New("3004,DEX Item is already end, use auction finish")
	case MRC402DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is already finished")
	default:
		return errors.New("3004,DEX Item status is unknown")

	}

	// bidder exists ?
	if dex.AuctionCurrentBidder != "" {
		return errors.New("3004,DEX Item there is a bidder, so the auction cannot be canceled")
	}

	// sign check.
	if sellerWallet, err = GetAddressInfo(stub, dex.Seller); err != nil {
		return err
	}

	if err = NonceCheck(&sellerWallet, args[2],
		strings.Join([]string{args[0], args[2]}, "|"),
		args[1]); err != nil {
		return err
	}

	if err = mrc402Add(stub, &sellerWallet, dex.MRC402, dex.Amount, MRC402MT_Auction); err != nil {
		return err
	}

	params := []string{dex.Id, dex.Seller, dex.Amount, dex.MRC402, dex.SellPrice, dex.SellToken, args[1], args[2]}

	if err = setDEX402(stub, dex, "mrc402_unauction", params); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, sellerWallet, "mrc402unauction", params); err != nil {
		return err
	}
	return nil
}

/*
경매중인 아이템 입찰

args arguments
- dexid, buyer, bidAmount, signature, nonce
*/
func Mrc402AuctionBid(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var buyerWallet mtc.TWallet
	var dex TMRC402DEX
	var buyerAddress string
	var refunderWallet mtc.TWallet
	var refunderAddress string

	var PaymentInfo []mtc.TDexPaymentInfo
	PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)

	var buyNow, oldBidPrice, newBidPrice, bidUnit decimal.Decimal
	var isBuynow bool

	if len(args) < 5 {
		return errors.New("1000,mrc402bid operation must include four arguments : " +
			"mrc402dexid, address, amount, signature, nonce")
	}
	buyerAddress = args[1]

	// 0 mrc402id
	if dex, _, err = GetDEX402(stub, args[0]); err != nil {
		return err
	}

	switch dex402Status(dex) {
	case MRC402DS_AUCTION:
		// OK
	case MRC402DS_SALE, MRC402DS_SOLDOUT:
		return errors.New("3004,DEX Item is not auction item")
	case MRC402DS_AUCTION_WAIT:
		return errors.New("3004,This is not the auction bidding period")
	case MRC402DS_AUCTION_END:
		return errors.New("3004,DEX Item is already end, use auction finish")
	case MRC402DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is already finished")
	case MRC402DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	// buyer is seller ?
	if dex.Seller == buyerAddress {
		return errors.New("3004,Seller is don't buy")
	}

	// 1 buyer
	if buyerWallet, err = GetAddressInfo(stub, buyerAddress); err != nil {
		return err
	}

	if err = NonceCheck(&buyerWallet, args[4],
		strings.Join([]string{args[0], buyerAddress, args[2], args[4]}, "|"),
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
	buyNow, _ = decimal.NewFromString(dex.AuctionBuyNowPrice)
	bidUnit, _ = decimal.NewFromString(dex.AuctionBiddingUnit)

	if util.IsAddress(dex.AuctionCurrentBidder) {
		oldBidPrice, _ = decimal.NewFromString(dex.AuctionCurrentPrice)
		if newBidPrice.Sub(bidUnit).Cmp(oldBidPrice) < 0 {
			return errors.New("3004,The bid amount must be greater than the current bid amount plus the bid unit")
		}

		refunderAddress = dex.AuctionCurrentBidder
		// set payment info 2nd - Refund of previous bidder
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: refunderAddress,
			Amount: dex.AuctionCurrentPrice, TokenID: dex.SellToken, PayType: "mrc402_recv_refund"})
	} else {
		oldBidPrice, _ = decimal.NewFromString(dex.AuctionStartPrice)
		if newBidPrice.Cmp(oldBidPrice) < 0 {
			return errors.New("3004,The bid amount must be equal to or greater than the starting price")
		}
		refunderAddress = ""
	}

	isBuynow = false
	if !buyNow.IsZero() {
		if newBidPrice.Cmp(buyNow) == 0 {
			isBuynow = true
		} else if newBidPrice.Cmp(buyNow) > 0 {
			return errors.New("3004,The bid amount must be less than or equal to the purchase buynow price")
		}
	}

	if !isBuynow {
		if dex.AuctionCurrentBidder == buyerAddress {
			return errors.New("3004,You are already the highest bidder")
		}
	}

	// subtract auction bidding price
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: buyerAddress, ToAddr: dex.Id,
		Amount: newBidPrice.String(), TokenID: dex.SellToken, TradeAmount: "", TradeID: "", PayType: "mrc402_bid"})

	// set new bidder
	dex.AuctionCurrentPrice = newBidPrice.String()
	dex.AuctionCurrentBidder = buyerAddress
	dex.AuctionBidCount = dex.AuctionBidCount + 1

	// buynow
	if isBuynow {
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: buyerAddress,
			Amount: "", TokenID: "", TradeAmount: dex.Amount, TradeID: dex.MRC402, PayType: "mrc402_recv_item"})
		return mrc402DexProcess(stub, dex, buyerWallet, PaymentInfo, MRC402MT_Auction, dex.Amount, args[3], args[4])
	}

	// not buy now
	if err = MRC010Subtract(stub, &buyerWallet, dex.SellToken, newBidPrice.String(), MRC010MT_Normal); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, buyerWallet, "transfer_mrc402bid",
		[]string{buyerAddress, dex.Id, newBidPrice.String(), dex.SellToken, args[3], "0", "", dex.MRC402, args[4]}); err != nil {
		return err
	}

	if util.IsAddress(refunderAddress) {
		if refunderWallet, err = GetAddressInfo(stub, refunderAddress); err != nil {
			return err
		}
		if err = MRC010Add(stub, &refunderWallet, dex.SellToken, oldBidPrice.String(), 0); err != nil {
			return err
		}
		if err = SetAddressInfo(stub, refunderWallet, "receive_mrc402refund",
			[]string{dex.Id, refunderAddress, oldBidPrice.String(), dex.SellToken, args[3], "0", "", dex.MRC402, args[4]}); err != nil {
			return err
		}
	}

	// save bid info
	if err = setDEX402(stub, dex, "mrc402_auctionbid", []string{dex.Id, dex.Seller, buyerAddress, util.JSONEncode(PaymentInfo), args[3], args[4]}); err != nil {
		return err
	}
	return nil
}

/*
경매에 실패한 아이템이나
경매 기간이 종료된 아이템에 대하여 자산을 정산합니다.

별도의 서명 없이 작동됩니다.

args[0]:
	string[40] mrc402dexid
*/
func Mrc402AuctionFinish(stub shim.ChaincodeStubInterface, args []string) error {
	var err error

	var dex TMRC402DEX
	var sellerWallet mtc.TWallet

	var buyerWallet mtc.TWallet

	var PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)

	if len(args) < 1 {
		return errors.New("1000,mrc402auctionfinish operation must include four arguments : " +
			"mrc402dexid")
	}
	// get item info
	if dex, _, err = GetDEX402(stub, args[0]); err != nil {
		return err
	}

	switch dex402Status(dex) {
	case MRC402DS_AUCTION_END:
		// OK
	case MRC402DS_SALE, MRC402DS_SOLDOUT:
		return errors.New("3004,DEX Item is not auction item")
	case MRC402DS_AUCTION_WAIT, MRC402DS_AUCTION:
		return errors.New("3004,It cannot be closed while the auction is pending")
	case MRC402DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is already finished")
	case MRC402DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	if dex.AuctionCurrentBidder != "" {
		if buyerWallet, err = GetAddressInfo(stub, dex.AuctionCurrentBidder); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.AuctionCurrentBidder,
			Amount: "", TokenID: "", TradeAmount: dex.Amount, TradeID: dex.MRC402, PayType: "mrc402_recv_item"})
		return mrc402DexProcess(stub, dex, buyerWallet, PaymentInfo, MRC402MT_Auction, dex.Amount, "", "")
	} else {

		if sellerWallet, err = GetAddressInfo(stub, dex.Seller); err != nil {
			return err
		}

		param := []string{dex.Id, dex.Seller, dex.Amount, dex.MRC402}
		if err = mrc402Add(stub, &sellerWallet, dex.MRC402, dex.Amount, MRC402MT_Auction); err != nil {
			return err
		}
		SetAddressInfo(stub, sellerWallet, "mrc402auctionfailure", param)
		return setDEX402(stub, dex, "mrc402_auctionfailure", param)
	}
}

// 1. 지급 내역 계산
// 2. 주소별 지급 내역 저장
// 3. DEx 저장.
func mrc402DexProcess(stub shim.ChaincodeStubInterface, dex TMRC402DEX, buyerWallet mtc.TWallet,
	PaymentInfo []mtc.TDexPaymentInfo, tradeType MRC402ModifyType, tradeAmount, sign, tkey string) error {
	var mrc402 TMRC402
	var receiveAmount, paymentAmount, tradeAmountDecimal decimal.Decimal
	var commission decimal.Decimal
	var buyerAddress string
	var walletData mtc.TWallet
	var addrParams []string
	var err error
	var checkAddr string
	var dexType, jobType, sellerType string
	type RecvMapType struct {
		fromAddr    string
		toAddr      string
		amount      decimal.Decimal
		tradeAmount string
		jobType     string
	}
	var RecvMap map[string]RecvMapType

	if tradeType == MRC402MT_Auction {
		buyerAddress = dex.AuctionCurrentBidder
		sellerType = "mrc402_recv_auction"
		if dex.AuctionCurrentPrice == dex.AuctionBuyNowPrice {
			dexType = "mrc402_auctionbuynow"
		} else {
			dexType = "mrc402_auctionwinning"
		}
		receiveAmount, _ = decimal.NewFromString(dex.AuctionCurrentPrice)
		paymentAmount = receiveAmount
	} else if tradeType == MRC402MT_Sell {
		buyerAddress = dex.Buyer
		sellerType = "mrc402_recv_sell"
		dexType = "mrc402_buy"
		receiveAmount, _ = decimal.NewFromString(dex.SellPrice)
		tradeAmountDecimal, _ = decimal.NewFromString(tradeAmount)
		paymentAmount = tradeAmountDecimal.Mul(receiveAmount)
		receiveAmount = paymentAmount
	}

	// total payment price.
	if mrc402, _, err = GetMRC402(stub, dex.MRC402); err != nil {
		return err
	}

	// 3. creator commission calc
	if commission, err = DexFeeCalc(paymentAmount, mrc402.CreatorCommission, dex.SellToken); err == nil {
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: mrc402.Creator,
			Amount: commission.String(), TokenID: dex.SellToken, PayType: "mrc402_recv_fee_creator"})
		receiveAmount = receiveAmount.Sub(commission)
	}

	// 4. platform commission calc
	if util.IsAddress(dex.PlatformAddress) {
		if commission, err = DexFeeCalc(paymentAmount, dex.PlatformCommission, dex.SellToken); err == nil {
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.PlatformAddress,
				Amount: commission.String(), TokenID: dex.SellToken, PayType: "mrc402_recv_fee_platform"})
			receiveAmount = receiveAmount.Sub(commission)
		}
	}

	// 5. shareholder commission calc
	for shareholderAddress, shcomm := range mrc402.ShareHolder {
		if commission, err = DexFeeCalc(paymentAmount, shcomm, dex.SellToken); err == nil {
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: shareholderAddress,
				Amount: commission.String(), TokenID: dex.SellToken, PayType: "mrc402_recv_fee_shareholder"})
			receiveAmount = receiveAmount.Sub(commission)
		}
	}

	// recv sell price.
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.Seller,
		Amount: receiveAmount.String(), TokenID: dex.SellToken, PayType: sellerType})

	// payinfo grouping
	RecvMap = make(map[string]RecvMapType)
	Nev, _ := decimal.NewFromString("-1")
	for _, pi := range PaymentInfo {
		tAmount, _ := decimal.NewFromString(pi.Amount)
		if (pi.PayType == "mrc402_bid") || (pi.PayType == "mrc402_buy") {
			tAmount = tAmount.Mul(Nev)
			checkAddr = pi.FromAddr
		} else {
			checkAddr = pi.ToAddr
		}

		if pi.PayType == "mrc402_recv_item" {
			if err = mrc402Add(stub, &buyerWallet, dex.MRC402, pi.TradeAmount, MRC402MT_Normal); err != nil {
				return err
			}
		}

		if dt, exists := RecvMap[checkAddr]; !exists {
			RecvMap[checkAddr] = RecvMapType{
				fromAddr:    pi.FromAddr,
				toAddr:      pi.ToAddr,
				amount:      tAmount,
				jobType:     pi.PayType,
				tradeAmount: pi.TradeAmount,
			}
		} else {
			// overwrite mrc402_recv_fee_* type
			if strings.Index(dt.jobType, "mrc402_recv_fee_") == 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

				// overwrite mrc402_recv_item type but current type is not mrc402_recv_fee_
			} else if dt.jobType == "mrc402_recv_item" && strings.Index(pi.PayType, "mrc402_recv_fee_") != 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

				// overwrite mrc402_recv_refund type
			} else if dt.jobType == "mrc402_recv_refund" {
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
		case "mrc402_recv_item": // dex => buyer mrc402
			jobType = "receive_mrc402item"
		case "mrc402_buy": // buyer => dex	구매비용 지불(MRC010)
			jobType = "transfer_mrc402buy"
		case "mrc402_bid": // buyer => dex	입찰 비용 지불(MRC010)
			jobType = "transfer_mrc402bid"
		case "mrc402_recv_sell": // dex => seller	판매자 대금 받음(MRC010)
			jobType = "receive_mrc402sell"
		case "mrc402_recv_auction": // dex => seller	경매 낙찰금액 받음(MRC010)
			jobType = "receive_mrc402auction"
		case "mrc402_recv_refund": // dex=>refund	입찰 대금 환불(MRC010)
			jobType = "receive_mrc402refund"
		case "mrc402_recv_fee_creator": // dex => creator(MRC010)
			jobType = "receive_mrc402fee"
		case "mrc402_recv_fee_platform": // dex => platform(MRC010)
			jobType = "receive_mrc402fee"
		case "mrc402_recv_fee_shareholder": // dex => shareholder(MRC010)
			jobType = "receive_mrc402fee"
		}

		// add mrc402 buy amount to buyer
		if (v.jobType == "mrc402_bid") || (v.jobType == "mrc402_buy") {
			checkAddr = v.fromAddr
		} else {
			checkAddr = v.toAddr
		}

		if (checkAddr == dex.Buyer) || (checkAddr == dex.AuctionCurrentBidder) {
			walletData = buyerWallet
		} else {
			if walletData, err = GetAddressInfo(stub, checkAddr); err != nil {
				return err
			}
		}
		if v.amount.Cmp(decimal.Zero) < 0 {
			if err = MRC010Subtract(stub, &walletData, dex.SellToken, v.amount.Abs().String(), MRC010MT_Normal); err != nil {
				return err
			}
		} else if v.amount.Cmp(decimal.Zero) > 0 {
			if err = MRC010Add(stub, &walletData, dex.SellToken, v.amount.Abs().String(), 0); err != nil {
				return err
			}
		} else if v.jobType != "mrc402_recv_item" {
			continue
		}

		if checkAddr == dex.Seller {
			if tradeType == MRC402MT_Auction {
				if err = mrc402SubtractSubBalance(stub, &walletData, dex.MRC402, tradeAmount, MRC402MT_Auction); err != nil {
					return err
				}
			} else if tradeType == MRC402MT_Sell {
				if err = mrc402SubtractSubBalance(stub, &walletData, dex.MRC402, tradeAmount, MRC402MT_Sell); err != nil {
					return err
				}
			}
		}

		addrParams = []string{v.fromAddr, v.toAddr, v.amount.Abs().String(), dex.SellToken, sign, "0", v.tradeAmount, dex.MRC402, tkey}
		if err = SetAddressInfo(stub, walletData, jobType, addrParams); err != nil {
			return err
		}
	}

	// dex save
	dex.Buyer = buyerAddress
	addrParams = []string{dex.Id, dex.Seller, buyerAddress, util.JSONEncode(PaymentInfo), dex.MRC402}
	if err = setDEX402(stub, dex, dexType, addrParams); err != nil {
		return err
	}

	return nil
}
