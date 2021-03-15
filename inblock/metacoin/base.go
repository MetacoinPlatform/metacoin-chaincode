package metacoin

import (
	"errors"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"encoding/pem"

	"crypto/ecdsa"
	"crypto/x509"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/shopspring/decimal"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

// NewWallet Create new wallet and address
func NewWallet(stub shim.ChaincodeStubInterface, publicKey string, addinfo string) (string, error) {
	var pub interface{}
	var pubkey *ecdsa.PublicKey
	var ok bool
	var err error
	var block *pem.Block
	var address string
	var dat []byte

	mcData := mtc.MetaWallet{Regdate: time.Now().Unix(),
		Addinfo:  addinfo,
		Password: publicKey,
		JobDate:  time.Now().Unix(),
		JobType:  "NewWallet",
		Nonce:    util.MakeRandomString(40),
		Balance:  []mtc.BalanceInfo{mtc.BalanceInfo{Balance: "0", Token: 0, UnlockDate: 0}}}

	var isSuccess = false
	for i := 1; i <= 10; i++ {
		w := util.MakeRandomString(30)
		address = fmt.Sprintf("MT%30s%08x", w, crc32.Checksum([]byte(w), crc32.MakeTable(crc32.IEEE)))

		dat, err = stub.GetState(address)
		if err != nil { // already exists.
			fmt.Printf("setAddressInfo stub.GetState(key) [%s] Error %s\n", address, err)
			return "", errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if dat != nil {
			continue
		} else {
			isSuccess = true
			break
		}
	}

	if isSuccess == false {
		return "", errors.New("3005,Address generate error")
	}

	if len(publicKey) < 40 {
		return "", errors.New("3103,Invalid Public key")
	}

	block, _ = pem.Decode([]byte(mcData.Password))
	if block == nil {
		if strings.Index(publicKey, "\n") == -1 {
			var dt = len(publicKey) - 24
			if dt < 26 {
				return "", errors.New("3103,Public key decode error " + publicKey)
			}
			fmt.Sprintf("Key DATA [%s]", publicKey)
			var buf = make([]string, 3)
			buf[0] = publicKey[0:26]
			buf[1] = publicKey[26:dt]
			buf[2] = publicKey[dt:len(publicKey)]
			publicKey = strings.Join(buf, "\n")
		}
		block, _ = pem.Decode([]byte(publicKey))
		if block == nil {
			return "", errors.New("3103,Public key decode error")
		}
	}

	pub, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", errors.New("3105,Public key parsing error")
	}

	switch pub.(type) {
	case *ecdsa.PublicKey:
		break
	default:
		return "", errors.New("3106,Public key type error")
	}

	pubkey, ok = pub.(*ecdsa.PublicKey)
	if !ok {
		return "", errors.New("3104,Public key format error")
	}

	switch pubkey.Curve.Params().BitSize {
	case 256:
		break
	case 384:
		break
	case 521:
		break
	default:
		return "", errors.New("3102,Public key curve size must be 384 or 521")
	}

	if err := SetAddressInfo(stub, address, mcData, "NewWallet", []string{address, publicKey, addinfo}); err != nil {
		return "", err
	}
	return address, nil
}

// BalanceOf - get balance of address.
func BalanceOf(stub shim.ChaincodeStubInterface, address string) (string, error) {
	var err error
	var dat mtc.MetaWallet
	var value []byte
	if dat, err = GetAddressInfo(stub, address); err != nil {
		return "[]", err
	}

	if value, err = json.Marshal(dat.Balance); err != nil {
		return "[]", err
	}
	if value == nil {
		return "[]", nil
	}
	return string(value), nil
}

// AddToken 잔액 추가
func AddToken(stub shim.ChaincodeStubInterface, wallet *mtc.MetaWallet, TokenSN string, amount string, iUnlockDate int64) error {
	var err error
	var toCoin, addAmount decimal.Decimal
	var toIDX, iTokenSN int

	nowTime := time.Now().Unix()
	if iUnlockDate < nowTime {
		iUnlockDate = 0
	}

	if addAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1101," + amount + " is not positive integer")
	}

	if _, iTokenSN, err = GetToken(stub, TokenSN); err != nil {
		return err
	}

	toIDX = -1
	toCoin = decimal.Zero
	for index, element := range wallet.Balance {
		if element.Token != iTokenSN {
			continue
		}
		if element.UnlockDate != iUnlockDate {
			continue
		}
		toCoin, _ = decimal.NewFromString(element.Balance)
		toIDX = index
		break
	}

	toCoin = toCoin.Add(addAmount).Truncate(0)
	if toIDX == -1 {
		if iUnlockDate > 0 {
			wallet.Balance = append(wallet.Balance, mtc.BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: iUnlockDate})
		} else {
			wallet.Balance = append(wallet.Balance, mtc.BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: 0})
		}
	} else {
		wallet.Balance[toIDX].Balance = toCoin.String()
		if iUnlockDate > 0 {
			wallet.Balance[toIDX].UnlockDate = iUnlockDate
		}
	}
	return nil
}

