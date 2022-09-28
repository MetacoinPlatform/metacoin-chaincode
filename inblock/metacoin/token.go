package metacoin

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/shopspring/decimal"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

type TMRC010DEX struct {
	Id           string `json:"dexid"`
	MRC010       string `json:"mrc010"` // MRC010 ID
	Seller       string `json:"seller"` // 판매자
	Buyer        string `json:"buyer"`  // 판매자
	Amount       string `json:"amount"` // 총 판매/구매 수량
	RemainAmount string `json:"remain_amount"`

	PlatformName       string `json:"platform_name"` // 플렛폼 이름
	PlatformURL        string `json:"platform_url"`
	PlatformAddress    string `json:"platform_address"`    // 판매/경매시 수수료를 받을 플렛폼 주소
	PlatformCommission string `json:"platform_commission"` // 판매/경매시 수수료를 받을 플렛폼가 가져가는 수수료 (0~10%)

	RegDate    int64 `json:"regdate"`     // 등록 일시		must > 0
	SellDate   int64 `json:"sell_date"`   // 거래 완료 일시  0 : not sale or auction finish, > 0 : sale or auction win
	CancelDate int64 `json:"cancel_date"` // 취소 일시		0 : not cancel, > 0 : canceled

	SellPrice     string `json:"sell_price"`      // 판매 금액, 만약, "MRC010" 의  decimal 이 2 이라면, 100 개당 판매 금액
	SellToken     string `json:"sell_token"`      // 구매 토큰
	BuyPrice      string `json:"buy_price"`       // 구매 금액, 만약, "MRC010" 의  decimal 이 2 이라면, 100 개당 구매 금액
	BuyToken      string `json:"buy_token"`       // 판매 토큰
	BuyTotalPrice string `json:"buy_price_total"` // 플렛폼 수수료가 포함된 총 구매 금액

	AuctionBidCount      int    `json:"auction_bid_count"`
	AuctionStartDate     int64  `json:"auction_start_date"`     // 경매 시작 일시	> 0 : auction item, 0: not auction item
	AuctionEndDate       int64  `json:"auction_end_date"`       // 경매 종료 일시   > 0 : auction item
	AuctionSettledDate   int64  `json:"auction_settle_date"`    // 경매 정산 일시   > 0 : trade or auction settled, 0: not yet.
	AuctionBiddingUnit   string `json:"auction_bidding_unit"`   // 경매 최소 입찰 단위	0 : free bidding
	AuctionStartPrice    string `json:"auction_start_price"`    // 경매 시작 금액	> 0 : auction item
	AuctionBuyNowPrice   string `json:"auction_buynow_price"`   // 경매 즉시 구매 금액
	AuctionCurrentPrice  string `json:"auction_current_price"`  // 경매 현 금액		"" : nothing bidder
	AuctionCurrentBidder string `json:"auction_current_bidder"` // 현재 입찰자		"" : nothing bidder

	JobType string `json:"job_type"`
	JobArgs string `json:"job_args"`
	JobDate int64  `json:"jobdate"`
}

type MRC010ModifyType int

const (
	MRC010MT_Normal = iota
	MRC010MT_Sell
	MRC010MT_Buy
	MRC010MT_Auction
)

type MRC010DexStatus int

const (
	MRC010DS_UNKNOWN        = iota // auction finish
	MRC010DS_SELL                  // on sale
	MRC010DS_BUY                   // on buy == request sell
	MRC010DS_CANCLED               // cancled
	MRC010DS_SOLDOUT               // saled
	MRC010DS_AUCTION_WAIT          // wait for auction start
	MRC010DS_AUCTION               // auction(biddable)
	MRC010DS_AUCTION_END           // auction end
	MRC010DS_AUCTION_FINISH        // auction finish

)

