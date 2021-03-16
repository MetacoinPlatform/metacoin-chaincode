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

// =========================================
// stodex.go
// =========================================

// Mrc040get - get MRC040 Token
func Mrc040get(stub shim.ChaincodeStubInterface, mrc040Key string) (string, error) {
	var dat []byte
	var err error
	var mrc040 mtc.ExchangeItem

	if strings.Index(mrc040Key, "MRC040_") != 0 {
		return "", errors.New("6103,invalid MRC040 data address")
	}

	dat, err = stub.GetState(mrc040Key)
	if err != nil {
		return "", errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return "", errors.New("6005,MRC040 data not exist")
	}
	if err = json.Unmarshal(dat, &mrc040); err != nil {
		return "", errors.New("6206,MRC040 [" + mrc040Key + "] is in the wrong data")
	}

	return string(dat), nil
}

// StodexRegister - STO token register for DEX
func StodexRegister(stub shim.ChaincodeStubInterface,
	owner, side, BaseToken, TargetToken, price, qtt, exchangeItemPK, signature, tkey string,
	args []string) error {
	var err error
	var Price, Qtt, TotalAmount, tempCoin decimal.Decimal
	var ownerData mtc.MetaWallet
	var exists bool
	var BaseTokenSN, TargetTokenSN int
	var BaseTokenData, TargetTokenData mtc.Token
	var tokenSN int
	var CurrentPending string
	var item mtc.ExchangeItem
	var data []byte

	if strings.Index(exchangeItemPK, "MRC040_") != 0 {
		return errors.New("6103,invalid MRC040 data address")
	}

	if data, err = stub.GetState(exchangeItemPK); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data != nil {
		return errors.New("6100,MRC040 [" + exchangeItemPK + "] is already exists")
	}

	// get owner info.
	if ownerData, err = GetAddressInfo(stub, owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, tkey,
		strings.Join([]string{owner, BaseToken, TargetToken, price, qtt, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if BaseTokenData, BaseTokenSN, err = GetToken(stub, BaseToken); err != nil {
		return err
	}

	if TargetTokenData, TargetTokenSN, err = GetToken(stub, TargetToken); err != nil {
		return err
	}

	// token pair check.
	if BaseTokenData.TargetToken == nil {
		return errors.New("6501,Base token is not allow exchange to target token")
	}

	if _, exists := BaseTokenData.TargetToken[TargetTokenSN]; exists == false {
		return errors.New("6501,Base token is not allow exchange to target token")
	}

	if TargetTokenData.BaseToken != BaseTokenSN {
		return errors.New("6502,Exchange token is not allow exchange to base token")
	}

	// price, qtt format check.
	if Price, err = util.ParsePositive(price); err != nil {
		return errors.New("1103,Price must be an integer string")
	}

	if Qtt, err = util.ParsePositive(qtt); err != nil {
		return errors.New("1105,Qtt must be an integer string")
	}

	if side == "SELL" {
		tokenSN = TargetTokenSN
		TotalAmount = Qtt
	} else if side == "BUY" {
		tokenSN = BaseTokenSN
		Divider, _ := decimal.NewFromString("10")
		TotalAmount = Price.Mul(Qtt).Div(Divider.Pow(decimal.New(int64(TargetTokenData.Decimal), 0)))
	} else {
		return errors.New("1205,Side must SELL or BUY")
	}

	// total amount, qtt precision check.
	if TotalAmount.Cmp(TotalAmount.Truncate(0)) != 0 {
		return errors.New("1203,Price precision is too long")
	}

	// collect token balance
	if err = SubtractToken(stub, &ownerData, strconv.Itoa(tokenSN), TotalAmount.String()); err != nil {
		return err
	}

	if ownerData.Pending == nil {
		ownerData.Pending = make(map[int]string)
	}
	CurrentPending, exists = ownerData.Pending[tokenSN]
	if exists {
		tempCoin, err = decimal.NewFromString(CurrentPending)
		ownerData.Pending[tokenSN] = tempCoin.Add(TotalAmount).String()
	} else {
		ownerData.Pending[tokenSN] = TotalAmount.String()
	}

	if err = SetAddressInfo(stub, owner, ownerData, "stodexRegister", args); err != nil {
		return err
	}

	item.Owner = owner
	item.Side = side
	item.BaseToken = BaseTokenSN
	item.TargetToken = TargetTokenSN
	item.Price = Price.String()
	item.Qtt = Qtt.String()
	item.RemainQtt = Qtt.String()
	item.Regdate = time.Now().Unix()
	item.CompleteDate = 0
	item.Status = "WAIT"
	item.Type = "MRC040"
	item.JobDate = time.Now().Unix()
	item.JobType = "stodexRegister"
	if args != nil && len(args) > 0 {
		if data, err = json.Marshal(args); err == nil {
			item.JobArgs = string(data)
		}
	} else {
		item.JobArgs = ""
	}

	buf, _ := json.Marshal(item)
	if err = stub.PutState(exchangeItemPK, buf); err != nil {
		fmt.Printf("stodexRegister stub.PutState(key, dat) [%s] Error %s\n", exchangeItemPK, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// StodexUnRegister - STO token unregister for DEX.
func StodexUnRegister(stub shim.ChaincodeStubInterface, owner, exchangeItemPK, signature, tkey string, args []string) error {
	var err error
	var Price, Qtt, TotalAmount, tempCoin decimal.Decimal
	var ownerData mtc.MetaWallet
	var tokenSN int
	var balance mtc.BalanceInfo
	var item mtc.ExchangeItem
	var data []byte
	var TargetTokenData mtc.Token

	if strings.Index(exchangeItemPK, "MRC040_") != 0 {
		return errors.New("6103,invalid MRC040 data address")
	}

	if data, err = stub.GetState(exchangeItemPK); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return errors.New("4207,Exchange item not found")
	}

	if err = json.Unmarshal(data, &item); err != nil {
		return errors.New("4208,Invalid exchange item data")
	}

	if item.Owner != owner {
		return errors.New("4206,Is not your MRC040 ITEM")
	}

	if item.Status == "CANCEL" {
		return errors.New("4203,Item is already canceled")
	}

	// get owner info.
	if ownerData, err = GetAddressInfo(stub, item.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, tkey,
		strings.Join([]string{owner, exchangeItemPK, tkey}, "|"),
		signature); err != nil {
		return err
	}

	// price, qtt format check.
	if Price, err = decimal.NewFromString(item.Price); err != nil {
		return errors.New("4101,Invalid Price format, Price must numeric only")
	}
	if Qtt, err = util.ParsePositive(item.RemainQtt); err != nil {
		return errors.New("4102,Invalid Qtt format, Qtt must numeric only")
	}

	if item.Side == "SELL" {
		tokenSN = item.TargetToken
		TotalAmount = Qtt
	} else if item.Side == "BUY" {
		tokenSN = item.BaseToken
		Divider, _ := decimal.NewFromString("10")

		if TargetTokenData, _, err = GetToken(stub, strconv.Itoa(item.TargetToken)); err != nil {
			return err
		}
		TotalAmount = Price.Mul(Qtt).Div(Divider.Pow(decimal.New(int64(TargetTokenData.Decimal), 0)))
	} else {
		return errors.New("4103,Invalid item data")
	}

	// collect token balance
	isBalanceFound := false
	nowTime := time.Now().Unix()
	for index, element := range ownerData.Balance {
		if element.Token != tokenSN {
			continue
		}

		if nowTime <= element.UnlockDate {
			continue
		}

		if tempCoin, err = decimal.NewFromString(element.Balance); err != nil {
			continue
		}

		ownerData.Balance[index].Balance = TotalAmount.Add(tempCoin).String()
		isBalanceFound = true
		break
	}

	// remainAmount > 0 ?
	if isBalanceFound == false {
		balance.Balance = TotalAmount.String()
		balance.Token = tokenSN
		balance.UnlockDate = 0
		ownerData.Balance = append(ownerData.Balance, balance)
	}

	if ownerData.Pending == nil {
		ownerData.Pending = make(map[int]string)
	}
	_, exists := ownerData.Pending[tokenSN]
	if exists {
		t, _ := decimal.NewFromString(ownerData.Pending[tokenSN])
		ownerData.Pending[tokenSN] = t.Sub(TotalAmount).String()
	}
	if ownerData.Pending[tokenSN] == "0" {
		delete(ownerData.Pending, tokenSN)
	}

	if err = SetAddressInfo(stub, owner, ownerData, "stodexUnRegister", args); err != nil {
		return err
	}

	item.CancelDate = time.Now().Unix()
	item.Status = "CANCEL"
	item.Type = "MRC040"
	item.JobDate = time.Now().Unix()
	item.JobType = "stodexUnRegister"
	if args != nil && len(args) > 0 {
		if data, err = json.Marshal(args); err == nil {
			item.JobArgs = string(data)
		}
	} else {
		item.JobArgs = ""
	}
	data, _ = json.Marshal(item)
	if err = stub.PutState(exchangeItemPK, data); err != nil {
		fmt.Printf("stodexUnRegister stub.PutState(key, dat) [%s] Error %s\n", exchangeItemPK, err)
		return err
	}
	return nil
}

// StodexExchange - STO token exchange using DEX.
func StodexExchange(stub shim.ChaincodeStubInterface, requester, qtt, exchangeItemPK, exchangePK, signature, tkey string, args []string) error {
	var err error
	var Price, Qtt, ownerPlusAmount, ownerMinusAmount, remainAmount, tAmount decimal.Decimal
	var ownerData, requesterData mtc.MetaWallet
	var BaseTokenData, TargetTokenData mtc.Token
	var ownerPlusToken, ownerMinusToken int
	var now int64
	var item mtc.ExchangeItem
	var exchangeResult mtc.ExchangeResult
	var data []byte
	var targs []string
	var balance mtc.BalanceInfo
	var balanceList []mtc.BalanceInfo
	var requesterSide string

	now = time.Now().Unix()
	if data, err = stub.GetState(exchangePK); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data != nil {

	}
	if strings.Index(exchangeItemPK, "MRC040_") != 0 {
		return errors.New("6103,invalid MRC040 data address")
	}
	if data, err = stub.GetState(exchangeItemPK); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return errors.New("6002,ExchangeItem not found - " + exchangeItemPK)
	}
	json.Unmarshal(data, &item)
	if item.Status == "COMPLETE" {
		return errors.New("6300,Already completed item")
	}

	if item.Status == "CANCEL" {
		return errors.New("6301,Already canceled item")
	}

	if item.Owner == requester {
		return errors.New("4100,You can't trade your own ITEM")
	}
	// get owner info.
	if requesterData, err = GetAddressInfo(stub, requester); err != nil {
		return err
	}

	if err = NonceCheck(&requesterData, tkey,
		strings.Join([]string{requester, exchangeItemPK, qtt, tkey}, "|"),
		signature); err != nil {
		return err
	}

	// check base token
	if BaseTokenData, _, err = GetToken(stub, strconv.Itoa(item.BaseToken)); err != nil {
		return err
	}

	// check exchange token
	if TargetTokenData, _, err = GetToken(stub, strconv.Itoa(item.TargetToken)); err != nil {
		return err
	}

	// token pair check.
	if BaseTokenData.TargetToken == nil {
		return errors.New("6501,Base token is not allow exchange to target token")
	}

	if _, exists := BaseTokenData.TargetToken[item.TargetToken]; exists == false {
		targs = nil
		targs = append(targs, item.Owner, exchangeItemPK, "Base token is not allow exchange to target token")
		StodexUnRegister(stub, item.Owner, exchangeItemPK, "", "", targs)
		return errors.New("6501,Base token is not allow exchange to target token")
	}

	if TargetTokenData.BaseToken != item.BaseToken {
		targs = nil
		targs = append(targs, item.Owner, exchangeItemPK, "Exchange token is not allow exchange to base token")
		StodexUnRegister(stub, item.Owner, exchangeItemPK, "", "", targs)
		return errors.New("6502,Exchange token is not allow exchange to base token")
	}

	// price, qtt format check.
	if Price, err = util.ParsePositive(item.Price); err != nil {
		return errors.New("1103,Price must be an integer string")
	}
	if Qtt, err = util.ParsePositive(qtt); err != nil {
		return errors.New("1105,Qtt must be an integer string")
	}

	if tAmount, err = decimal.NewFromString(item.RemainQtt); err != nil {
		return errors.New("1105,Qtt must be an integer string")
	}

	if Qtt.Cmp(tAmount) > 0 {
		return errors.New("1106,Qtt must be greater then zero")
	}
	item.RemainQtt = tAmount.Sub(Qtt).String()

	// pending item is sell, requester is buy.
	// pending item is buy, requster is sell.

	if item.Side == "SELL" {
		ownerPlusToken = item.BaseToken
		ownerMinusToken = item.TargetToken

		Divider, _ := decimal.NewFromString("10")
		ownerPlusAmount = Price.Mul(Qtt).Div(Divider.Pow(decimal.New(int64(TargetTokenData.Decimal), 0)))
		ownerMinusAmount = Qtt
		requesterSide = "BUY"
		if ownerPlusAmount.Cmp(ownerPlusAmount.Truncate(0)) != 0 {
			return errors.New("1204,QTT precision is too long")
		}
	} else if item.Side == "BUY" {
		ownerPlusToken = item.TargetToken
		ownerMinusToken = item.BaseToken
		ownerPlusAmount = Qtt
		Divider, _ := decimal.NewFromString("10")
		ownerMinusAmount = Price.Mul(Qtt).Div(Divider.Pow(decimal.New(int64(TargetTokenData.Decimal), 0)))
		requesterSide = "SELL"
		if ownerMinusAmount.Cmp(ownerMinusAmount.Truncate(0)) != 0 {
			return errors.New("1204,QTT precision is too long")
		}
	} else {
		return errors.New("6600,Exchange item side is invalid")
	}

	// owner plus check.
	isBalanceClean := false
	nowTime := time.Now().Unix()
	remainAmount = ownerPlusAmount
	// requester balance check.
	for index, element := range requesterData.Balance {
		if element.Token != ownerPlusToken {
			continue
		}

		if nowTime <= element.UnlockDate {
			continue
		}

		if tAmount, err = decimal.NewFromString(element.Balance); err != nil {
			continue
		}

		if tAmount.Cmp(remainAmount) < 1 {
			remainAmount = remainAmount.Sub(tAmount)
			requesterData.Balance[index].Balance = "0"
			if ownerPlusToken > 0 {
				isBalanceClean = true
			}
		} else {
			requesterData.Balance[index].Balance = tAmount.Sub(remainAmount).String()
			remainAmount = remainAmount.Sub(remainAmount)
			break
		}
	}

	// remainAmount > 0 ?
	if remainAmount.IsPositive() {
		return errors.New("5000,Not enough balance")
	}

	// balance 0 token clean up.
	if isBalanceClean {
		for _, element := range requesterData.Balance {
			if element.Token > 0 && element.Balance == "0" {
				continue
			}
			balanceList = append(balanceList, element)
		}
		requesterData.Balance = balanceList
	}

	// get owner info.
	if ownerData, err = GetAddressInfo(stub, item.Owner); err != nil {
		return err
	}

	// requester -> owner
	isPlusProcess := false
	for index, element := range ownerData.Balance {
		if element.Token != ownerPlusToken {
			continue
		}

		if nowTime <= element.UnlockDate {
			continue
		}

		if tAmount, err = decimal.NewFromString(element.Balance); err != nil {
			continue
		}
		ownerData.Balance[index].Balance = tAmount.Add(ownerPlusAmount).String()
		isPlusProcess = true
	}
	if isPlusProcess == false {
		balance.Balance = ownerPlusAmount.String()
		balance.Token = ownerPlusToken
		balance.UnlockDate = 0
		ownerData.Balance = append(ownerData.Balance, balance)
	}

	// owner pending check.
	if tAmount, err = decimal.NewFromString(ownerData.Pending[ownerMinusToken]); err != nil {
		return errors.New("5100,Owner pending balance error - " + err.Error())
	}

	tAmount = tAmount.Sub(ownerMinusAmount)
	if tAmount.IsNegative() == true {
		return errors.New("1300," + fmt.Sprintf("Owner pending balance remain error - remain %s, need %s", tAmount.Add(ownerMinusAmount).String(), ownerMinusAmount.String()))
	}

	if tAmount.IsZero() == true {
		delete(ownerData.Pending, ownerMinusToken)
	} else {
		ownerData.Pending[ownerMinusToken] = tAmount.String()
	}

	// save pending item owner data.
	targs = nil
	targs = append(targs, item.Owner)
	targs = append(targs, requester)
	targs = append(targs, item.Side)
	targs = append(targs, strconv.Itoa(item.BaseToken))
	targs = append(targs, strconv.Itoa(item.TargetToken))
	targs = append(targs, item.Price)
	targs = append(targs, qtt)
	targs = append(targs, strconv.FormatInt(now, 10))
	targs = append(targs, exchangeItemPK)
	targs = append(targs, exchangePK)
	fmt.Printf("Set AddressInfo [%s], %s", item.Owner, string(data))
	if err = SetAddressInfo(stub, item.Owner, ownerData, "stodexExchangePending", targs); err != nil {
		return err
	}

	// owner -> requester
	isMunusProcess := false
	for index, element := range requesterData.Balance {
		if element.Token != ownerMinusToken {
			continue
		}

		if nowTime <= element.UnlockDate {
			continue
		}

		if tAmount, err = decimal.NewFromString(element.Balance); err != nil {
			continue
		}
		requesterData.Balance[index].Balance = tAmount.Add(ownerMinusAmount).String()
		isMunusProcess = true
	}

	if isMunusProcess == false {
		balance.Balance = ownerMinusAmount.String()
		balance.Token = ownerMinusToken
		balance.UnlockDate = 0
		requesterData.Balance = append(requesterData.Balance, balance)
	}

	// save exchange requester data.
	targs = nil
	targs = append(targs, item.Owner)
	targs = append(targs, requester)
	targs = append(targs, requesterSide)
	targs = append(targs, strconv.Itoa(item.BaseToken))
	targs = append(targs, strconv.Itoa(item.TargetToken))
	targs = append(targs, item.Price)
	targs = append(targs, qtt)
	targs = append(targs, strconv.FormatInt(now, 64))
	targs = append(targs, exchangeItemPK)
	targs = append(targs, exchangePK)
	fmt.Printf("Set AddressInfo Requester  [%s], %s", requester, string(data))
	if err = SetAddressInfo(stub, requester, requesterData, "stodexExchangeRequest", targs); err != nil {
		return err
	}
	if item.Side == "BUY" {
		exchangeResult.BuyItemTX = exchangeItemPK
		exchangeResult.BuyOwner = item.Owner
		exchangeResult.BuyToken = item.TargetToken
		exchangeResult.SellItemTX = exchangePK
		exchangeResult.SellOwner = requester
		exchangeResult.SellToken = item.BaseToken
	} else {
		exchangeResult.BuyItemTX = exchangePK
		exchangeResult.BuyOwner = requester
		exchangeResult.BuyToken = item.BaseToken
		exchangeResult.SellItemTX = exchangeItemPK
		exchangeResult.SellOwner = item.Owner
		exchangeResult.SellToken = item.TargetToken
	}
	exchangeResult.Price = item.Price
	exchangeResult.Qtt = qtt
	exchangeResult.Regdate = now
	exchangeResult.Type = "MRC040_RESULT"
	exchangeResult.JobDate = time.Now().Unix()
	exchangeResult.JobType = "stodexExchange"
	if args != nil && len(args) > 0 {
		if data, err = json.Marshal(args); err == nil {
			exchangeResult.JobArgs = string(data)
		}
	} else {
		exchangeResult.JobArgs = ""
	}

	if data, err = stub.GetState(exchangePK); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data != nil {
		return errors.New("6000,Exchange result item is already exists")
	}

	data, err = json.Marshal(exchangeResult)
	if err = stub.PutState(exchangePK, data); err != nil {
		return err
	}

	if item.RemainQtt == "0" {
		item.CompleteDate = time.Now().Unix()
		item.Status = "COMPLETE"
	} else {
		item.Status = "TRADING"
	}
	item.Type = "MRC040"
	item.JobDate = time.Now().Unix()
	item.JobType = "stodexExchange"
	if args != nil && len(args) > 0 {
		if data, err = json.Marshal(args); err == nil {
			item.JobArgs = string(data)
		}
	} else {
		item.JobArgs = ""
	}

	data, _ = json.Marshal(item)
	if err = stub.PutState(exchangeItemPK, data); err != nil {
		fmt.Printf("stodexRequest stub.PutState(key, dat) [%s] Error %s\n", "EXCH_"+exchangeItemPK, err)
		return err
	}
	return nil
}

// Exchange exchange token and send fee
func Exchange(stub shim.ChaincodeStubInterface,
	fromAddr, fromAmount, fromToken, fromFeeAddr, fromFeeAmount, fromFeeToken, fromTKey, fromSignature string,
	toAddr, toAmount, toToken, toFeeAddr, toFeeAmount, toFeeToken, toTKey, toSignature string,
	args []string) error {
	var err error
	var mwFrom, mwTo, mwFromfee, mwTofee mtc.MetaWallet
	var PmwFrom, PmwTo, PmwFromfee, PmwTofee *mtc.MetaWallet

	// addr check
	if util.IsAddress(fromAddr) {
		return errors.New("3001,Invalid from address")
	}
	if util.IsAddress(toAddr) {
		return errors.New("3002,Invalid to address")
	}
	if fromAddr == toAddr {
		return errors.New("3201,From address and to address must be different values")
	}
	if fromToken == toToken {
		return errors.New("3202,From token and to token must be different values")
	}

	// token check
	if _, _, err = GetToken(stub, fromToken); err != nil {
		return err
	}
	if _, _, err = GetToken(stub, toToken); err != nil {
		return err
	}
	if _, _, err = GetToken(stub, fromFeeToken); err != nil {
		return err
	}
	if _, _, err = GetToken(stub, toFeeToken); err != nil {
		return err
	}

	if mwFrom, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}
	PmwFrom = &mwFrom

	if mwTo, err = GetAddressInfo(stub, toAddr); err != nil {
		return err
	}
	PmwTo = &mwTo

	switch fromFeeAddr {
	case "":
		PmwFromfee = nil
		break
	case fromAddr:
		PmwFromfee = nil
		break
	case toAddr:
		PmwFromfee = PmwTo
		break
	default:
		if util.IsAddress(fromFeeAddr) {
			return errors.New("3003,Invalid from fee address")
		}

		if mwFromfee, err = GetAddressInfo(stub, fromFeeAddr); err != nil {
			return err
		}
		PmwFromfee = &mwFromfee
	}

	switch toFeeAddr {
	case "":
		PmwTofee = nil
		break
	case toAddr:
		PmwTofee = nil
		break
	case fromAddr:
		PmwTofee = PmwFrom
	case fromFeeAddr:
		PmwTofee = PmwFromfee
	default:
		fmt.Printf("uniq toFeeAddr %s \n", toFeeAddr)
		if util.IsAddress(toFeeAddr) {
			return errors.New("3004,Invalid to fee address")
		}
		if mwTofee, err = GetAddressInfo(stub, toFeeAddr); err != nil {
			return err
		}
		PmwTofee = &mwTofee
	}

	if err = NonceCheck(&mwFrom, fromTKey,
		strings.Join([]string{fromAddr, fromAmount, fromToken, fromFeeAddr, fromFeeAmount, fromFeeToken, toAddr, toAmount, toToken, fromTKey}, "|"),
		fromSignature); err != nil {
		return err
	}

	if err = NonceCheck(&mwTo, toTKey,
		strings.Join([]string{toAddr, toAmount, toToken, toFeeAddr, toFeeAmount, toFeeToken, fromAddr, fromAmount, fromToken, toTKey}, "|"),
		toSignature); err != nil {
		return err
	}

	// from -> to
	if err = MoveToken(stub, PmwFrom, PmwTo, fromToken, fromAmount, 0); err != nil {
		if strings.Index(err.Error(), "5000,") == 0 {
			return errors.New("5001,The balance of fromuser is insufficient")
		}
		return err
	}

	// to -> from
	if err = MoveToken(stub, PmwTo, PmwFrom, toToken, toAmount, 0); err != nil {
		if strings.Index(err.Error(), "5000,") == 0 {
			return errors.New("5002,The balance of touser is insufficient")
		}
		return err
	}

	// from fee
	if PmwFromfee != nil {
		if _, err = util.ParsePositive(fromFeeAmount); err == nil {
			if _, _, err = GetToken(stub, fromFeeToken); err != nil {
				return err
			}
			if err = MoveToken(stub, PmwFrom, PmwFromfee, fromFeeToken, fromFeeAmount, 0); err != nil {
				if strings.Index(err.Error(), "5000,") == 0 {
					return errors.New("5001,The balance of fromuser is insufficient")
				}
				return err
			}
		}
	}

	// to fee
	if PmwTofee != nil {
		if _, err = util.ParsePositive(toFeeAmount); err == nil {
			if _, _, err = GetToken(stub, toFeeToken); err != nil {
				return err
			}
			if err = MoveToken(stub, PmwTo, PmwTofee, toFeeToken, toFeeAmount, 0); err != nil {
				if strings.Index(err.Error(), "5000,") == 0 {
					return errors.New("5002,The balance of touser is insufficient")
				}
				return err
			}
		}
	}

	if err = SetAddressInfo(stub, fromAddr, mwFrom, "exchange", args); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, toAddr, mwTo, "exchangePair", args); err != nil {
		return err
	}

	// from fee
	if PmwFromfee != nil {
		if err = SetAddressInfo(stub, fromFeeAddr, *PmwFromfee, "exchangeFee", args); err != nil {
			return err
		}
	}

	// to fee
	if PmwTofee != nil {
		if err = SetAddressInfo(stub, toFeeAddr, *PmwTofee, "exchangeFeePair", args); err != nil {
			return err
		}
	}
	fmt.Printf("Exchange [%s] <=> [%s], => [%s][%s], <= [%s][%s]\n", fromAddr, toAddr, fromAmount, fromToken, toAmount, toToken)

	return nil
}