// SubtractToken 잔액 감소
func SubtractToken(stub shim.ChaincodeStubInterface, wallet *mtc.MetaWallet, TokenSN string, amount string) error {
	var err error
	var subtractAmount, fromCoin decimal.Decimal
	var fromIDX int
	var balanceTemp []mtc.BalanceInfo
	var iTokenSN int

	nowTime := time.Now().Unix()

	if subtractAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1101,Amount must be an integer string")
	}

	if _, iTokenSN, err = GetToken(stub, TokenSN); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}

	isBalanceClean := false
	for index, element := range wallet.Balance {
		if element.Token != iTokenSN {
			continue
		}

		if nowTime < element.UnlockDate {
			continue
		}

		if fromCoin, err = decimal.NewFromString(element.Balance); err != nil {
			continue
		}

		fromIDX = index
		if fromCoin.Cmp(subtractAmount) < 0 {
			subtractAmount = subtractAmount.Sub(fromCoin)
			wallet.Balance[fromIDX].Balance = "0"
			if iTokenSN > 0 {
				isBalanceClean = true
			}
			continue
		} else {
			wallet.Balance[fromIDX].Balance = fromCoin.Sub(subtractAmount).String()
			subtractAmount = subtractAmount.Sub(subtractAmount)
			break
		}
	}

	if isBalanceClean {
		for _, element := range wallet.Balance {
			if element.Token > 0 && element.Balance == "0" {
				continue
			}
			balanceTemp = append(balanceTemp, element)
		}
		wallet.Balance = balanceTemp
	}

	if subtractAmount.IsPositive() {
		return errors.New("5000,Not enough balance")
	}
	return nil
}

// MoveToken 잔액을 다른 Wallet 로 이동
func MoveToken(stub shim.ChaincodeStubInterface, fromwallet *mtc.MetaWallet, towallet *mtc.MetaWallet, TokenSN string, amount string, iUnlockDate int64) error {
	var err error
	var subtractAmount, fromCoin decimal.Decimal
	var fromIDX int
	var toCoin, addAmount decimal.Decimal
	var toIDX int
	var balanceTemp []mtc.BalanceInfo
	var iTokenSN int
	var nowTime int64

	nowTime = time.Now().Unix()
	if iUnlockDate < nowTime {
		iUnlockDate = 0
	}
	if subtractAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1101,Amount must be an integer string")
	}
	addAmount = subtractAmount

	if _, iTokenSN, err = GetToken(stub, TokenSN); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}

	isBalanceClean := false
	for index, element := range fromwallet.Balance {
		if element.Token != iTokenSN {
			continue
		}

		if nowTime < element.UnlockDate {
			continue
		}

		if fromCoin, err = decimal.NewFromString(element.Balance); err != nil {
			continue
		}

		fromIDX = index
		if fromCoin.Cmp(subtractAmount) < 0 {
			subtractAmount = subtractAmount.Sub(fromCoin)
			fromwallet.Balance[fromIDX].Balance = "0"
			if iTokenSN > 0 {
				isBalanceClean = true
			}
			continue
		} else {
			fromwallet.Balance[fromIDX].Balance = fromCoin.Sub(subtractAmount).String()
			subtractAmount = subtractAmount.Sub(subtractAmount)
			break
		}
	}

	if isBalanceClean {
		for _, element := range fromwallet.Balance {
			if element.Token > 0 && element.Balance == "0" {
				continue
			}
			balanceTemp = append(balanceTemp, element)
		}
		fromwallet.Balance = balanceTemp
	}

	if subtractAmount.IsPositive() {
		return errors.New("5000,Not enough balance")
	}

	toIDX = -1
	toCoin = decimal.Zero
	for index, element := range towallet.Balance {
		if element.Token == iTokenSN {
			if element.UnlockDate == iUnlockDate {
				toCoin, _ = decimal.NewFromString(element.Balance)
				toIDX = index
				break
			}
		}
	}

	toCoin = toCoin.Add(addAmount).Truncate(0)
	if toIDX == -1 {
		if iUnlockDate > 0 {
			towallet.Balance = append(towallet.Balance, mtc.BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: iUnlockDate})
		} else {
			towallet.Balance = append(towallet.Balance, mtc.BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: 0})
		}
	} else {
		towallet.Balance[toIDX].Balance = toCoin.String()
		if iUnlockDate > 0 {
			towallet.Balance[toIDX].UnlockDate = iUnlockDate
		}
	}
	return nil
}