// setMRC010 : save token info
func setMRC010(stub shim.ChaincodeStubInterface, tk mtc.TMRC010, JobType string, args []string) error {
	var dat []byte
	var err error

	tk.JobType = JobType
	tk.JobDate = time.Now().Unix()
	if len(args) > 0 {
		if dat, err = json.Marshal(args); err == nil {
			tk.JobArgs = string(dat)
		}
	} else {
		tk.JobArgs = ""
	}

	if dat, err = json.Marshal(tk); err != nil {
		return errors.New("4204,Invalid token data format")
	}
	if err = stub.PutState("TOKEN_DATA_"+tk.Id, dat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// GetMRC010 : get token info
func GetMRC010(stub shim.ChaincodeStubInterface, TokenID string) (mtc.TMRC010, int, error) {
	var data []byte
	var tk mtc.TMRC010
	var err error
	var TokenSN int

	if len(TokenID) == 0 {
		return tk, 0, errors.New("4002,Token id missing")
	}
	if TokenSN, err = util.Strtoint(TokenID); err != nil {
		return tk, 0, errors.New("4104,Invalid toekn SN")
	}
	if data, err = stub.GetState("TOKEN_DATA_" + TokenID); err != nil {
		return tk, TokenSN, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return tk, TokenSN, errors.New("4001,Token " + TokenID + " not exists")
	}
	if err = json.Unmarshal(data, &tk); err != nil {
		return tk, TokenSN, errors.New("4204,Invalid token data format - " + err.Error())
	}
	if tk.Id == "" {
		tk.Id = TokenID
	}
	return tk, TokenSN, nil
}

// TokenRegister - Token Register.
func TokenRegister(stub shim.ChaincodeStubInterface, data, signature, tkey string) (string, error) {
	var dat []byte
	var value []byte
	var err error
	var tk mtc.TMRC010
	var currNo int
	var reserveInfo mtc.TMRC010Reserve
	var OwnerData, reserveAddr mtc.TWallet
	var t, RemainSupply decimal.Decimal

	// unmarshal data.
	if err = json.Unmarshal([]byte(data), &tk); err != nil {
		return "", errors.New("4105,Invalid token Data format")
	}

	// data check
	if len(tk.Symbol) < 1 {
		return "", errors.New("1002,Symbol is empty")
	}

	if len(tk.Name) < 1 {
		return "", errors.New("1001,Name is empty")
	}

	if tk.Decimal < 0 {
		return "", errors.New("1003,The decimal number must be bigger then 0")
	}

	if tk.Decimal > 18 {
		return "", errors.New("1004,The decimal number must be less than 18")
	}

	if value, err = stub.GetState("TOKEN_MAX_NO"); err != nil {
		return "", errors.New("8100,Hyperledger internal error - " + err.Error())
	}

	// set Token ID
	if value == nil {
		currNo = 0
	} else {
		currNo64, _ := strconv.ParseInt(string(value), 10, 32)
		currNo = int(currNo64)
		currNo = currNo + 1
	}

	if OwnerData, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return "", err
	}

	if err = NonceCheck(&OwnerData, tkey,
		strings.Join([]string{tk.Owner, tk.Name, tkey}, "|"),
		signature); err != nil {
		return "", err
	}

	if len(tk.Type) == 0 {
		tk.Type = "010"
	}
	tk.Token = currNo
	tk.JobDate = time.Now().Unix()
	tk.CreateDate = time.Now().Unix()
	tk.JobType = "tokenRegister"
	tk.Id = strconv.Itoa(currNo)

	if dat, err = json.Marshal(tk); err != nil {
		return "", errors.New("4200,Invalid Data format")
	}
	tk.JobArgs = string(dat)
	if dat, err = json.Marshal(tk); err != nil {
		return "", errors.New("4200,Invalid Data format")
	}

	if err = stub.PutState("TOKEN_DATA_"+strconv.Itoa(currNo), dat); err != nil {
		return "", err
	}

	if RemainSupply, err = util.ParsePositive(tk.TotalSupply); err != nil {
		return "", errors.New("1101,TotalSupply is not positive integer")
	}

	for _, reserveInfo = range tk.Reserve {
		if t, err = util.ParsePositive(reserveInfo.Value); err != nil {
			return "", errors.New("1102,Reserve amount " + reserveInfo.Value + " is not interger")
		}
		if reserveAddr, err = GetAddressInfo(stub, reserveInfo.Address); err != nil {
			return "", errors.New("1102,Token reserve address " + reserveInfo.Address + " not found")
		}
		if currNo == 0 {
			reserveAddr.Balance[0].Balance = reserveInfo.Value
		} else {
			reserveAddr.Balance = append(reserveAddr.Balance, mtc.TMRC010Balance{Balance: reserveInfo.Value, Token: currNo, UnlockDate: reserveInfo.UnlockDate})
		}

		RemainSupply := RemainSupply.Sub(t)
		if RemainSupply.IsNegative() {
			return "", errors.New("1103,The reserve amount is greater than totalsupply")
		}
		if err = SetAddressInfo(stub, reserveAddr, "token_reserve", []string{tk.Owner, reserveInfo.Address, reserveInfo.Value, strconv.Itoa(currNo)}); err != nil {
			return "", err
		}
	}

	if err = stub.PutState("TOKEN_MAX_NO", []byte(strconv.Itoa(currNo))); err != nil {
		return "", err
	}

	return strconv.Itoa(currNo), nil
}

// TokenAddLogger - MRC100 token logger add
func TokenAddLogger(stub shim.ChaincodeStubInterface, TokenID, logger, signature, tkey string, args []string) error {
	var tk mtc.TMRC010
	var err error
	var mwOwner mtc.TWallet

	if tk, _, err = GetMRC010(stub, TokenID); err != nil {
		return err
	}

	if _, err = GetAddressInfo(stub, logger); err != nil {
		return errors.New("1202,The Logger is not exists")
	}

	if mwOwner, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&mwOwner, tkey,
		strings.Join([]string{tk.Owner, TokenID, logger, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if logger == tk.Owner {
		return errors.New("1201,The Logger is the same as the token owner")
	}

	if tk.Logger == nil {
		tk.Logger = make(map[string]int64)
	} else {
		if _, exists := tk.Logger[logger]; exists {
			return errors.New("4205,Target token are in the target token list")
		}
	}
	tk.Logger[logger] = time.Now().Unix()

	return setMRC010(stub, tk, "tokenAddLogger", args)
}

// TokenRemoveLogger - MRC100 token logger remove
func TokenRemoveLogger(stub shim.ChaincodeStubInterface, TokenID, logger, signature, tkey string, args []string) error {
	var tk mtc.TMRC010
	var err error
	var mwOwner mtc.TWallet

	if tk, _, err = GetMRC010(stub, TokenID); err != nil {
		return err
	}

	if tk.Logger == nil {
		return errors.New("4202,Could not find logger in the logger list")
	}
	if _, exists := tk.Logger[logger]; !exists {
		return errors.New("4202,Could not find logger in the logger list")
	}

	if mwOwner, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&mwOwner, tkey,
		strings.Join([]string{tk.Owner, TokenID, logger, tkey}, "|"),
		signature); err != nil {
		return err
	}

	delete(tk.Logger, logger)

	return setMRC010(stub, tk, "tokenRemoveLogger", args)
}

// TokenUpdate - Token Information update.
func TokenUpdate(stub shim.ChaincodeStubInterface, TokenID, url, info, image, signature, tkey string, args []string) error {
	var tk mtc.TMRC010
	var err error
	var ownerData mtc.TWallet
	var isUpdate bool

	if tk, _, err = GetMRC010(stub, TokenID); err != nil {
		return err
	}

	if ownerData, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, tkey,
		strings.Join([]string{TokenID, url, info, image, tkey}, "|"),
		signature); err != nil {
		return err
	}

	isUpdate = false

	if tk.URL != url {
		tk.URL = url
		isUpdate = true
	}

	if tk.Information != info {
		tk.Information = info
		isUpdate = true
	}

	if len(image) > 0 && tk.Image != image {
		tk.Image = image
		isUpdate = true
	}

	if !isUpdate {
		return errors.New("4900,No data change")
	}

	return setMRC010(stub, tk, "tokenUpdate", args)
}

// TokenBurning - Token Information update.
func TokenBurning(stub shim.ChaincodeStubInterface, args []string) error {
	var tk mtc.TMRC010
	var err error
	var ownerData mtc.TWallet
	var memo string

	var BurnningAmount, BurnAmount decimal.Decimal

	if len(args) < 4 {
		return errors.New("1000,tokenBurn operation must include four arguments : TokenID, amount, memo, sign, tkey")
	}

	if tk, _, err = GetMRC010(stub, args[0]); err != nil {
		return err
	}

	if BurnAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1206,The amount must be an integer")
	}
	if err = util.DataAssign(args[2], &memo, "string", 0, 4096, true); err != nil {
		return errors.New("3005,Memo must be 1 to 4096 characters long")
	}

	if ownerData, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, args[4],
		strings.Join([]string{args[0], args[1], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = MRC010Subtract(stub, &ownerData, args[0], args[1], MRC010MT_Normal); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, ownerData, "ownerBurning", args); err != nil {
		return err
	}

	if BurnningAmount, err = util.ParseNotNegative(tk.BurnningAmount); err != nil {
		BurnningAmount = decimal.Zero
	}
	tk.BurnningAmount = BurnningAmount.Add(BurnAmount).String()

	return setMRC010(stub, tk, "tokenBurning", args)
}

// TokenIncrease - Token Information update.
func TokenIncrease(stub shim.ChaincodeStubInterface, args []string) error {
	var tk mtc.TMRC010
	var err error
	var ownerData mtc.TWallet
	var memo string
	var TotalAmount, IncrAmount decimal.Decimal

	if len(args) < 5 {
		return errors.New("1000,tokenIncrease operation must include four arguments : TokenID, amount, memo, sign, tkey")
	}

	if tk, _, err = GetMRC010(stub, args[0]); err != nil {
		return err
	}

	if IncrAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1206,amount must be a positive integer")
	}

	if err = util.DataAssign(args[2], &memo, "string", 0, 4096, true); err != nil {
		return errors.New("3005,Memo must be 1 to 4096 characters long")
	}

	if ownerData, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, args[4],
		strings.Join([]string{args[0], args[1], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	if err = MRC010Add(stub, &ownerData, args[0], args[1], 0); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, ownerData, "ownerIncrease", args); err != nil {
		return err
	}

	if TotalAmount, err = util.ParseNotNegative(tk.TotalSupply); err != nil {
		TotalAmount = decimal.Zero
	}
	tk.TotalSupply = TotalAmount.Add(IncrAmount).String()

	return setMRC010(stub, tk, "tokenIncrease", args)
}

// 잔액 감소
//
// Example:
//
func mrc010SubtractSubBalance(stub shim.ChaincodeStubInterface,
	wallet *mtc.TWallet, mrc010id string, amount string, SubtractType MRC010ModifyType) error {
	var err error
	var subtractAmount decimal.Decimal
	var tokenID int

	if subtractAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1104," + amount + " is not positive integer")
	}

	if _, tokenID, err = GetMRC010(stub, mrc010id); err != nil {
		return err
	}

	for index, element := range wallet.Balance {
		// is empty elements ?
		if element.Token != tokenID {
			continue
		}
		if element.UnlockDate != 0 {
			continue
		}

		if SubtractType == MRC010MT_Sell {
			sa, _ := decimal.NewFromString(wallet.Balance[index].SaleAmount)
			wallet.Balance[index].SaleAmount = sa.Sub(subtractAmount).String()
		} else if SubtractType == MRC010MT_Auction {
			sa, _ := decimal.NewFromString(wallet.Balance[index].AuctionAmount)
			wallet.Balance[index].AuctionAmount = sa.Sub(subtractAmount).String()
		}
		return nil
	}
	return errors.New("5000,Not enough balance")
}

