package metacoin

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/shopspring/decimal"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

// TokenRegister - Token Register.
func TokenRegister(stub shim.ChaincodeStubInterface, data, signature, tkey string) (string, error) {
	var dat []byte
	var value []byte
	var err error
	var tk mtc.Token
	var currNo int
	var reserveInfo mtc.TokenReserve
	var OwnerData, reserveAddr mtc.MetaWallet
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
			reserveAddr.Balance = append(reserveAddr.Balance, mtc.BalanceInfo{Balance: reserveInfo.Value, Token: currNo, UnlockDate: reserveInfo.UnlockDate})
		}

		RemainSupply := RemainSupply.Sub(t)
		if RemainSupply.IsNegative() {
			return "", errors.New("1103,The reserve amount is greater than totalsupply.")
		}
		if err = SetAddressInfo(stub, reserveInfo.Address, reserveAddr, "token_reserve", []string{tk.Owner, reserveInfo.Address, reserveInfo.Value, strconv.Itoa(currNo)}); err != nil {
			return "", err
		}
	}

	if err = stub.PutState("TOKEN_MAX_NO", []byte(strconv.Itoa(currNo))); err != nil {
		return "", err
	}

	return strconv.Itoa(currNo), nil
}

// TokenSetBase - Set Token BASE token for STO DEX
func TokenSetBase(stub shim.ChaincodeStubInterface, TokenID, BaseTokenSN, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var sn int
	var mwOwner mtc.MetaWallet

	// token check.
	if TokenID == BaseTokenSN {
		return errors.New("4210,Must TokenID and basetoken sn is not qeual")
	}

	if tk, _, err = GetToken(stub, TokenID); err != nil {
		return err
	}

	if _, sn, err = GetToken(stub, BaseTokenSN); err != nil {
		return err
	}

	if tk.BaseToken == sn {
		return errors.New("4210,Basetoken same as the existing value")
	}

	if _, exists := tk.TargetToken[sn]; exists == true {
		return errors.New("4201,Base token are in the target token list")
	}

	if mwOwner, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&mwOwner, tkey,
		strings.Join([]string{tk.Owner, TokenID, BaseTokenSN, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if BaseTokenSN == "0" {
		// base token unset.
		tk.BaseToken = 0
	} else {
		// token check.
		tk.BaseToken = sn
	}

	return SetTokenInfo(stub, TokenID, tk, "SetBase", args)
}

// TokenAddTarget - Set Token target token for STO DEX
func TokenAddTarget(stub shim.ChaincodeStubInterface, TokenID, TargetTokenSN, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var sn int

	if tk, _, err = GetToken(stub, TokenID); err != nil {
		return err
	}

	if _, sn, err = GetToken(stub, TargetTokenSN); err != nil {
		return err
	}

	var mwOwner mtc.MetaWallet
	if mwOwner, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&mwOwner, tkey,
		strings.Join([]string{tk.Owner, TokenID, TargetTokenSN, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if tk.BaseToken == sn {
		return errors.New("1201,The TargetToken is the same as the BaseToken")
	}

	if tk.TargetToken == nil {
		tk.TargetToken = make(map[int]int64)
	} else {
		if _, exists := tk.TargetToken[sn]; exists == true {
			return errors.New("4205,Target token are in the target token list")
		}
	}
	tk.TargetToken[sn] = time.Now().Unix()

	err = SetTokenInfo(stub, TokenID, tk, "tokenAddTarget", args)
	return err
}

// TokenRemoveTarget - Set Token remove token for STO DEX
func TokenRemoveTarget(stub shim.ChaincodeStubInterface, TokenID, TargetTokenSN, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var sn int
	var mwOwner mtc.MetaWallet

	if tk, sn, err = GetToken(stub, TokenID); err != nil {
		return err
	}

	if tk.TargetToken == nil {
		return errors.New("4202,Could not find target token in the target token list")
	}
	if _, exists := tk.TargetToken[sn]; exists == false {
		return errors.New("4202,Could not find target token in the target token list")
	}

	if mwOwner, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&mwOwner, tkey,
		strings.Join([]string{tk.Owner, TokenID, TargetTokenSN, tkey}, "|"),
		signature); err != nil {
		return err
	}

	delete(tk.TargetToken, sn)
	return SetTokenInfo(stub, TokenID, tk, "tokenRemoveTarget", args)
}

// TokenAddLogger - MRC100 token logger add
func TokenAddLogger(stub shim.ChaincodeStubInterface, TokenID, logger, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var mwOwner mtc.MetaWallet

	if tk, _, err = GetToken(stub, TokenID); err != nil {
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
		if _, exists := tk.Logger[logger]; exists == true {
			return errors.New("4205,Target token are in the target token list")
		}
	}
	tk.Logger[logger] = time.Now().Unix()

	return SetTokenInfo(stub, TokenID, tk, "tokenAddLogger", args)
}

// TokenRemoveLogger - MRC100 token logger remove
func TokenRemoveLogger(stub shim.ChaincodeStubInterface, TokenID, logger, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var mwOwner mtc.MetaWallet

	if tk, _, err = GetToken(stub, TokenID); err != nil {
		return err
	}

	if tk.Logger == nil {
		return errors.New("4202,Could not find logger in the logger list")
	}
	if _, exists := tk.Logger[logger]; exists == false {
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

	return SetTokenInfo(stub, TokenID, tk, "tokenRemoveLogger", args)
}

// TokenUpdate - Token Information update.
func TokenUpdate(stub shim.ChaincodeStubInterface, TokenID, url, info, image, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var ownerData mtc.MetaWallet
	var isUpdate bool

	if tk, _, err = GetToken(stub, TokenID); err != nil {
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

	return SetTokenInfo(stub, TokenID, tk, "tokenUpdate", args)
}

// TokenBurning - Token Information update.
func TokenBurning(stub shim.ChaincodeStubInterface, TokenID, amount, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var ownerData mtc.MetaWallet
	var BurnningAmount, BurnAmount decimal.Decimal

	if BurnAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1206,The amount must be an integer")
	}

	if tk, _, err = GetToken(stub, TokenID); err != nil {
		return err
	}

	if ownerData, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, tkey,
		strings.Join([]string{TokenID, amount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = MRC010Subtract(stub, &ownerData, TokenID, amount); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, tk.Owner, ownerData, "ownerBurning", args); err != nil {
		return err
	}

	if BurnningAmount, err = util.ParseNotNegative(tk.BurnningAmount); err != nil {
		BurnningAmount = decimal.Zero
	}
	tk.BurnningAmount = BurnningAmount.Add(BurnAmount).String()

	return SetTokenInfo(stub, TokenID, tk, "tokenBurning", args)
}

// TokenIncrease - Token Information update.
func TokenIncrease(stub shim.ChaincodeStubInterface, TokenID, amount, signature, tkey string, args []string) error {
	var tk mtc.Token
	var err error
	var ownerData mtc.MetaWallet
	var TotalAmount, IncrAmount decimal.Decimal

	if IncrAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1206,amount must be a positive integer")
	}

	if tk, _, err = GetToken(stub, TokenID); err != nil {
		return err
	}

	if ownerData, err = GetAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerData, tkey,
		strings.Join([]string{TokenID, amount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = MRC010Add(stub, &ownerData, TokenID, amount, 0); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, tk.Owner, ownerData, "ownerIncrease", args); err != nil {
		return err
	}

	if TotalAmount, err = util.ParseNotNegative(tk.TotalSupply); err != nil {
		TotalAmount = decimal.Zero
	}
	tk.TotalSupply = TotalAmount.Add(IncrAmount).String()

	return SetTokenInfo(stub, TokenID, tk, "tokenIncrease", args)
}