// Transfer send token
func Transfer(stub shim.ChaincodeStubInterface, fromAddr, toAddr, transferAmount, token, unlockdate, signature, tkey string, args []string) error {
	var err error
	var fromData, toData mtc.MetaWallet
	var iUnlockDate int64

	if util.IsAddress(fromAddr) {
		return errors.New("3001,Invalid from address")
	}
	if util.IsAddress(toAddr) {
		return errors.New("3002,Invalid to address")
	}
	if fromAddr == toAddr {
		return errors.New("3201,From address and to address must be different values")
	}

	if _, _, err = GetToken(stub, token); err != nil {
		return err
	}
	if iUnlockDate, err = util.Strtoint64(unlockdate); err != nil {
		return errors.New("1102,Invalid unlock date")
	}
	if fromData, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}
	if toData, err = GetAddressInfo(stub, toAddr); err != nil {
		return err
	}

	if err = NonceCheck(&fromData, tkey,
		strings.Join([]string{fromAddr, toAddr, token, transferAmount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = MoveToken(stub, &fromData, &toData, token, transferAmount, iUnlockDate); err != nil {
		if strings.Index(err.Error(), "5000,") == 0 {
			return errors.New("5001,The balance of fromuser is insufficient")
		}
		return err
	}

	if err = SetAddressInfo(stub, fromAddr, fromData, "transfer", args); err != nil {
		return err
	}
	if err = SetAddressInfo(stub, toAddr, toData, "receive", args); err != nil {
		return err
	}
	fmt.Printf("Transfer [%s] => [%s]  / Amount : [%s] TokenID : [%s] UnlockDate : [%s]\n", fromAddr, toAddr, transferAmount, token, unlockdate)
	return nil
}

// MultiTransfer send token to multi address
func MultiTransfer(stub shim.ChaincodeStubInterface, fromAddr, transferlist, token, signature, tkey string, args []string) error {
	var err error
	var fromData, toData mtc.MetaWallet
	var iUnlockDate int64
	var target []mtc.MultiTransferList
	var toList map[string]int

	if util.IsAddress(fromAddr) {
		return errors.New("3001,Invalid from address")
	}
	if fromData, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}
	if err = NonceCheck(&fromData, tkey,
		strings.Join([]string{fromAddr, transferlist, token, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(transferlist), &target); err != nil {
		return errors.New("3290,Transfer list is in the wrong data - " + err.Error())
	}

	if _, _, err = GetToken(stub, token); err != nil {
		return err
	}
	if len(target) < 1 {
		return errors.New("3002, There are no multiple transmission recipients")
	}

	if len(target) > 100 {
		return errors.New("3002,There must be 100 or fewer recipients of multitransfer")
	}

	toList = make(map[string]int)
	for _, ele := range target {
		if util.IsAddress(ele.Address) {
			return errors.New("3002,Invalid to address")
		}
		if _, exists := toList[ele.Address]; exists != false {
			return errors.New("6100, [" + ele.Address + "] already exists on the transfer list.")
		}
		toList[ele.Address] = 1
		if fromAddr == ele.Address {
			return errors.New("3201,From address and to address must be different values")
		}

		if iUnlockDate, err = util.Strtoint64(ele.UnlockDate); err != nil {
			return errors.New("1102,Invalid unlock date")
		}

		if toData, err = GetAddressInfo(stub, ele.Address); err != nil {
			return err
		}

		if err = MoveToken(stub, &fromData, &toData, token, ele.Amount, iUnlockDate); err != nil {
			if strings.Index(err.Error(), "5000,") == 0 {
				return errors.New("5001,The balance of fromuser is insufficient")
			}
			return err
		}
		if len(ele.Tag) > 64 {
			ele.Tag = ele.Tag[0:64]
		}
		if len(ele.Memo) > 2048 {
			ele.Memo = ele.Memo[0:2048]
		}

		if err = SetAddressInfo(stub, ele.Address, toData, "receive", []string{fromAddr, ele.Address, ele.Amount, token, signature, ele.UnlockDate, ele.Tag, ele.Memo, tkey}); err != nil {
			return err
		}
		fmt.Printf("Transfer [%s] => [%s]  / Amount : [%s] TokenID : [%s] UnlockDate : [%s]\n", fromAddr, ele.Address, ele.Amount, token, ele.UnlockDate)

	}
	if err = SetAddressInfo(stub, fromAddr, fromData, "multi_transfer", args); err != nil {
		return err
	}
	return nil
}

// GetNonce address info.
func GetNonce(stub shim.ChaincodeStubInterface, address string) (string, error) {
	var walletData mtc.MetaWallet
	var err error

	if walletData, err = GetAddressInfo(stub, address); err != nil {
		return "", err
	}
	if walletData.Nonce != "" {
		return walletData.Nonce, nil
	}
	return strconv.FormatInt(walletData.JobDate, 10), nil
}

// NonceCheck - nonce check & sign check & generate new nonce
func NonceCheck(walletData *mtc.MetaWallet, nonce, Data, signature string) error {
	fmt.Printf("NonceCheck - Wallet nonce : %s\n", walletData.Nonce)
	fmt.Printf("NonceCheck - Your nonce   : %s\n", nonce)
	fmt.Printf("NonceCheck - Data         : %s\n", Data)
	fmt.Printf("NonceCheck - signature    : %s\n", signature)
	if walletData.Nonce != "" {
		if nonce != walletData.Nonce {
			return errors.New("1102,nonce error")
		}
	} else {
		if nonce != strconv.FormatInt(walletData.JobDate, 10) {
			return errors.New("1102,nonce error")
		}
	}

	if err := util.EcdsaSignVerify(walletData.Password,
		Data,
		signature); err != nil {
		return err
	}
	walletData.Nonce = util.MakeRandomString(40)
	return nil

}

// GetAddressInfo address info.
func GetAddressInfo(stub shim.ChaincodeStubInterface, key string) (mtc.MetaWallet, error) {
	var mcData mtc.MetaWallet

	if util.IsAddress(key) {
		return mcData, errors.New("3190,[" + key + "] is not Metacoin address")
	}
	value, err := stub.GetState(key)
	if err != nil {
		return mcData, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if value == nil {
		return mcData, errors.New("3090,Can not find the address [" + key + "]")
	}
	if err = json.Unmarshal(value, &mcData); err != nil {
		return mcData, errors.New("3290,Address [" + key + "] is in the wrong data - " + err.Error())
	}
	return mcData, nil
}

// SetAddressInfo address info
func SetAddressInfo(stub shim.ChaincodeStubInterface, key string, mcData mtc.MetaWallet, JobType string, args []string) error {
	var dat []byte
	var argdat []byte
	var err error

	mcData.JobType = JobType
	mcData.JobDate = time.Now().Unix()
	if args != nil && len(args) > 0 {
		if argdat, err = json.Marshal(args); err == nil {
			mcData.JobArgs = string(argdat)
		}
	} else {
		mcData.JobArgs = ""
	}

	if dat, err = json.Marshal(mcData); err != nil {
		fmt.Printf("setAddressInfo json.Marshal(mcData) [%s] Marshal error %s\n", key, err)
		return errors.New("3209,Invalid address data format")
	}
	if err := stub.PutState(key, dat); err != nil {
		fmt.Printf("setAddressInfo stub.PutState(key, dat) [%s] Error %s\n", key, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error() + key)
	}
	return nil
}

// SetTokenInfo : save token info
func SetTokenInfo(stub shim.ChaincodeStubInterface, key string, tk mtc.Token, JobType string, args []string) error {
	var dat []byte
	var err error

	tk.JobType = JobType
	tk.JobDate = time.Now().Unix()
	if args != nil && len(args) > 0 {
		if dat, err = json.Marshal(args); err == nil {
			tk.JobArgs = string(dat)
		}
	} else {
		tk.JobArgs = ""
	}

	if dat, err = json.Marshal(tk); err != nil {
		fmt.Printf("setTokenInfo json.Marshal(mcData) [%s] Marshal error %s\n", key, err)
		return errors.New("4204,Invalid token data format")
	}
	if err = stub.PutState("TOKEN_DATA_"+key, dat); err != nil {
		fmt.Printf("setTokenInfo stub.PutState(key, dat) [%s] Error %s\n", key, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// GetToken : get token info
func GetToken(stub shim.ChaincodeStubInterface, TokenID string) (mtc.Token, int, error) {
	var data []byte
	var tk mtc.Token
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
	return tk, TokenSN, nil
}