// GetDEX010 get MRC010 Dex item
//
// Example :
//
//	TMRC010DEX, err := GetDEX010(stub, "MRC010Dex ITEM ID")
func GetDEX010(stub shim.ChaincodeStubInterface, dexid string) (TMRC010DEX, []byte, error) {
	var byte_data []byte
	var err error
	var mrc010dex TMRC010DEX

	if strings.Index(dexid, "DEX010_") != 0 || len(dexid) != 40 {
		return mrc010dex, nil, errors.New("6102,invalid DEX010 ID")
	}

	byte_data, err = stub.GetState(dexid)
	if err != nil {
		return mrc010dex, nil, errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if byte_data == nil {
		return mrc010dex, nil, errors.New("6004,MRC010 [" + dexid + "] not exist")
	}
	if err = json.Unmarshal(byte_data, &mrc010dex); err != nil {
		return mrc010dex, nil, err
	}
	return mrc010dex, byte_data, nil
}

func dex010Status(dex TMRC010DEX) MRC010DexStatus {
	var now = time.Now().Unix()
	if dex.CancelDate > 0 {
		return MRC010DS_CANCLED // Sale or auction canceled
	}

	if dex.AuctionStartDate == 0 {
		if dex.SellDate != 0 {
			return MRC010DS_SOLDOUT // solded
		}
		if dex.SellPrice != "" {
			return MRC010DS_SELL // on sale
		}
		if dex.BuyPrice != "" {
			return MRC010DS_BUY // on sale
		}
	} else {
		if dex.AuctionStartDate > now {
			return MRC010DS_AUCTION_WAIT // wait for auction
		}
		if dex.AuctionEndDate > now {
			return MRC010DS_AUCTION // in auction
		}
		if dex.AuctionSettledDate == 0 {
			return MRC010DS_AUCTION_END // auction finish but not yet price calc
		} else {
			return MRC010DS_AUCTION_FINISH // auction finish && price complete.
		}
	}
	return MRC010DS_UNKNOWN
}

// Mrc4dex010set02set set MRC010 Dex item
//
// Example :
//
//	err := setDEX010(stub, TMRC010DEX, "MRC010Dex ITEM ID", "jobtype", arguments)
func setDEX010(stub shim.ChaincodeStubInterface, MRC010DexItem TMRC010DEX, jobType string, jobArgs []string) error {
	var err error
	var byte_data []byte

	if strings.Index(MRC010DexItem.Id, "DEX010_") != 0 || len(MRC010DexItem.Id) != 40 {
		return errors.New("6102,invalid DEX010 data address")
	}

	MRC010DexItem.JobType = jobType
	MRC010DexItem.JobDate = time.Now().Unix()
	if byte_data, err = json.Marshal(jobArgs); err == nil {
		MRC010DexItem.JobArgs = string(byte_data)
	}

	if MRC010DexItem.RegDate == 0 {
		MRC010DexItem.RegDate = MRC010DexItem.JobDate
	}

	if (jobType == "mrc010_unsell" || jobType == "mrc010_unreqsell") && (MRC010DexItem.CancelDate == 0) {
		MRC010DexItem.CancelDate = MRC010DexItem.JobDate
	}

	if (jobType == "mrc010_unauction") && (MRC010DexItem.CancelDate == 0) {
		MRC010DexItem.CancelDate = MRC010DexItem.JobDate
	}

	if (jobType == "mrc010_buy" || jobType == "mrc010_acceptreqsell") && (MRC010DexItem.SellDate == 0) {
		if MRC010DexItem.RemainAmount == "0" {
			MRC010DexItem.SellDate = MRC010DexItem.JobDate
		}
	}

	if (jobType == "mrc010_auctionwinning" || jobType == "mrc010_auctionbuynow") && (MRC010DexItem.AuctionSettledDate == 0) {
		MRC010DexItem.SellDate = MRC010DexItem.JobDate
		MRC010DexItem.AuctionSettledDate = MRC010DexItem.JobDate
	} else if (jobType == "mrc010_auctionfailure") && (MRC010DexItem.AuctionSettledDate == 0) {
		MRC010DexItem.AuctionSettledDate = MRC010DexItem.JobDate
	}

	if byte_data, err = json.Marshal(MRC010DexItem); err != nil {
		return errors.New("3209,Invalid MRC010DexItem data format")
	}

	if err := stub.PutState(MRC010DexItem.Id, byte_data); err != nil {
		return errors.New("8600,dex010set stub.PutState [" + MRC010DexItem.Id + "] Error " + err.Error())
	}
	return nil
}

// Mrc010Sell Mrc010Sell
func Mrc010Sell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var mrc010 mtc.TMRC010
	var sellAmount, unitPrice, totalPrice decimal.Decimal
	var dex TMRC010DEX
	var token mtc.TMRC010
	var argdat []byte

	if len(args) < 11 {
		return errors.New("1000,mrc010sell operation must include four arguments : " +
			"seller, amount, mrc010id, sellPrice, selltoken, " +
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

	// 2 mrc010id
	if mrc010, _, err = GetMRC010(stub, args[2]); err != nil {
		return err
	}

	// seller : Decrease MRC010 by Amount
	// seller : SellToken increases by Amount * SellPrice
	dex = TMRC010DEX{
		MRC010:           mrc010.Id,           // Sale Token ID(Token ID to send)
		Seller:           sellerWallet.Id,     // Seller address
		Amount:           sellAmount.String(), // Total sale amount
		RemainAmount:     sellAmount.String(), // remain sale amount
		SellToken:        args[4],             // Payment Token(Token ID to receive)
		SellPrice:        "",                  // Amount per sale token
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
	if token, _, err = GetMRC010(stub, dex.SellToken); err != nil {
		return err
	}
	if mrc010.Id == token.Id {
		return errors.New("3005,The sale token must be different from the payment token")
	}

	// 5 platformname
	if err = util.DataAssign(args[5], &dex.PlatformName, "", 1, 256, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 6 PlatformURL
	if err = util.DataAssign(args[6], &dex.PlatformURL, "url", 1, 256, true); err != nil {
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

	if err = MRC010Subtract(stub, &sellerWallet, mrc010.Id, sellAmount.String(), MRC010MT_Sell); err != nil {
		return err
	}

	// generate Mrc010 ID
	var isSuccess = false
	temp := util.GenerateKey("DEX010_", args)
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

	if err = setDEX010(stub, dex, "mrc010_sell", params); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, sellerWallet, "mrc010sell", params); err != nil {
		return err
	}
	return nil
}

// Mrc010SellRequest Mrc010SellRequest
func Mrc010ReqSell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var buyerWallet mtc.TWallet
	var mrc010 mtc.TMRC010
	var buyAmount, unitPrice, totalPrice, commission decimal.Decimal
	var dex TMRC010DEX
	var token mtc.TMRC010
	var argdat []byte

	if len(args) < 11 {
		return errors.New("1000,mrc010requestsell operation must include four arguments : " +
			"seller, amount, mrc010id, buyPrice, buytoken, " +
			"platformName, platformURL, platformAddress, platformCommission, " +
			"signature, nonce")
	}

	// 0 seller
	if buyerWallet, err = GetAddressInfo(stub, args[0]); err != nil {
		return err
	}

	// 1 amount
	if buyAmount, err = util.ParsePositive(args[1]); err != nil {
		return errors.New("1107," + args[1] + " is not positive integer")
	}

	// 2 mrc010id
	if mrc010, _, err = GetMRC010(stub, args[2]); err != nil {
		return err
	}

	// seller : Decrease BuyToken  by Amount * BuyPrice
	// seller : MRC010 increases by Amount
	dex = TMRC010DEX{
		MRC010:           mrc010.Id,          // Buy Token ID(Toekn to receive)
		Buyer:            buyerWallet.Id,     // buyer address
		Amount:           buyAmount.String(), // Total buy amount
		RemainAmount:     buyAmount.String(), // remain buy amount
		BuyToken:         args[4],            // Payment token ID(Toekn id to send)
		BuyPrice:         "",                 // Amount per 1 sale token
		RegDate:          0,
		SellDate:         0,
		CancelDate:       0,
		AuctionStartDate: 0,
		AuctionEndDate:   0,
	}

	// 3 payment price
	if err = util.NumericDataCheck(args[3], &dex.BuyPrice, "1", "", 0, false); err != nil {
		return err
	}
	if unitPrice, err = util.ParsePositive(args[3]); err != nil {
		return err
	}

	totalPrice = buyAmount.Mul(unitPrice)
	if err = util.NumericDataCheck(totalPrice.String(), nil, "1", "", 0, false); err != nil {
		return errors.New("3005,The total purchase amount is too low")
	}

	// 4 payment token
	if token, _, err = GetMRC010(stub, dex.BuyToken); err != nil {
		return err
	}
	if mrc010.Id == token.Id {
		return errors.New("3005,The sale token must be different from the payment token")
	}

	// 5 platformname
	if err = util.DataAssign(args[5], &dex.PlatformName, "", 1, 256, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 6 PlatformURL
	if err = util.DataAssign(args[6], &dex.PlatformURL, "url", 1, 256, true); err != nil {
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

	if err = NonceCheck(&buyerWallet, args[10],
		strings.Join([]string{args[0], args[1], args[2], args[3], args[4],
			args[5], args[6], args[7], args[8],
			args[10]}, "|"),
		args[9]); err != nil {
		return err
	}

	if commission, err = DexFeeCalc(totalPrice, dex.PlatformCommission, dex.BuyToken); err == nil {
		totalPrice = totalPrice.Add(commission)
	}
	dex.BuyTotalPrice = totalPrice.String()
	if err = MRC010Subtract(stub, &buyerWallet, dex.BuyToken, totalPrice.String(), MRC010MT_Sell); err != nil {
		return err
	}

	// generate Mrc010 ID
	var isSuccess = false
	temp := util.GenerateKey("DEX010_", args)
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

	if err = setDEX010(stub, dex, "mrc010_reqsell", params); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, buyerWallet, "mrc010reqsell", params); err != nil {
		return err
	}
	return nil
}

// Mrc010Melt Mrc010Melt
func Mrc010UnSell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var dex TMRC010DEX

	if len(args) < 3 {
		return errors.New("1000,mrc010unsell operation must include four arguments : " +
			"dexid, signature, nonce")
	}

	// 0 mrc010id
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}
	if dex.AuctionStartDate > 0 {
		return errors.New("3004,DEX Item is not sell item")
	}
	if (dex.JobType != "mrc010_sell") && (dex.JobType != "mrc010_buy") {
		return errors.New("3004,DEX Item is not sell item")
	}
	switch dex010Status(dex) {
	case MRC010DS_SELL:
		// OK
	case MRC010DS_AUCTION_WAIT, MRC010DS_AUCTION, MRC010DS_AUCTION_END, MRC010DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is not sell item")
	case MRC010DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is already traded")
	case MRC010DS_BUY:
		return errors.New("3004,DEX Item is not sell item")
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

	if err = mrc010SubtractSubBalance(stub, &sellerWallet, dex.MRC010, dex.RemainAmount, MRC010MT_Sell); err != nil {
		return err
	}

	if err = MRC010Add(stub, &sellerWallet, dex.MRC010, dex.RemainAmount, 0); err != nil {
		return err
	}

	params := []string{dex.Id, dex.Seller, dex.RemainAmount, dex.MRC010, dex.SellPrice,
		dex.SellToken, args[1], args[2]}

	if err = setDEX010(stub, dex, "mrc010_unsell", params); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, sellerWallet, "mrc010unsell", params); err != nil {
		return err
	}
	return nil
}

// Mrc010Melt Mrc010Melt
func Mrc010UnReqSell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var buyerWallet mtc.TWallet
	var buyAmount decimal.Decimal
	var dex TMRC010DEX

	if len(args) < 3 {
		return errors.New("1000,mrc010unsell operation must include four arguments : " +
			"dexid, signature, nonce")
	}

	// 0 mrc010id
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}
	if dex.AuctionStartDate > 0 {
		return errors.New("3004,DEX Item is not sell item")
	}
	if (dex.JobType != "mrc010_reqsell") && (dex.JobType != "mrc010_acceptreqsell") {
		return errors.New("3004,DEX Item is not sell item")
	}
	switch dex010Status(dex) {
	case MRC010DS_BUY:
		// OK
	case MRC010DS_AUCTION_WAIT, MRC010DS_AUCTION, MRC010DS_AUCTION_END, MRC010DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is not sell item")
	case MRC010DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is already traded")
	case MRC010DS_SELL:
		return errors.New("3004,DEX Item is not sellreqeust item")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	// 1 seller
	if buyerWallet, err = GetAddressInfo(stub, dex.Buyer); err != nil {
		return err
	}

	if err = NonceCheck(&buyerWallet, args[2],
		strings.Join([]string{args[0], args[2]}, "|"),
		args[1]); err != nil {
		return err
	}
	if buyAmount, err = util.ParsePositive(dex.BuyTotalPrice); err != nil {
		return err
	}

	if err = mrc010SubtractSubBalance(stub, &buyerWallet, dex.BuyToken, buyAmount.String(), MRC010MT_Sell); err != nil {
		return err
	}
	if err = MRC010Add(stub, &buyerWallet, dex.BuyToken, buyAmount.String(), 0); err != nil {
		return err
	}

	params := []string{dex.Id, dex.Buyer, dex.RemainAmount, dex.MRC010, buyAmount.String(),
		dex.BuyToken, args[1], args[2]}

	if err = setDEX010(stub, dex, "mrc010_unreqsell", params); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, buyerWallet, "mrc010unreqsell", params); err != nil {
		return err
	}
	return nil
}

/*
Mrc010DEX item buy.
DEX status is must be MRC010DS_SALE
*/
func Mrc010Buy(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var buyerWallet mtc.TWallet
	var buyAmount, unitPrice, remainAmount decimal.Decimal
	var paymentPrice decimal.Decimal

	var dex TMRC010DEX
	var PaymentInfo []mtc.TDexPaymentInfo

	// argument check
	if len(args) < 5 {
		return errors.New("1000,mrc010unsell operation must include four arguments : " +
			"dexid, buyer, amount, signature, nonce")
	}

	// 0 mrc010id
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}

	// is sell item ?
	if dex.JobType != "mrc010_sell" && dex.JobType != "mrc010_buy" {
		return errors.New("3004,DEX Item is not request to sell item")
	}

	// status check
	switch dex010Status(dex) {
	case MRC010DS_SELL:
		// OK
	case MRC010DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is already traded")
	case MRC010DS_AUCTION_WAIT, MRC010DS_AUCTION, MRC010DS_AUCTION_END, MRC010DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is not sell item")
	case MRC010DS_BUY:
		return errors.New("3004,DEX Item is sell item")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	// self trade ?
	if dex.Seller == args[1] {
		return errors.New("3004,Seller is don't buy")
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

	paymentPrice = buyAmount.Mul(unitPrice)

	// dex.buyer => dex.id  paymentprice : dex.selltoken
	// dex.id => dex.buyer  buyamount: dex.mrc010
	dex.Buyer = args[1]
	PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)

	// buyer -> dex paymentprice selltoken
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Buyer, ToAddr: dex.Id,
		Amount: paymentPrice.String(), TokenID: dex.SellToken, TradeAmount: "", TradeID: "",
		PayType: "mrc010_buy"})

	// dex -> buyer buyamount mrc010
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.Buyer,
		Amount: "", TokenID: "", TradeAmount: buyAmount.String(), TradeID: dex.MRC010,
		PayType: "mrc010_recv_item"})

	return mrc010DexProcess(stub, dex, buyerWallet, PaymentInfo, MRC010MT_Sell,
		paymentPrice, buyAmount.String(), args[3], args[4])
}

/*
Mrc010DEX item accept by request sell.
DEX status is must be MRC010DS_SALE
*/
func Mrc010AcceptReqSell(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var sellAmount, unitPrice, remainAmount decimal.Decimal
	var sendPrice decimal.Decimal

	var dex TMRC010DEX
	var PaymentInfo []mtc.TDexPaymentInfo

	// argument check
	if len(args) < 5 {
		return errors.New("1000,mrc010unsell operation must include four arguments : " +
			"dexid, seller, amount, signature, nonce")
	}

	// 0 mrc010id
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}

	// is request sell item ?
	if dex.JobType != "mrc010_reqsell" && dex.JobType != "mrc010_acceptreqsell" {
		return errors.New("3004,DEX Item is not request to buy item")
	}

	// status check
	switch dex010Status(dex) {
	case MRC010DS_BUY:
		// OK
	case MRC010DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is already traded")
	case MRC010DS_AUCTION_WAIT, MRC010DS_AUCTION, MRC010DS_AUCTION_END, MRC010DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is not sell item")
	case MRC010DS_SELL:
		return errors.New("3004,DEX Item is not sell reqeust item")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	// self trade ?
	if dex.Buyer == args[1] {
		return errors.New("3004,Buyer is don't sell")
	}

	// 1 seller & sign check
	if sellerWallet, err = GetAddressInfo(stub, args[1]); err != nil {
		return err
	}
	if err = NonceCheck(&sellerWallet, args[4],
		strings.Join([]string{args[0], args[1], args[2], args[4]}, "|"),
		args[3]); err != nil {
		return err
	}

	// 2 sell amount check.
	if sellAmount, err = util.ParsePositive(args[2]); err != nil {
		return err
	}
	if remainAmount, err = util.ParsePositive(dex.RemainAmount); err != nil {
		return err
	}
	if remainAmount.Cmp(sellAmount) < 0 {
		return errors.New("3004,The quantity available for purchase is " + remainAmount.String() + " pieces.")
	}
	dex.RemainAmount = remainAmount.Sub(sellAmount).String()

	if unitPrice, err = util.ParsePositive(dex.BuyPrice); err != nil {
		return err
	}

	sendPrice = sellAmount.Mul(unitPrice)
	if sendPrice.Cmp(decimal.Zero) < 1 {
		return errors.New("3004,The payment amount is too low to purchase")
	}

	// dex.seller => dex.id  paymentprice : dex.mrc010
	// dex.id => dex.seller  buyamount * buyprice : dex.buytoken
	dex.Seller = args[1]
	PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)
	// seller -> dex sellamount mrc010
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Seller, ToAddr: dex.Id,
		Amount: "", TokenID: "", TradeAmount: sellAmount.String(), TradeID: dex.MRC010,
		PayType: "mrc010_sell"})

	// dex -> seller paymentprice dex.buytoken
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.Seller,
		Amount: sendPrice.String(), TokenID: dex.BuyToken, TradeAmount: "", TradeID: "",
		PayType: "mrc010_send_item"})

	return mrc010DexProcessReqSell(stub, dex, sellerWallet, PaymentInfo, MRC010MT_Buy,
		sendPrice, sellAmount.String(), args[3], args[4])
}

func Mrc010DexMatch(stub shim.ChaincodeStubInterface, args []string) error {
	return nil
}

/*
	Mrc010Auction
	args arguments
	seller, amount, mrc010id, auction_start_price, selltoken,
	auction_bidding_unit, auction_buynow_price, auction_start_date, auction_end_date, platformName,
	platformURL, platformAddress, platformCommission, signature, nonce
*/
func Mrc010Auction(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var mrc010 mtc.TMRC010
	var sellAmount, startPrice, buyNowPrice decimal.Decimal
	var dex TMRC010DEX
	var argdat []byte
	var now int64

	if len(args) < 15 {
		return errors.New("1000,mrc010auction operation must include four arguments : " +
			"address, amount, mrc010id, auction_start_price, selltoken, " +
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

	// 2 mrc010id
	if mrc010, _, err = GetMRC010(stub, args[2]); err != nil {
		return err
	}
	dex = TMRC010DEX{
		MRC010:               mrc010.Id,
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
	if err = util.DataAssign(args[9], &dex.PlatformName, "", 1, 256, true); err != nil {
		return errors.New("3005,Url value error : " + err.Error())
	}

	// 9 PlatformURL
	if err = util.DataAssign(args[10], &dex.PlatformURL, "url", 1, 256, true); err != nil {
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

	if err = MRC010Subtract(stub, &sellerWallet, mrc010.Id, sellAmount.String(), MRC010MT_Auction); err != nil {
		return err
	}

	// generate MRC010 ID
	var isSuccess = false
	temp := util.GenerateKey("DEX010_", args)
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

	// 	"seller, amount, mrc010id, auction_start_price, selltoken, " +
	//	"auction_bidding_unit, auction_buynow_price, auction_start_date, auction_end_date, platformName, " +
	//	"platformURL, platformAddress, platformCommission, signature, nonce")
	params := []string{dex.Id, args[0], args[1], args[2], args[3], args[4],
		args[5], args[6], args[7], args[8], args[9],
		args[10], args[11], args[12], args[14]}

	if err = setDEX010(stub, dex, "mrc010_auction", params); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, sellerWallet, "mrc010auction", params); err != nil {
		return err
	}
	return nil
}

/*
	Mrc010UnAuction

args arguments

	dexid, signature, nonce
*/
func Mrc010UnAuction(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var sellerWallet mtc.TWallet
	var dex TMRC010DEX

	if len(args) < 3 {
		return errors.New("1000,mrc010unauction operation must include four arguments : " +
			"mrc010dexid, signature, nonce")
	}

	// 0 mrc010id
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}
	if dex.AuctionStartDate == 0 {
		return errors.New("3004,DEX Item is not sell item")
	}

	switch dex010Status(dex) {
	case MRC010DS_AUCTION, MRC010DS_AUCTION_WAIT:
		// ok
	case MRC010DS_SELL, MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is not auction item")
	case MRC010DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	case MRC010DS_AUCTION_END:
		return errors.New("3004,DEX Item is already end, use auction finish")
	case MRC010DS_AUCTION_FINISH:
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

	if err = MRC010Add(stub, &sellerWallet, dex.MRC010, dex.Amount, 0); err != nil {
		return err
	}
	if err = mrc010SubtractSubBalance(stub, &sellerWallet, dex.MRC010, dex.Amount, MRC010MT_Sell); err != nil {
		return err
	}

	params := []string{dex.Id, dex.Seller, dex.Amount, dex.MRC010, dex.SellPrice, dex.SellToken, args[1], args[2]}

	if err = setDEX010(stub, dex, "mrc010_unauction", params); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, sellerWallet, "mrc010unauction", params); err != nil {
		return err
	}
	return nil
}

/*
경매중인 아이템 입찰

args arguments
- dexid, buyer, bidAmount, signature, nonce
*/
func Mrc010AuctionBid(stub shim.ChaincodeStubInterface, args []string) error {
	var err error
	var buyerWallet mtc.TWallet
	var dex TMRC010DEX
	var buyerAddress string
	var refunderWallet mtc.TWallet
	var refunderAddress string

	var PaymentInfo []mtc.TDexPaymentInfo
	PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)

	var buyNow, oldBidPrice, newBidPrice, bidUnit decimal.Decimal
	var isBuynow bool

	if len(args) < 5 {
		return errors.New("1000,mrc010bid operation must include four arguments : " +
			"mrc010dexid, address, amount, signature, nonce")
	}
	buyerAddress = args[1]

	// 0 mrc010id
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}

	switch dex010Status(dex) {
	case MRC010DS_AUCTION:
		// OK
	case MRC010DS_SELL, MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is not auction item")
	case MRC010DS_AUCTION_WAIT:
		return errors.New("3004,This is not the auction bidding period")
	case MRC010DS_AUCTION_END:
		return errors.New("3004,DEX Item is already end, use auction finish")
	case MRC010DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is already finished")
	case MRC010DS_CANCLED:
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
			Amount: dex.AuctionCurrentPrice, TokenID: dex.SellToken, PayType: "mrc010_recv_refund"})
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
		Amount: newBidPrice.String(), TokenID: dex.SellToken, TradeAmount: "", TradeID: "", PayType: "mrc010_bid"})

	// set new bidder
	dex.AuctionCurrentPrice = newBidPrice.String()
	dex.AuctionCurrentBidder = buyerAddress
	dex.AuctionBidCount = dex.AuctionBidCount + 1

	// buynow
	if isBuynow {
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: buyerAddress,
			Amount: "", TokenID: "", TradeAmount: dex.Amount, TradeID: dex.MRC010, PayType: "mrc010_recv_item"})
		return mrc010DexProcess(stub, dex, buyerWallet, PaymentInfo, MRC010MT_Auction,
			newBidPrice, dex.Amount, args[3], args[4])
	}

	// not buy now
	if err = MRC010Subtract(stub, &buyerWallet, dex.SellToken, newBidPrice.String(), MRC010MT_Normal); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, buyerWallet, "transfer_mrc010bid",
		[]string{buyerAddress, dex.Id, newBidPrice.String(), dex.SellToken, args[3], "0", "", dex.MRC010, args[4]}); err != nil {
		return err
	}

	if util.IsAddress(refunderAddress) {
		if refunderWallet, err = GetAddressInfo(stub, refunderAddress); err != nil {
			return err
		}
		if err = MRC010Add(stub, &refunderWallet, dex.SellToken, oldBidPrice.String(), 0); err != nil {
			return err
		}
		if err = SetAddressInfo(stub, refunderWallet, "receive_mrc010refund",
			[]string{dex.Id, refunderAddress, oldBidPrice.String(), dex.SellToken, args[3], "0", "", dex.MRC010, args[4]}); err != nil {
			return err
		}
	}

	// save bid info
	if err = setDEX010(stub, dex, "mrc010_auctionbid", []string{dex.Id, dex.Seller, buyerAddress, util.JSONEncode(PaymentInfo), args[3], args[4]}); err != nil {
		return err
	}
	return nil
}

/*
경매에 실패한 아이템이나
경매 기간이 종료된 아이템에 대하여 자산을 정산합니다.

별도의 서명 없이 작동됩니다.

args[0]:

	string[40] mrc010dexid
*/
func Mrc010AuctionFinish(stub shim.ChaincodeStubInterface, args []string) error {
	var err error

	var dex TMRC010DEX
	var sellerWallet mtc.TWallet

	var buyerWallet mtc.TWallet
	var paymentAmount decimal.Decimal
	var PaymentInfo = make([]mtc.TDexPaymentInfo, 0, 12)

	if len(args) < 1 {
		return errors.New("1000,mrc010auctionfinish operation must include four arguments : " +
			"mrc010dexid")
	}
	// get item info
	if dex, _, err = GetDEX010(stub, args[0]); err != nil {
		return err
	}

	switch dex010Status(dex) {
	case MRC010DS_AUCTION_END:
		// OK
	case MRC010DS_SELL, MRC010DS_SOLDOUT:
		return errors.New("3004,DEX Item is not auction item")
	case MRC010DS_AUCTION_WAIT, MRC010DS_AUCTION:
		return errors.New("3004,It cannot be closed while the auction is pending")
	case MRC010DS_AUCTION_FINISH:
		return errors.New("3004,DEX Item is already finished")
	case MRC010DS_CANCLED:
		return errors.New("3004,DEX Item is already canceled")
	default:
		return errors.New("3004,DEX Item status is unknown")
	}

	if dex.AuctionCurrentBidder != "" {
		if buyerWallet, err = GetAddressInfo(stub, dex.AuctionCurrentBidder); err != nil {
			return err
		}
		paymentAmount, _ = decimal.NewFromString(dex.AuctionCurrentPrice)

		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.AuctionCurrentBidder,
			Amount: "", TokenID: "", TradeAmount: dex.Amount, TradeID: dex.MRC010, PayType: "mrc010_recv_item"})
		return mrc010DexProcess(stub, dex, buyerWallet, PaymentInfo,
			MRC010MT_Auction, paymentAmount, dex.Amount, "", "")
	} else {
		if sellerWallet, err = GetAddressInfo(stub, dex.Seller); err != nil {
			return err
		}

		param := []string{dex.Id, dex.Seller, dex.Amount, dex.MRC010}
		if err = MRC010Add(stub, &sellerWallet, dex.MRC010, dex.Amount, 0); err != nil {
			return err
		}
		SetAddressInfo(stub, sellerWallet, "mrc010auctionfailure", param)
		return setDEX010(stub, dex, "mrc010_auctionfailure", param)
	}
}

// 1. 지급 내역 계산
// 2. 주소별 지급 내역 저장
// 3. DEx 저장.
func mrc010DexProcess(stub shim.ChaincodeStubInterface, dex TMRC010DEX, buyerWallet mtc.TWallet,
	PaymentInfo []mtc.TDexPaymentInfo, tradeType MRC010ModifyType, paymentAmount decimal.Decimal,
	tradeAmount, sign, tkey string) error {
	var receiveAmount decimal.Decimal
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

	if tradeType == MRC010MT_Auction {
		buyerAddress = dex.AuctionCurrentBidder
		sellerType = "mrc010_recv_auction"
		if dex.AuctionCurrentPrice == dex.AuctionBuyNowPrice {
			dexType = "mrc010_auctionbuynow"
		} else {
			dexType = "mrc010_auctionwinning"
		}
		receiveAmount = paymentAmount
	} else if tradeType == MRC010MT_Sell {
		buyerAddress = dex.Buyer
		sellerType = "mrc010_recv_sell"
		dexType = "mrc010_buy"
		receiveAmount = paymentAmount
	}

	// 4. platform commission calc
	if util.IsAddress(dex.PlatformAddress) {
		if commission, err = DexFeeCalc(paymentAmount, dex.PlatformCommission, dex.SellToken); err == nil {
			PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.PlatformAddress,
				Amount: commission.String(), TokenID: dex.SellToken, PayType: "mrc010_recv_fee_platform"})
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
		if (pi.PayType == "mrc010_bid") || (pi.PayType == "mrc010_buy") {
			tAmount = tAmount.Mul(Nev)
			checkAddr = pi.FromAddr
		} else {
			checkAddr = pi.ToAddr
		}

		if pi.PayType == "mrc010_recv_item" {
			if err = MRC010Add(stub, &buyerWallet, dex.MRC010, pi.TradeAmount, 0); err != nil {
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
			// overwrite mrc010_recv_fee_* type
			if strings.Index(dt.jobType, "mrc010_recv_fee_") == 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

				// overwrite mrc010_recv_item type but current type is not mrc010_recv_fee_
			} else if dt.jobType == "mrc010_recv_item" && strings.Index(pi.PayType, "mrc010_recv_fee_") != 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

				// overwrite mrc010_recv_refund type
			} else if dt.jobType == "mrc010_recv_refund" {
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
		case "mrc010_recv_item": // dex => buyer mrc010
			jobType = "receive_mrc010item"
		case "mrc010_buy": // buyer => dex      구매비용 지불(MRC010)
			jobType = "transfer_mrc010buy"
		case "mrc010_bid": // buyer => dex      입찰 비용 지불(MRC010)
			jobType = "transfer_mrc010bid"
		case "mrc010_recv_sell": // dex => seller       판매자 대금 받음(MRC010)
			jobType = "receive_mrc010sell"
		case "mrc010_recv_auction": // dex => seller    경매 낙찰금액 받음(MRC010)
			jobType = "receive_mrc010auction"
		case "mrc010_recv_refund": // dex=>refund       입찰 대금 환불(MRC010)
			jobType = "receive_mrc010refund"
		case "mrc010_recv_fee_creator": // dex => creator(MRC010)
			jobType = "receive_mrc010fee"
		case "mrc010_recv_fee_platform": // dex => platform(MRC010)
			jobType = "receive_mrc010fee"
		case "mrc010_recv_fee_shareholder": // dex => shareholder(MRC010)
			jobType = "receive_mrc010fee"
		}

		// add mrc010 buy amount to buyer
		if (v.jobType == "mrc010_bid") || (v.jobType == "mrc010_buy") {
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
		} else if v.jobType != "mrc010_recv_item" {
			continue
		}

		if checkAddr == dex.Seller {
			if tradeType == MRC010MT_Auction {
				if err = mrc010SubtractSubBalance(stub, &walletData, dex.MRC010, tradeAmount, MRC010MT_Auction); err != nil {
					return err
				}
			} else if tradeType == MRC010MT_Sell {
				if err = mrc010SubtractSubBalance(stub, &walletData, dex.MRC010, tradeAmount, MRC010MT_Sell); err != nil {
					return err
				}
			}
		}

		addrParams = []string{v.fromAddr, v.toAddr, v.amount.Abs().String(), dex.SellToken, sign, "0", v.tradeAmount, dex.MRC010, tkey}
		if err = SetAddressInfo(stub, walletData, jobType, addrParams); err != nil {
			return err
		}
	}

	// dex save
	dex.Buyer = buyerAddress
	addrParams = []string{dex.Id, dex.Seller, buyerAddress, util.JSONEncode(PaymentInfo), dex.MRC010}
	if err = setDEX010(stub, dex, dexType, addrParams); err != nil {
		return err
	}

	return nil
}

// 1. 지급 내역 계산
// 2. 주소별 지급 내역 저장
// 3. DEx 저장.
// dex : Dex Struct
// actorWallet : buyer or auction bidder
// PaymentInfo
// tradeType : sell or auction bidding
// paymentAmount : buyer or auction bidder => dex token amount
// tradeAmount : sell or auction item amount
func mrc010DexProcessReqSell(stub shim.ChaincodeStubInterface, dex TMRC010DEX, actorWallet mtc.TWallet,
	PaymentInfo []mtc.TDexPaymentInfo, tradeType MRC010ModifyType, paymentAmount decimal.Decimal,
	tradeAmount, sign, tkey string) error {
	var commission, tAmount, totalPayment decimal.Decimal
	var actorAddress string
	var walletData mtc.TWallet
	var addrParams []string
	var err error
	var checkAddr string
	var dexType, jobType, sellerType string
	var paymentToken string

	type RecvMapType struct {
		fromAddr    string
		toAddr      string
		amount      decimal.Decimal
		tradeAmount string
		jobType     string
	}
	var RecvMap map[string]RecvMapType
	fmt.Println("DEX", dex)
	fmt.Println("actorWallet", actorWallet)
	fmt.Println("PaymentInfo", PaymentInfo)
	fmt.Println("tradeType", tradeType)
	fmt.Println("paymentAmount", paymentAmount) // requester => seller
	fmt.Println("tradeAmount", tradeAmount)     // seller => requester

	if tradeType == MRC010MT_Buy { // reqsell => acceptreqsell
		actorAddress = dex.Seller
		sellerType = "mrc010_recv_buy"
		dexType = "mrc010_acceptreqsell"
		totalPayment = paymentAmount
		paymentToken = dex.BuyToken
	}

	// platform commission calc
	commission = decimal.Zero
	if util.IsAddress(dex.PlatformAddress) {
		fmt.Println("PlatformCommission", dex.PlatformCommission)
		fmt.Println("Commission tokenID", paymentToken)
		if commission, err = DexFeeCalc(paymentAmount, dex.PlatformCommission, paymentToken); err != nil {
			commission = decimal.Zero
		}
	}

	fmt.Println("Calc Commission", commission)
	// commission or remain price send to platform
	tAmount, _ = decimal.NewFromString(dex.BuyTotalPrice)
	tAmount = tAmount.Sub(paymentAmount)
	if commission.IsPositive() {
		if tAmount.Cmp(commission) < 0 {
			commission = tAmount
		}
		tAmount = tAmount.Sub(commission)
		totalPayment = totalPayment.Add(commission)
		// dex -> platform, commission dex.buytoken
		PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.PlatformAddress,
			Amount: commission.String(), TokenID: paymentToken, PayType: "mrc010_recv_fee_platform"})
	}

	dex.BuyTotalPrice = tAmount.String() // set remain token.
	// dex -> buyer sellamount
	PaymentInfo = append(PaymentInfo, mtc.TDexPaymentInfo{FromAddr: dex.Id, ToAddr: dex.Buyer,
		TradeAmount: tradeAmount, TradeID: dex.MRC010, PayType: sellerType})

	// payinfo grouping
	RecvMap = make(map[string]RecvMapType)
	Nev, _ := decimal.NewFromString("-1")
	for _, pi := range PaymentInfo {
		fmt.Println("\nLine 2003", pi.PayType, pi.FromAddr, "=>", pi.ToAddr,
			"Amount : ", pi.Amount, pi.TokenID,
			"Trade : ", pi.TradeAmount, pi.TradeID)

		if pi.PayType == "mrc010_sell" {
			tAmount, _ = decimal.NewFromString(pi.TradeAmount)
			pi.TradeAmount = tAmount.Mul(Nev).String()
			tAmount = decimal.Zero
			checkAddr = pi.FromAddr
		} else if pi.PayType == "mrc010_recv_buy" {
			tAmount = decimal.Zero
			checkAddr = pi.ToAddr
		} else {
			tAmount, _ = decimal.NewFromString(pi.Amount)
			checkAddr = pi.ToAddr
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
			// overwrite mrc010_recv_fee_* type
			if strings.Index(dt.jobType, "mrc010_recv_fee_") == 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

				// overwrite mrc010_recv_item type but current type is not mrc010_recv_fee_
			} else if dt.jobType == "mrc010_recv_item" && strings.Index(pi.PayType, "mrc010_recv_fee_") != 0 {
				dt.jobType = pi.PayType
				dt.fromAddr = pi.FromAddr
				dt.toAddr = pi.ToAddr

				// overwrite mrc010_recv_refund type
			} else if dt.jobType == "mrc010_recv_refund" {
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

		fmt.Println("RecvMap", checkAddr,
			RecvMap[checkAddr].jobType,
			RecvMap[checkAddr].fromAddr,
			RecvMap[checkAddr].toAddr,
			"Payment : ", RecvMap[checkAddr].amount,
			"TradeAmount : ", RecvMap[checkAddr].tradeAmount)
	}

	// payinfo save
	for _, v := range RecvMap {
		fmt.Println("\nLine 2063", v.jobType, v.fromAddr, "=>", v.toAddr, "Amount", v.amount.String(), "Trade", v.tradeAmount)
		switch v.jobType {
		case "mrc010_sell": // seller => dex  mrc010
			jobType = "transfer_mrc010sell"
		case "mrc010_send_item": // dex => seller buytoken
			jobType = "receive_mrc010sell"
		case "mrc010_recv_fee_platform": // dex => platform(MRC010)
			jobType = "receive_mrc010fee"
		case "mrc010_recv_buy": // dex => buyer	recv item
			jobType = "receive_mrc010buy"
		}
		// add mrc010 buy amount to buyer
		if v.jobType == "mrc010_sell" {
			checkAddr = v.fromAddr
		} else {
			checkAddr = v.toAddr
		}
		fmt.Println(v.jobType, jobType, checkAddr, v.amount.String())

		if tradeType == MRC010MT_Buy && checkAddr == dex.Seller {
			walletData = actorWallet
		} else {
			if walletData, err = GetAddressInfo(stub, checkAddr); err != nil {
				return err
			}
		}
		decTradeAmount, _ := decimal.NewFromString(v.tradeAmount)
		if decTradeAmount.Cmp(decimal.Zero) < 0 {
			fmt.Println("SubTract", walletData.Id, paymentToken, decTradeAmount.Abs().String())
			if err = MRC010Subtract(stub, &walletData, dex.MRC010, decTradeAmount.Abs().String(), MRC010MT_Normal); err != nil {
				return err
			}
		} else if decTradeAmount.Cmp(decimal.Zero) > 0 {
			fmt.Println("Add", walletData.Id, paymentToken, v.tradeAmount)
			if err = MRC010Add(stub, &walletData, dex.MRC010, v.tradeAmount, 0); err != nil {
				return err
			}
		}
		if v.amount.Cmp(decimal.Zero) > 0 {
			fmt.Println("Add", walletData.Id, paymentToken, v.amount.String())
			if err = MRC010Add(stub, &walletData, paymentToken, v.amount.String(), 0); err != nil {
				return err
			}
		}

		if decTradeAmount.Cmp(decimal.Zero) >= 0 &&
			v.amount.Cmp(decimal.Zero) <= 0 &&
			v.jobType != "mrc010_recv_item" && v.jobType != "mrc010_recv_buy" {
			continue
		}

		if tradeType == MRC010MT_Buy && checkAddr == dex.Buyer {
			fmt.Println("SubBalance", walletData.Id, paymentToken, totalPayment.String())
			if err = mrc010SubtractSubBalance(stub, &walletData, paymentToken, totalPayment.String(), MRC010MT_Sell); err != nil {
				return err
			}
		}

		addrParams = []string{v.fromAddr, v.toAddr, v.amount.Abs().String(), paymentToken, sign, "0", decTradeAmount.Abs().String(), dex.MRC010, tkey}
		if err = SetAddressInfo(stub, walletData, jobType, addrParams); err != nil {
			return err
		}
	}
	if tradeType == MRC010MT_Buy { // reqsell => acceptreqsell
		addrParams = []string{dex.Id, dex.Buyer, actorAddress, util.JSONEncode(PaymentInfo), dex.MRC010}
	}
	if err = setDEX010(stub, dex, dexType, addrParams); err != nil {
		return err
	}

	return nil
}
