package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"strconv"
	"strings"
	"time"

	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"

	"crypto/ecdsa"
	"crypto/sha512"
	"crypto/x509"
	"hash/crc32"

	"math/big"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/shopspring/decimal"
)

// MetaCoin dummy struct for init
type MetaCoin struct {
}

// =========================================
// const.go
// =========================================

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
	Regdate  int64          `json:"regdate"`
	Password string         `json:"password"`
	Addinfo  string         `json:"addinfo"`
	JobType  string         `json:"job_type"`
	JobArgs  string         `json:"job_args"`
	JobDate  int64          `json:"jobdate"`
	Balance  []BalanceInfo  `json:"balance"`
	Pending  map[int]string `json:"pending"`
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
	JobType        string           `json:"job_type"`
	JobArgs        string           `json:"job_args"`
	JobDate        int64            `json:"jobdate"`
	TargetToken    map[int]int64    `json:"targettoken"`
	BaseToken      int              `json:"basetoken"`
	Type           string           `json:"type"`
	Logger         map[string]int64 `json:"logger"`
}

// TokenReserve token ico reserve
type TokenReserve struct {
	Address    string `json:"address"`
	Value      string `json:"value"`
	UnlockDate int64  `json:"unlockdate"`
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

// ecdsaSignature : ecdsa signature
type ecdsaSignature struct {
	R, S *big.Int
}

// =========================================
// utils.go
// =========================================
// mtc error class.

// ecdsaSignVerify : ecdsa signature verify
func ecdsaSignVerify(PublicKeyPem, Data, Sign string) (bool, error) {
	var data = []byte(Data)
	var signature []byte
	var pub interface{}
	var pubkey *ecdsa.PublicKey
	var ok bool
	var err error
	var esig ecdsaSignature
	var block *pem.Block
	var hasher hash.Hash

	signature, _ = base64.StdEncoding.DecodeString(Sign)

	block, _ = pem.Decode([]byte(PublicKeyPem))
	if block == nil {
		if strings.Index(PublicKeyPem, "\n") == -1 {
			var dt = len(PublicKeyPem) - 24
			var buf = make([]string, 3)
			buf[0] = PublicKeyPem[0:26]
			buf[1] = PublicKeyPem[26:dt]
			buf[2] = PublicKeyPem[dt:len(PublicKeyPem)]
			PublicKeyPem = strings.Join(buf, "\n")
		}
		block, _ = pem.Decode([]byte(PublicKeyPem))
		if block == nil {
			return false, errors.New("2310,Public key decode error")
		}
	}

	pub, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}

	switch pub.(type) {
	case *ecdsa.PublicKey:
		break
	default:
		return false, errors.New("2210,PublicKey type error")
	}

	pubkey, ok = pub.(*ecdsa.PublicKey)
	if !ok {
		return false, errors.New("2110,PublicKey format error")
	}

	switch pubkey.Curve.Params().BitSize {
	case 384:
		hasher = sha512.New384()
		break
	case 521:
		hasher = sha512.New()
		break
	default:
		return false, errors.New("2220,Invalid public key curve")
	}

	if _, err = asn1.Unmarshal(signature, &esig); err != nil {
		return false, errors.New("2230,Signature format error - " + err.Error())
	}

	hasher.Write([]byte(data))
	if ecdsa.Verify(pubkey, hasher.Sum(nil), esig.R, esig.S) {
		return true, nil
	}
	return false, errors.New("2010,Invalid signature")
}

// IsInvalidID - address check
func IsInvalidID(address string) bool {
	if address == CoinName {
		return true
	}
	if len(strings.TrimSpace(address)) != 40 {
		return true
	}
	if address[:2] != "MT" {
		return true
	}

	calcCRC := fmt.Sprintf("%08x", crc32.Checksum([]byte(address[2:32]), crc32.MakeTable(crc32.IEEE)))
	if address[32:] != calcCRC {
		return true
	}
	return false
}

// string to int.
func strtoint(data string) (int, error) {
	var err error
	var i int
	if i, err = strconv.Atoi(data); err != nil {
		return 0, errors.New("9900,Data [" + data + "] is not integer")
	}
	return i, nil
}

// string to int.
func strtoint64(data string) (int64, error) {
	var err error
	var i64 int64
	if i64, err = strconv.ParseInt(data, 10, 64); err != nil {
		return 0, errors.New("9900,Data [" + data + "] is not integer")
	}
	return i64, nil
}

// remove slice element
func remove(s []string, i int) []string {
	if i < 0 {
		return s
	}
	if i >= len(s) {
		i = len(s) - 1
	}

	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// get address info.
func (mtc *MetaCoin) getAddressInfo(stub shim.ChaincodeStubInterface, key string) (MetaWallet, error) {
	var mcData MetaWallet
	if IsInvalidID(key) {
		return mcData, errors.New("3190,Address [" + key + "] is in the wrong format")
	}
	value, err := stub.GetState(key)
	if err != nil {
		return mcData, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if value == nil {
		return mcData, errors.New("3090,Can not find the address [" + key + "]")
	}
	if err = json.Unmarshal(value, &mcData); err != nil {
		return mcData, errors.New("3290,Address [" + key + "] is in the wrong data")
	}
	return mcData, nil
}

// set address info
func (mtc *MetaCoin) setAddressInfo(stub shim.ChaincodeStubInterface, key string, mcData MetaWallet, JobType string, args []string) error {
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
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// setTokenInfo : save token info
func (mtc *MetaCoin) setTokenInfo(stub shim.ChaincodeStubInterface, key string, tk Token, JobType string, args []string) error {
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

// getToken : get token info
func (mtc *MetaCoin) getToken(stub shim.ChaincodeStubInterface, TokenID string) (Token, int, error) {
	var data []byte
	var tk Token
	var err error
	var TokenSN int

	if len(TokenID) == 0 {
		return tk, 0, errors.New("4002,Token id missing")
	}
	if TokenSN, err = strtoint(TokenID); err != nil {
		return tk, 0, errors.New("4104,Invalid toekn SN")
	}
	if data, err = stub.GetState("TOKEN_DATA_" + TokenID); err != nil {
		return tk, TokenSN, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return tk, TokenSN, errors.New("4001,Token " + TokenID + " not exists")
	}
	if err = json.Unmarshal(data, &tk); err != nil {
		return tk, TokenSN, errors.New("4204,Invalid token data format")
	}
	return tk, TokenSN, nil
}

// setMRC011 : save MRC011
func (mtc *MetaCoin) setMRC011(stub shim.ChaincodeStubInterface, MRC011ID string, tk MRC011, JobType string, args []string) error {
	var dat []byte
	var err error
	if len(MRC011ID) != 40 {
		return errors.New("4202,MRC011 id length is must be 40")
	}

	if strings.Index(MRC011ID, "MRC011_") != 0 {
		return errors.New("4204,Invalid ID")
	}

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
		fmt.Printf("setTokenInfo json.Marshal(tk) [%s] Marshal error %s\n", MRC011ID, err)
		return errors.New("4204,Invalid MRC011 data format")
	}
	if err = stub.PutState(MRC011ID, dat); err != nil {
		fmt.Printf("setTokenInfo stub.PutState(key, dat) [%s] Error %s\n", MRC011ID, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// getMRC011 : get MRC011
func (mtc *MetaCoin) getMRC011(stub shim.ChaincodeStubInterface, MRC011ID string) (MRC011, error) {
	var data []byte
	var tk MRC011
	var err error

	if len(MRC011ID) != 40 {
		return tk, errors.New("4202,MRC011 id length is must be 40")
	}
	if strings.Index(MRC011ID, "MRC011_") != 0 {
		return tk, errors.New("4204,Invalid ID")
	}

	if data, err = stub.GetState(MRC011ID); err != nil {
		return tk, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return tk, errors.New("4201,MRC011 " + MRC011ID + " not exists")
	}
	if err = json.Unmarshal(data, &tk); err != nil {
		return tk, errors.New("4204,Invalid MRC011 data format")
	}
	return tk, nil
}

// setMRC012 : save MRC012
func (mtc *MetaCoin) setMRC012(stub shim.ChaincodeStubInterface, MRC012ID string, tk MRC012, JobType string, args []string) error {
	var dat []byte
	var err error
	if len(MRC012ID) != 40 {
		return errors.New("4202,MRC012 id length is must be 40")
	}
	if strings.Index(MRC012ID, "MRC012_") != 0 {
		return errors.New("4204,Invalid ID")
	}

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
		fmt.Printf("setMRC012 json.Marshal(tk) [%s] Marshal error %s\n", MRC012ID, err)
		return errors.New("4204,Invalid MRC012 data format")
	}
	if err = stub.PutState(MRC012ID, dat); err != nil {
		fmt.Printf("setMRC012 stub.PutState(key, dat) [%s] Error %s\n", MRC012ID, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// setMRC030 : save setMRC030
func (mtc *MetaCoin) setMRC030(stub shim.ChaincodeStubInterface, MRC030ID string, tk MRC030, JobType string, args []string) error {
	var dat []byte
	var err error
	if len(MRC030ID) != 40 {
		return errors.New("4202,MRC030 id length is must be 40")
	}
	if strings.Index(MRC030ID, "MRC030_") != 0 {
		return errors.New("4204,Invalid ID")
	}

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
		fmt.Printf("setMRC030 json.Marshal(tk) [%s] Marshal error %s\n", MRC030ID, err)
		return errors.New("4204,Invalid MRC030 data format")
	}
	if err = stub.PutState(MRC030ID, dat); err != nil {
		fmt.Printf("setMRC030 stub.PutState(key, dat) [%s] Error %s\n", MRC030ID, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// getMRC030 : get MRC030
func (mtc *MetaCoin) getMRC030(stub shim.ChaincodeStubInterface, MRC030ID string) (MRC030, error) {
	var data []byte
	var tk MRC030
	var err error

	if len(MRC030ID) != 40 {
		return tk, errors.New("4202,MRC030 id length is must be 40")
	}
	if strings.Index(MRC030ID, "MRC030_") != 0 {
		return tk, errors.New("4204,Invalid ID")
	}

	if data, err = stub.GetState(MRC030ID); err != nil {
		return tk, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return tk, errors.New("4201,MRC030 " + MRC030ID + " not exists")
	}
	if err = json.Unmarshal(data, &tk); err != nil {
		fmt.Printf("setMRC030 json.Unmarshal(key, dat) [%s] Error %s\n", MRC030ID, err)
		return tk, errors.New("4204,Invalid MRC030 data format")
	}
	return tk, nil
}

// getMRC031 : get MRC031
func (mtc *MetaCoin) getMRC031(stub shim.ChaincodeStubInterface, MRC031ID string) (MRC031, error) {
	var data []byte
	var tk MRC031
	var err error

	if len(MRC031ID) != 81 {
		return tk, errors.New("4202,MRC031 id length is must be 81")
	}
	if strings.Index(MRC031ID, "MRC030_") != 0 {
		return tk, errors.New("4204,Invalid ID")
	}

	if data, err = stub.GetState(MRC031ID); err != nil {
		return tk, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return tk, errors.New("4201,MRC031 " + MRC031ID + " not exists")
	}
	if err = json.Unmarshal(data, &tk); err != nil {
		return tk, errors.New("4204,Invalid MRC031 data format")
	}
	return tk, nil
}

// =========================================
// base.go
// =========================================

// NewWallet Create new wallet and address.
func (mtc *MetaCoin) NewWallet(stub shim.ChaincodeStubInterface, walletID string, publicKey string, addinfo string, args []string) error {
	var pub interface{}
	var pubkey *ecdsa.PublicKey
	var ok bool
	var err error
	var block *pem.Block

	mcData := MetaWallet{Regdate: time.Now().Unix(),
		Addinfo:  addinfo,
		Password: publicKey,
		JobDate:  time.Now().Unix(),
		JobType:  "NewWallet",
		Balance:  []BalanceInfo{BalanceInfo{Balance: "0", Token: 0, UnlockDate: 0}}}

	value, err := stub.GetState(walletID)
	if err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}

	if value != nil {
		return errors.New("3005," + walletID + " is already exists")
	}

	block, _ = pem.Decode([]byte(publicKey))
	if block == nil {
		if strings.Index(publicKey, "\n") == -1 {
			var dt = len(publicKey) - 24
			var buf = make([]string, 3)
			buf[0] = publicKey[0:26]
			buf[1] = publicKey[26:dt]
			buf[2] = publicKey[dt:len(publicKey)]
			publicKey = strings.Join(buf, "\n")
		}
		block, _ = pem.Decode([]byte(publicKey))
		if block == nil {
			return errors.New("3103,Public key decode error")
		}
	}

	pub, err = x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return errors.New("3105,Public key parsing error")
	}

	switch pub.(type) {
	case *ecdsa.PublicKey:
		break
	default:
		return errors.New("3106,Public key type error")
	}

	pubkey, ok = pub.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("3104,Public key format error")
	}

	switch pubkey.Curve.Params().BitSize {
	case 384:
		break
	case 521:
		break
	default:
		return errors.New("3102,Invalid public key curve size")
	}

	if err := mtc.setAddressInfo(stub, walletID, mcData, "NewWallet", args); err != nil {
		return err
	}
	return nil
}

// BalanceOf - get balance of address.
func (mtc *MetaCoin) BalanceOf(stub shim.ChaincodeStubInterface, address string) (string, error) {
	var err error
	var dat MetaWallet
	var value []byte
	if dat, err = mtc.getAddressInfo(stub, address); err != nil {
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

// 잔액 추가
func (mtc *MetaCoin) addToken(stub shim.ChaincodeStubInterface, wallet *MetaWallet, TokenSN string, amount string, iUnlockDate int64) error {
	var err error
	var toCoin, addAmount decimal.Decimal
	var toIDX, iTokenSN int

	nowTime := time.Now().Unix()
	if iUnlockDate < nowTime {
		iUnlockDate = 0
	}

	if addAmount, err = decimal.NewFromString(amount); err != nil {
		return errors.New("1101,Amount must be an integer string")
	}
	if !addAmount.IsPositive() {
		return errors.New("1202,Amount must be greater then zero")
	}

	if _, iTokenSN, err = mtc.getToken(stub, TokenSN); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}

	toIDX = -1
	toCoin, _ = decimal.NewFromString("0")
	for index, element := range wallet.Balance {
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
			wallet.Balance = append(wallet.Balance, BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: iUnlockDate})
		} else {
			wallet.Balance = append(wallet.Balance, BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: 0})
		}
	} else {
		wallet.Balance[toIDX].Balance = toCoin.String()
		if iUnlockDate > 0 {
			wallet.Balance[toIDX].UnlockDate = iUnlockDate
		}
	}
	return nil
}

// 잔액 감소
func (mtc *MetaCoin) subtractToken(stub shim.ChaincodeStubInterface, wallet *MetaWallet, TokenSN string, amount string) error {
	var err error
	var subtractAmount, fromCoin decimal.Decimal
	var fromIDX int
	var balanceTemp []BalanceInfo
	var iTokenSN int

	nowTime := time.Now().Unix()

	if subtractAmount, err = decimal.NewFromString(amount); err != nil {
		return errors.New("1101,Amount must be an integer string")
	}
	if !subtractAmount.IsPositive() {
		return errors.New("1202,Amount must be greater then zero")
	}

	if _, iTokenSN, err = mtc.getToken(stub, TokenSN); err != nil {
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

// 잔액을 다른 Wallet 로 이동
func (mtc *MetaCoin) moveToken(stub shim.ChaincodeStubInterface, fromwallet *MetaWallet, towallet *MetaWallet, TokenSN string, amount string, iUnlockDate int64) error {
	var err error
	var subtractAmount, fromCoin decimal.Decimal
	var fromIDX int
	var toCoin, addAmount decimal.Decimal
	var toIDX int
	var balanceTemp []BalanceInfo
	var iTokenSN int
	var nowTime int64

	nowTime = time.Now().Unix()
	if iUnlockDate < nowTime {
		iUnlockDate = 0
	}
	if subtractAmount, err = decimal.NewFromString(amount); err != nil {
		return errors.New("1101,Amount must be an integer string")
	}
	if !subtractAmount.IsPositive() {
		return errors.New("1202,Amount must be greater then zero")
	}
	addAmount = subtractAmount

	if _, iTokenSN, err = mtc.getToken(stub, TokenSN); err != nil {
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
	toCoin, _ = decimal.NewFromString("0")
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
			towallet.Balance = append(towallet.Balance, BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: iUnlockDate})
		} else {
			towallet.Balance = append(towallet.Balance, BalanceInfo{Balance: toCoin.String(), Token: iTokenSN, UnlockDate: 0})
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
func (mtc *MetaCoin) Transfer(stub shim.ChaincodeStubInterface, fromAddr, toAddr, transferAmount, token, unlockdate, signature, tkey string, args []string) error {
	var err error
	var fromData, toData MetaWallet
	var iUnlockDate int64

	if IsInvalidID(fromAddr) {
		return errors.New("3001,Invalid from address")
	}
	if IsInvalidID(toAddr) {
		return errors.New("3002,Invalid to address")
	}
	if fromAddr == toAddr {
		return errors.New("3201,From address and to address must be different values")
	}

	if _, _, err = mtc.getToken(stub, token); err != nil {
		return err
	}
	if iUnlockDate, err = strtoint64(unlockdate); err != nil {
		return errors.New("1102,Invalid unlock date")
	}
	if fromData, err = mtc.getAddressInfo(stub, fromAddr); err != nil {
		return err
	}
	if toData, err = mtc.getAddressInfo(stub, toAddr); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(fromData.Password,
		strings.Join([]string{fromAddr, toAddr, token, transferAmount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = mtc.moveToken(stub, &fromData, &toData, token, transferAmount, iUnlockDate); err != nil {
		if strings.Index(err.Error(), "5000,") == 0 {
			return errors.New("5001,The balance of fromuser is insufficient")
		}
		return err
	}
	if err = mtc.setAddressInfo(stub, fromAddr, fromData, "transfer", args); err != nil {
		return err
	}
	if err = mtc.setAddressInfo(stub, toAddr, toData, "receive", args); err != nil {
		return err
	}
	fmt.Printf("Transfer [%s] => [%s]  / Amount : [%s] TokenID : [%s] UnlockDate : [%s]\n", fromAddr, toAddr, transferAmount, token, unlockdate)
	return nil
}

// Exchange exchange token and send fee
func (mtc *MetaCoin) Exchange(stub shim.ChaincodeStubInterface,
	fromAddr, fromAmount, fromToken, fromFeeAddr, fromFeeAmount, fromFeeToken, fromTKey, fromSignature string,
	toAddr, toAmount, toToken, toFeeAddr, toFeeAmount, toFeeToken, toTKey, toSignature string,
	args []string) error {
	var err error
	var mwFrom, mwTo, mwFromfee, mwTofee MetaWallet
	var PmwFrom, PmwTo, PmwFromfee, PmwTofee *MetaWallet
	var decimalCheck decimal.Decimal

	// addr check
	if IsInvalidID(fromAddr) {
		return errors.New("3001,Invalid from address")
	}
	if IsInvalidID(toAddr) {
		return errors.New("3002,Invalid to address")
	}
	if fromAddr == toAddr {
		return errors.New("3201,From address and to address must be different values")
	}
	if fromToken == toToken {
		return errors.New("3202,From token and to token must be different values")
	}

	// token check
	if _, _, err = mtc.getToken(stub, fromToken); err != nil {
		return err
	}
	if _, _, err = mtc.getToken(stub, toToken); err != nil {
		return err
	}
	if _, _, err = mtc.getToken(stub, fromFeeToken); err != nil {
		return err
	}
	if _, _, err = mtc.getToken(stub, toFeeToken); err != nil {
		return err
	}

	if mwFrom, err = mtc.getAddressInfo(stub, fromAddr); err != nil {
		return err
	}
	PmwFrom = &mwFrom

	if mwTo, err = mtc.getAddressInfo(stub, toAddr); err != nil {
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
		if IsInvalidID(fromFeeAddr) {
			return errors.New("3003,Invalid from fee address")
		}

		if mwFromfee, err = mtc.getAddressInfo(stub, fromFeeAddr); err != nil {
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
		if IsInvalidID(toFeeAddr) {
			return errors.New("3004,Invalid to fee address")
		}
		if mwTofee, err = mtc.getAddressInfo(stub, toFeeAddr); err != nil {
			return err
		}
		PmwTofee = &mwTofee
	}

	if _, err = ecdsaSignVerify(mwFrom.Password,
		strings.Join([]string{fromAddr, fromAmount, fromToken, fromFeeAddr, fromFeeAmount, fromFeeToken, toAddr, toAmount, toToken, fromTKey}, "|"),
		fromSignature); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(mwTo.Password,
		strings.Join([]string{toAddr, toAmount, toToken, toFeeAddr, toFeeAmount, toFeeToken, fromAddr, fromAmount, fromToken, toTKey}, "|"),
		toSignature); err != nil {
		return err
	}

	// from -> to
	if err = mtc.moveToken(stub, PmwFrom, PmwTo, fromToken, fromAmount, 0); err != nil {
		if strings.Index(err.Error(), "5000,") == 0 {
			return errors.New("5001,The balance of fromuser is insufficient")
		}
		return err
	}

	// to -> from
	if err = mtc.moveToken(stub, PmwTo, PmwFrom, toToken, toAmount, 0); err != nil {
		if strings.Index(err.Error(), "5000,") == 0 {
			return errors.New("5002,The balance of touser is insufficient")
		}
		return err
	}

	// from fee
	if PmwFromfee != nil {
		if decimalCheck, err = decimal.NewFromString(fromFeeAmount); err != nil {
			decimalCheck = decimal.Zero
		}
		if decimalCheck.IsPositive() {
			if _, _, err = mtc.getToken(stub, fromFeeToken); err != nil {
				return err
			}
			if err = mtc.moveToken(stub, PmwFrom, PmwFromfee, fromFeeToken, fromFeeAmount, 0); err != nil {
				if strings.Index(err.Error(), "5000,") == 0 {
					return errors.New("5001,The balance of fromuser is insufficient")
				}
				return err
			}
		}
	}

	// to fee
	if PmwTofee != nil {
		if decimalCheck, err = decimal.NewFromString(toFeeAmount); err != nil {
			decimalCheck = decimal.Zero
		}
		if decimalCheck.IsPositive() {
			if _, _, err = mtc.getToken(stub, toFeeToken); err != nil {
				return err
			}
			if err = mtc.moveToken(stub, PmwTo, PmwTofee, toFeeToken, toFeeAmount, 0); err != nil {
				if strings.Index(err.Error(), "5000,") == 0 {
					return errors.New("5002,The balance of touser is insufficient")
				}
				return err
			}
		}
	}

	if err = mtc.setAddressInfo(stub, fromAddr, mwFrom, "exchange", args); err != nil {
		return err
	}
	if err = mtc.setAddressInfo(stub, toAddr, mwTo, "exchangePair", args); err != nil {
		return err
	}

	// from fee
	if PmwFromfee != nil {
		if err = mtc.setAddressInfo(stub, fromFeeAddr, *PmwFromfee, "exchangeFee", args); err != nil {
			return err
		}
	}

	// to fee
	if PmwTofee != nil {
		if err = mtc.setAddressInfo(stub, toFeeAddr, *PmwTofee, "exchangeFeePair", args); err != nil {
			return err
		}
	}
	fmt.Printf("Exchange [%s] <=> [%s], => [%s][%s], <= [%s][%s]\n", fromAddr, toAddr, fromAmount, fromToken, toAmount, toToken)

	return nil
}

// =========================================
// mrc020.go
// =========================================

// Mrc020set - MRC-020 Protocol set
func (mtc *MetaCoin) Mrc020set(stub shim.ChaincodeStubInterface, owner, algorithm, data, publickey, opendate, referencekey, sign, tkey string) (string, error) {
	var dat []byte
	var mrc020Key string
	var err error
	var mrc020 MRC020
	var currNo64 int64

	if currNo64, err = strtoint64(opendate); err != nil {
		return "", errors.New("1102,Invalid opendate")
	}
	mrc020.OpenDate = currNo64
	mrc020.Owner = owner
	mrc020.Algorithm = algorithm
	mrc020.Data = data
	mrc020.PublicKey = publickey
	mrc020.ReferenceKey = referencekey
	mrc020.IsOpen = 0
	mrc020.CreateDate = time.Now().Unix()

	mrc020Key = "MRC020_" + owner + "_" + referencekey
	dat, err = stub.GetState(mrc020Key)
	if err == nil && dat != nil {
		return "", errors.New("6003,MRC020 already exists")
	}

	if dat, err = json.Marshal(mrc020); err != nil {
		return "", errors.New("6204,Invalid MTC020 Data format")
	}

	var mwOwner MetaWallet
	if mwOwner, err = mtc.getAddressInfo(stub, owner); err != nil {
		return "", err
	}
	if _, err = ecdsaSignVerify(mwOwner.Password,
		strings.Join([]string{owner, data, opendate, referencekey, tkey}, "|"),
		sign); err != nil {
		return "", err
	}

	if err = stub.PutState(mrc020Key, dat); err != nil {
		return "", err
	}
	return mrc020Key, nil
}

// Mrc020get - MRC-020 Protocol Add
func (mtc *MetaCoin) Mrc020get(stub shim.ChaincodeStubInterface, mrc020Key string) (string, error) {
	var dat []byte
	var err error
	var mrc020 MRC020

	if strings.Index(mrc020Key, "MRC020_") != 0 {
		return "", errors.New("6102,invalid MRC020 data address")
	}

	dat, err = stub.GetState(mrc020Key)
	if err != nil {
		return "", errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return "", errors.New("6004,MRC020 data not exist")
	}
	if err = json.Unmarshal(dat, &mrc020); err != nil {
		return "", errors.New("6205,MRC020 [" + mrc020Key + "] is in the wrong data")
	}

	if mrc020.OpenDate > time.Now().Unix() {
		mrc020.PublicKey = ""
	} else if mrc020.IsOpen == 0 {
		mrc020.IsOpen = 1
		dat, _ = json.Marshal(mrc020)
		_ = stub.PutState(mrc020Key, dat)
	}
	return string(dat), nil
}

// =========================================
// stodex.go
// =========================================

// Mrc040get - get MRC040 Token
func (mtc *MetaCoin) Mrc040get(stub shim.ChaincodeStubInterface, mrc040Key string) (string, error) {
	var dat []byte
	var err error
	var mrc040 ExchangeItem

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
func (mtc *MetaCoin) StodexRegister(stub shim.ChaincodeStubInterface,
	owner, side, BaseToken, TargetToken, price, qtt, exchangeItemPK, sign, tkey string,
	args []string) error {
	var err error
	var Price, Qtt, TotalAmount, tempCoin decimal.Decimal
	var ownerData MetaWallet
	var exists bool
	var BaseTokenSN, TargetTokenSN int
	var BaseTokenData, TargetTokenData Token
	var tokenSN int
	var CurrentPending string
	var item ExchangeItem
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
	if ownerData, err = mtc.getAddressInfo(stub, owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join([]string{owner, BaseToken, TargetToken, price, qtt, tkey}, "|"),
		sign); err != nil {
		return err
	}

	if BaseTokenData, BaseTokenSN, err = mtc.getToken(stub, BaseToken); err != nil {
		return err
	}

	if TargetTokenData, TargetTokenSN, err = mtc.getToken(stub, TargetToken); err != nil {
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
	if Price, err = decimal.NewFromString(price); err != nil {
		return errors.New("1103,Price must be an integer string")
	}
	if !Price.IsPositive() {
		return errors.New("1104,Price must be greater then zero")
	}

	if Qtt, err = decimal.NewFromString(qtt); err != nil {
		return errors.New("1105,Qtt must be an integer string")
	}
	if !Qtt.IsPositive() {
		return errors.New("1106,Qtt must be greater then zero")
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
	if err = mtc.subtractToken(stub, &ownerData, strconv.Itoa(tokenSN), TotalAmount.String()); err != nil {
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

	if err = mtc.setAddressInfo(stub, owner, ownerData, "stodexRegister", args); err != nil {
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
	if _, err = stub.GetState(exchangeItemPK); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if err = stub.PutState(exchangeItemPK, buf); err != nil {
		fmt.Printf("stodexRegister stub.PutState(key, dat) [%s] Error %s\n", exchangeItemPK, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// StodexUnRegister - STO token unregister for DEX.
func (mtc *MetaCoin) StodexUnRegister(stub shim.ChaincodeStubInterface, owner, exchangeItemPK, sign, tkey string, args []string) error {
	var err error
	var Price, Qtt, TotalAmount, tempCoin decimal.Decimal
	var ownerData MetaWallet
	var tokenSN int
	var balance BalanceInfo
	var item ExchangeItem
	var data []byte
	var TargetTokenData Token

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
	if ownerData, err = mtc.getAddressInfo(stub, item.Owner); err != nil {
		return err
	}
	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join([]string{owner, exchangeItemPK, tkey}, "|"),
		sign); err != nil {
		return err
	}

	// price, qtt format check.
	if Price, err = decimal.NewFromString(item.Price); err != nil {
		return errors.New("4101,Invalid Price format, Price must numeric only")
	}
	if Qtt, err = decimal.NewFromString(item.RemainQtt); err != nil {
		return errors.New("4102,Invalid Qtt format, Qtt must numeric only")
	}

	if Qtt.IsPositive() == false {
		return errors.New("4102,Invalid Qtt format, Qtt must numeric only")
	}

	if item.Side == "SELL" {
		tokenSN = item.TargetToken
		TotalAmount = Qtt
	} else if item.Side == "BUY" {
		tokenSN = item.BaseToken
		Divider, _ := decimal.NewFromString("10")

		if TargetTokenData, _, err = mtc.getToken(stub, strconv.Itoa(item.TargetToken)); err != nil {
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

	if err = mtc.setAddressInfo(stub, owner, ownerData, "stodexUnRegister", args); err != nil {
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
func (mtc *MetaCoin) StodexExchange(stub shim.ChaincodeStubInterface, requester, qtt, exchangeItemPK, exchangePK, sign, tkey string, args []string) error {
	var err error
	var Price, Qtt, ownerPlusAmount, ownerMinusAmount, remainAmount, tAmount decimal.Decimal
	var ownerData, requesterData MetaWallet
	var BaseTokenData, TargetTokenData Token
	var ownerPlusToken, ownerMinusToken int
	var now int64
	var item ExchangeItem
	var exchangeResult ExchangeResult
	var data []byte
	var targs []string
	var balance BalanceInfo
	var balanceList []BalanceInfo
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
	if requesterData, err = mtc.getAddressInfo(stub, requester); err != nil {
		return err
	}
	if _, err = ecdsaSignVerify(requesterData.Password,
		strings.Join([]string{requester, exchangeItemPK, qtt, tkey}, "|"),
		sign); err != nil {
		return err
	}

	// check base token
	if BaseTokenData, _, err = mtc.getToken(stub, strconv.Itoa(item.BaseToken)); err != nil {
		return err
	}

	// check exchange token
	if TargetTokenData, _, err = mtc.getToken(stub, strconv.Itoa(item.TargetToken)); err != nil {
		return err
	}

	// token pair check.
	if BaseTokenData.TargetToken == nil {
		return errors.New("6501,Base token is not allow exchange to target token")
	}

	if _, exists := BaseTokenData.TargetToken[item.TargetToken]; exists == false {
		targs = nil
		targs = append(targs, item.Owner, exchangeItemPK, "Base token is not allow exchange to target token")
		mtc.StodexUnRegister(stub, item.Owner, exchangeItemPK, "", "", targs)
		return errors.New("6501,Base token is not allow exchange to target token")
	}

	if TargetTokenData.BaseToken != item.BaseToken {
		targs = nil
		targs = append(targs, item.Owner, exchangeItemPK, "Exchange token is not allow exchange to base token")
		mtc.StodexUnRegister(stub, item.Owner, exchangeItemPK, "", "", targs)
		return errors.New("6502,Exchange token is not allow exchange to base token")
	}

	// price, qtt format check.
	if Price, err = decimal.NewFromString(item.Price); err != nil {
		return errors.New("1103,Price must be an integer string")
	}
	if Qtt, err = decimal.NewFromString(qtt); err != nil {
		return errors.New("1105,Qtt must be an integer string")
	}
	if !Qtt.IsPositive() {
		return errors.New("1106,Qtt must be greater then zero")
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
	if ownerData, err = mtc.getAddressInfo(stub, item.Owner); err != nil {
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
	if err = mtc.setAddressInfo(stub, item.Owner, ownerData, "stodexExchangePending", targs); err != nil {
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
	if err = mtc.setAddressInfo(stub, requester, requesterData, "stodexExchangeRequest", targs); err != nil {
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

// =========================================
// token.go
// =========================================

// TokenRegister - Token Register.
func (mtc *MetaCoin) TokenRegister(stub shim.ChaincodeStubInterface, data, signature, tkey string) (string, error) {
	var dat []byte
	var value []byte
	var err error
	var tk Token
	var currNo int
	var reserveInfo TokenReserve
	var OwnerData, reserveAddr MetaWallet

	// unmarshal data.
	if err = json.Unmarshal([]byte(data), &tk); err != nil {
		return "", errors.New("4105,Invalid token Data format")
	}

	// data check
	if len(strings.TrimSpace(tk.Symbol)) < 1 {
		return "", errors.New("1002,Symbol is empty")
	}

	if len(strings.TrimSpace(tk.Name)) < 1 {
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

	if value == nil {
		currNo = 0
	} else {
		currNo64, _ := strconv.ParseInt(string(value), 10, 32)
		currNo = int(currNo64)
		currNo = currNo + 1
	}

	if OwnerData, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return "", err
	}

	if _, err = ecdsaSignVerify(OwnerData.Password,
		strings.Join([]string{tk.Owner, tk.Name, tkey}, "|"),
		signature); err != nil {
		return "", err
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

	for _, reserveInfo = range tk.Reserve {
		if reserveAddr, err = mtc.getAddressInfo(stub, reserveInfo.Address); err != nil {
			return "", errors.New("3000,Token reserve address " + reserveInfo.Address + " not found")
		}
		if currNo == 0 {
			reserveAddr.Balance[0].Balance = reserveInfo.Value
		} else {
			reserveAddr.Balance = append(reserveAddr.Balance, BalanceInfo{Balance: reserveInfo.Value, Token: currNo, UnlockDate: reserveInfo.UnlockDate})
		}

		if err = mtc.setAddressInfo(stub, reserveInfo.Address, reserveAddr, "token_reserve", []string{tk.Owner, reserveInfo.Address, reserveInfo.Value, strconv.Itoa(currNo)}); err != nil {
			return "", err
		}
	}

	if err = stub.PutState("TOKEN_MAX_NO", []byte(strconv.Itoa(currNo))); err != nil {
		return "", err
	}

	return strconv.Itoa(currNo), nil
}

// TokenSetBase - Set Token BASE token for STO DEX
func (mtc *MetaCoin) TokenSetBase(stub shim.ChaincodeStubInterface, TokenID, BaseTokenSN, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var sn int
	var mwOwner MetaWallet

	// token check.
	if TokenID == BaseTokenSN {
		return errors.New("4210,Must TokenID and basetoken sn is not qeual")
	}

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if _, sn, err = mtc.getToken(stub, BaseTokenSN); err != nil {
		return err
	}
	if tk.BaseToken == sn {
		return errors.New("4210,Basetoken same as the existing value")
	}

	if _, exists := tk.TargetToken[sn]; exists == true {
		return errors.New("4201,Base token are in the target token list")
	}

	if mwOwner, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}
	if _, err = ecdsaSignVerify(mwOwner.Password,
		strings.Join([]string{tk.Owner, TokenID, BaseTokenSN, tkey}, "|"),
		sign); err != nil {
		return err
	}

	if BaseTokenSN == "0" {
		// base token unset.
		tk.BaseToken = 0
	} else {
		// token check.
		tk.BaseToken = sn
	}

	err = mtc.setTokenInfo(stub, TokenID, tk, "SetBase", args)
	return err
}

// TokenAddTarget - Set Token target token for STO DEX
func (mtc *MetaCoin) TokenAddTarget(stub shim.ChaincodeStubInterface, TokenID, TargetTokenSN, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var sn int

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if _, sn, err = mtc.getToken(stub, TargetTokenSN); err != nil {
		return err
	}

	var mwOwner MetaWallet
	if mwOwner, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(mwOwner.Password,
		strings.Join([]string{tk.Owner, TokenID, TargetTokenSN, tkey}, "|"),
		sign); err != nil {
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

	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenAddTarget", args)
	return err
}

// TokenRemoveTarget - Set Token remove token for STO DEX
func (mtc *MetaCoin) TokenRemoveTarget(stub shim.ChaincodeStubInterface, TokenID, TargetTokenSN, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var sn int
	if tk, sn, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if tk.TargetToken == nil {
		return errors.New("4202,Could not find target token in the target token list")
	}
	if _, exists := tk.TargetToken[sn]; exists == false {
		return errors.New("4202,Could not find target token in the target token list")
	}

	var mwOwner MetaWallet
	if mwOwner, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(mwOwner.Password,
		strings.Join([]string{tk.Owner, TokenID, TargetTokenSN, tkey}, "|"),
		sign); err != nil {
		return err
	}

	delete(tk.TargetToken, sn)

	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenRemoveTarget", args)
	return err
}

// TokenAddLogger - MRC100 token logger add
func (mtc *MetaCoin) TokenAddLogger(stub shim.ChaincodeStubInterface, TokenID, logger, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var mwOwner MetaWallet

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if _, err = mtc.getAddressInfo(stub, logger); err != nil {
		return errors.New("1202,The Logger is not exists")
	}

	if mwOwner, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(mwOwner.Password,
		strings.Join([]string{tk.Owner, TokenID, logger, tkey}, "|"),
		sign); err != nil {
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

	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenAddLogger", args)
	return err
}

// TokenRemoveLogger - MRC100 token logger remove
func (mtc *MetaCoin) TokenRemoveLogger(stub shim.ChaincodeStubInterface, TokenID, logger, sign, tkey string, args []string) error {
	var tk Token
	var err error

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if tk.Logger == nil {
		return errors.New("4202,Could not find logger in the logger list")
	}
	if _, exists := tk.Logger[logger]; exists == false {
		return errors.New("4202,Could not find logger in the logger list")
	}

	var mwOwner MetaWallet
	if mwOwner, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(mwOwner.Password,
		strings.Join([]string{tk.Owner, TokenID, logger, tkey}, "|"),
		sign); err != nil {
		return err
	}

	delete(tk.Logger, logger)

	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenRemoveLogger", args)
	return err
}

// TokenUpdate - Token Information update.
func (mtc *MetaCoin) TokenUpdate(stub shim.ChaincodeStubInterface, TokenID, url, info, image, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var ownerData MetaWallet
	var isUpdate bool

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if ownerData, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}
	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join([]string{TokenID, url, info, image, tkey}, "|"),
		sign); err != nil {
		return err
	}

	isUpdate = false

	if len(strings.TrimSpace(url)) > 0 && tk.URL != url {
		tk.URL = url
		isUpdate = true
	}

	if len(strings.TrimSpace(info)) > 0 && tk.Information != info {
		tk.Information = info
		isUpdate = true
	}

	if len(strings.TrimSpace(image)) > 0 && tk.Image != image {
		tk.Image = image
		isUpdate = true
	}

	if !isUpdate {
		return errors.New("4900,No data change")
	}

	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenUpdate", args)
	return err
}

// TokenBurning - Token Information update.
func (mtc *MetaCoin) TokenBurning(stub shim.ChaincodeStubInterface, TokenID, amount, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var ownerData MetaWallet
	var BurnningAmount, BurnAmount decimal.Decimal

	if BurnAmount, err = decimal.NewFromString(amount); err != nil {
		return errors.New("1206,Amount is must integer")
	}

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if ownerData, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join([]string{TokenID, amount, tkey}, "|"),
		sign); err != nil {
		return err
	}

	if err = mtc.subtractToken(stub, &ownerData, TokenID, amount); err != nil {
		return err
	}
	if err = mtc.setAddressInfo(stub, tk.Owner, ownerData, "ownerBurning", args); err != nil {
		return err
	}

	if BurnningAmount, err = decimal.NewFromString(tk.BurnningAmount); err != nil {
		tk.BurnningAmount = "0"
		BurnningAmount, _ = decimal.NewFromString(tk.BurnningAmount)
	}
	tk.BurnningAmount = BurnningAmount.Add(BurnAmount).String()
	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenBurning", args)

	return err
}

// TokenIncrease - Token Information update.
func (mtc *MetaCoin) TokenIncrease(stub shim.ChaincodeStubInterface, TokenID, amount, sign, tkey string, args []string) error {
	var tk Token
	var err error
	var ownerData MetaWallet
	var TotalAmount, IncrAmount decimal.Decimal

	if IncrAmount, err = decimal.NewFromString(amount); err != nil {
		return errors.New("1206,Amount is must integer")
	}

	if tk, _, err = mtc.getToken(stub, TokenID); err != nil {
		return err
	}

	if ownerData, err = mtc.getAddressInfo(stub, tk.Owner); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join([]string{TokenID, amount, tkey}, "|"),
		sign); err != nil {
		return err
	}

	if err = mtc.addToken(stub, &ownerData, TokenID, amount, 0); err != nil {
		return err
	}
	if err = mtc.setAddressInfo(stub, tk.Owner, ownerData, "ownerIncrease", args); err != nil {
		return err
	}

	if TotalAmount, err = decimal.NewFromString(tk.TotalSupply); err != nil {
		tk.TotalSupply = "0"
		TotalAmount, _ = decimal.NewFromString(tk.TotalSupply)
	}
	tk.TotalSupply = TotalAmount.Add(IncrAmount).String()
	err = mtc.setTokenInfo(stub, TokenID, tk, "tokenIncrease", args)

	return err
}

// MRC011Create - MRC011 Creator
func (mtc *MetaCoin) MRC011Create(stub shim.ChaincodeStubInterface, mrc011id, creator, name, totalsupply, validitytype, istransfer, startdate, enddate, term, code, data, signature, tkey string, args []string) error {
	var err error
	var i int
	var j int64
	var buf string
	var mrc011 MRC011
	var creatorData MetaWallet

	if creatorData, err = mtc.getAddressInfo(stub, creator); err != nil {
		return err
	}

	mrc011.Creator = creator
	mrc011.Name = name
	if i, err = strtoint(totalsupply); err != nil {
		return errors.New("1003,The totalsupply must be integer")
	}
	if i < 1 {
		return errors.New("1003,The totalsupply must be bigger then 0")
	}
	mrc011.TotalSupply = i
	mrc011.UsedCount = 0
	mrc011.PublishCount = 0
	mrc011.RemainCount = i
	mrc011.StartDate = 0
	mrc011.EndDate = 0
	mrc011.Term = 0

	if validitytype == "term" {
		if j, err = strtoint64(startdate); err != nil {
			return errors.New("1003,The Start_date must be integer")
		}
		mrc011.StartDate = j
		if j, err = strtoint64(enddate); err != nil {
			return errors.New("1003,The end_date must be integer")
		}
		if j < time.Now().Unix() {
			return errors.New("1403,The end_date must be bigger then current timestamp")
		}
		mrc011.EndDate = j
	} else if validitytype == "duration" {
		if i, err = strtoint(term); err != nil {
			return errors.New("1003,The Term must be integer")
		}
		if i < 1 {
			return errors.New("1003,The term must be bigger then 0")
		}
		mrc011.Term = i
	} else {
		return errors.New("1003,The Validity_type must be term or duration")
	}

	if istransfer == "0" || istransfer == "" {
		mrc011.IsTransfer = 0
	} else {
		mrc011.IsTransfer = 1
	}

	time.Now()
	buf = strings.Join([]string{creator, name, totalsupply, validitytype, istransfer, code, data, tkey}, "|")
	if _, err = ecdsaSignVerify(creatorData.Password,
		buf,
		signature); err != nil {
		return err
	}

	mrc011.Code = code
	mrc011.Data = data

	return mtc.setMRC011(stub, mrc011id, mrc011, "mrc011create", args)
}

// Mrc100Payment : Game payment
func (mtc *MetaCoin) Mrc100Payment(stub shim.ChaincodeStubInterface, to, TokenID, tag, userlist, gameid, gamememo string, args []string) error {
	var err error
	var ownerData, playerData MetaWallet
	var playerList []MRC100Payment

	if ownerData, err = mtc.getAddressInfo(stub, to); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(userlist), &playerList); err != nil {
		return errors.New("4209,Invalid UserLIst data")
	}
	if len(playerList) < 0 {
		return errors.New("2007,Playerlist need more than one")
	}
	if len(playerList) > 32 {
		return errors.New("2007,Playerlist should be less than 32")
	}

	for _, elements := range playerList {
		if playerData, err = mtc.getAddressInfo(stub, elements.Address); err != nil {
			return err
		}

		if _, err = ecdsaSignVerify(playerData.Password, strings.Join([]string{elements.Address, to, TokenID, elements.Amount, elements.TKey}, "|"), elements.Signature); err != nil {
			return err
		}
		if elements.Amount == "" {
			return errors.New("1107,Amount is must integer")
		}

		if elements.Amount != "0" {
			if err = mtc.moveToken(stub, &playerData, &ownerData, TokenID, elements.Amount, 0); err != nil {
				return err
			}
		}

		if err = mtc.setAddressInfo(stub, elements.Address, playerData, "mrc100payment",
			[]string{elements.Address, to, elements.Amount, TokenID, elements.Signature, "", tag, elements.Memo, elements.TKey}); err != nil {
			return err
		}
	}

	if err = mtc.setAddressInfo(stub, to, ownerData, "mrc100paymentrecv", args); err != nil {
		return err
	}
	return nil
}

// MRC030Create  Vote create
func (mtc *MetaCoin) MRC030Create(stub shim.ChaincodeStubInterface, mrc030id, Creator, Title, Description, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType, URL, Question, SignNeed, sign, tkey string, args []string) error {
	var err error
	var CreatorData MetaWallet
	var decReward, totReward, decMaxRewardRecipient decimal.Decimal
	var iStartDate, iEndDate int64
	var iRewardType, iMaxRewardRecipient, iRewardToken int
	var vote MRC030
	var data []byte
	var q [20]MRC030Question

	if data, err = stub.GetState(mrc030id); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data != nil {
		return errors.New("6100,MRC030 [" + mrc030id + "] is already exists")
	}

	if CreatorData, err = mtc.getAddressInfo(stub, Creator); err != nil {
		return err
	}

	if decMaxRewardRecipient, err = decimal.NewFromString(MaxRewardRecipient); err != nil {
		return errors.New("1101,MaxRewardRecipient must be an integer string")
	}
	if !decMaxRewardRecipient.IsPositive() {
		return errors.New("1101,MaxRewardRecipient must be an greater then 0")
	}
	if iMaxRewardRecipient, err = strconv.Atoi(MaxRewardRecipient); err != nil {
		return errors.New("1101,MaxRewardRecipient must be an integer string")
	}

	if iRewardType, err = strconv.Atoi(RewardType); err != nil {
		return errors.New("1101,RewardType must be an integer string")
	}
	if iRewardType != 10 && iRewardType != 20 {
		return errors.New("1101,RewardType must be an 10 or 20")
	}

	if decReward, err = decimal.NewFromString(Reward); err != nil {
		return errors.New("1101,Reward must be an integer string")
	}
	if decReward.IsNegative() {
		return errors.New("1101,Reward must be an not negative")
	}

	if iRewardType == 20 {
		if iMaxRewardRecipient > 100 {
			return errors.New("1101,The maximum reward recipient is 100")
		}
		if decReward.IsZero() {
			return errors.New("1101, If the rewardtype is 20, the reward must be greater than 0")
		}
	}

	if iStartDate, err = strconv.ParseInt(StartDate, 10, 64); err != nil {
		return errors.New("1101,StartDate must be an integer string")
	}
	if iEndDate, err = strconv.ParseInt(EndDate, 10, 64); err != nil {
		return errors.New("1101,EndDate must be an integer string")
	}

	nowTime := time.Now().Unix()
	if iEndDate < nowTime {
		return errors.New("1101,The EndDate must be greater then now")
	}
	if iEndDate < iStartDate {
		return errors.New("1101,The EndDate must be greater then StartDate")
	}

	if _, iRewardToken, err = mtc.getToken(stub, RewardToken); err != nil {
		return err
	}

	if decReward.IsPositive() {
		totReward = decReward.Mul(decMaxRewardRecipient)
		if err = mtc.subtractToken(stub, &CreatorData, RewardToken, totReward.String()); err != nil {
			return err
		}
	} else {
		totReward = decimal.Zero
	}
	if err = mtc.setAddressInfo(stub, Creator, CreatorData, "mrc030create",
		[]string{Creator, mrc030id, totReward.String(), RewardToken, sign, "0", "", "", tkey}); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(Question), &q); err != nil {
		return errors.New("3290,Question is in the wrong data")
	}

	vote.QuestionCount = 0
	vote.Question = make([]MRC030Question, 0, 20)
	vote.QuestionInfo = make([]MRC030QuestionInfo, 0, 20)
	for index, ele := range q {
		if len(ele.Question) == 0 {
			break
		}
		vote.Question = append(vote.Question, ele)
		vote.QuestionInfo = append(vote.QuestionInfo, MRC030QuestionInfo{0, []int{}})
		vote.QuestionCount++
		for idx2, ele2 := range ele.Item {
			if len(ele2.Answer) == 0 && len(ele2.URL) == 0 {
				vote.Question[index].Item = vote.Question[index].Item[0:idx2]
				break
			}
			if idx2 >= 4 {
				vote.Question[index].Item = vote.Question[index].Item[0:idx2]
				break
			}
			vote.QuestionInfo[index].AnswerCount++
			vote.QuestionInfo[index].SubAnswerCount = append(vote.QuestionInfo[index].SubAnswerCount, 0)
			if len(ele2.SubQuery) == 0 {
				vote.Question[index].Item[idx2].SubItem = make([]MRC030SubItem, 0, 0)
				continue
			}
			for idx3, ele3 := range ele2.SubItem {
				if len(ele3.SubAnswer) == 0 && len(ele3.URL) == 0 {
					vote.Question[index].Item[idx2].SubItem = vote.Question[index].Item[idx2].SubItem[0:idx3]
					break
				}
				if idx3 >= 4 {
					vote.Question[index].Item[idx2].SubItem = vote.Question[index].Item[idx2].SubItem[0:idx3]
					break
				}
				vote.QuestionInfo[index].SubAnswerCount[idx2]++
			}
		}
	}

	if vote.QuestionCount == 0 {
		return errors.New("3290,Question is empty")
	}
	if _, err = ecdsaSignVerify(CreatorData.Password,
		strings.Join([]string{Creator, Title, Reward, RewardToken, MaxRewardRecipient, RewardType, tkey}, "|"),
		sign); err != nil {
		return err
	}
	vote.Creator = Creator
	vote.Description = Description
	vote.StartDate = iStartDate
	vote.EndDate = iEndDate
	vote.Reward = Reward
	vote.RewardToken = iRewardToken
	vote.RewardType = iRewardType
	vote.TotalReward = totReward.String()
	vote.MaxRewardRecipient = iMaxRewardRecipient
	vote.Title = Title
	vote.IsFinish = 0
	vote.URL = URL
	vote.Voter = make(map[string]int)

	if SignNeed == "1" {
		vote.IsNeedSign = 1
	} else {
		vote.IsNeedSign = 0
	}

	if err = mtc.setMRC030(stub, mrc030id, vote, "mrc030", []string{Creator, mrc030id, Title, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType}); err != nil {
		return err
	}
	return nil
}

// MRC030Join  Vote join
func (mtc *MetaCoin) MRC030Join(stub shim.ChaincodeStubInterface, mrc030id, Voter, Answer, voteCreatorSign, sign string, args []string) error {
	var err error
	var voterData, voteCreatorData MetaWallet
	var vote MRC030
	var voting MRC031
	var mrc031key string
	var data []byte
	var AnswerTemp [20]MRC031Answer
	var currentAnswer int

	nowTime := time.Now().Unix()

	if vote, err = mtc.getMRC030(stub, mrc030id); err != nil {
		return err
	}
	if vote.StartDate > nowTime {
		return errors.New("8100,Voting is not start")
	}
	if vote.EndDate < nowTime {
		return errors.New("8100,Voting is finish")
	}
	if vote.IsFinish != 0 {
		return errors.New("4922,This vote has already ended")
	}

	if _, exists := vote.Voter[Voter]; exists != false {
		return errors.New("6100,MRC031 [" + mrc030id + "] is already voting")
	}

	if vote.Creator == Answer {
		return errors.New("6100,Vote creators cannot participate")
	}

	if vote.RewardType == 10 && len(vote.Voter) >= vote.MaxRewardRecipient {
		fmt.Printf("vote.MaxRewardRecipient : %d\n", vote.MaxRewardRecipient)
		fmt.Printf("vote.Voter : %v\n", vote.Voter)
		return errors.New("3290,No more voting")
	}

	mrc031key = mrc030id + "_" + Voter
	voting = MRC031{
		Regdate: nowTime,
		Voter:   Voter,
		JobType: "mrc031",
		JobDate: time.Now().Unix(),
		JobArgs: "",
	}

	if err = json.Unmarshal([]byte(Answer), &AnswerTemp); err != nil {
		return errors.New("3290,Answer is in the wrong data")
	}

	currentAnswer = 0
	voting.Answer = make([]MRC031Answer, 0, 20)
	for i, a := range AnswerTemp {
		if i >= vote.QuestionCount {
			break
		}
		if vote.QuestionInfo[i].AnswerCount > 0 && (a.Answer < 1 || a.Answer > vote.QuestionInfo[i].AnswerCount) {
			return errors.New("3290,Answer [" + strconv.Itoa(i) + "] step 1 is out of range")
		}
		if vote.QuestionInfo[i].SubAnswerCount[a.Answer-1] > 0 && (a.SubAnswer < 1 || a.SubAnswer > vote.QuestionInfo[i].SubAnswerCount[a.Answer-1]) {
			return errors.New("3290,Answer [" + strconv.Itoa(i) + "] step 2 is out of range")
		}
		voting.Answer = append(voting.Answer, a)
		currentAnswer++
	}

	if currentAnswer < vote.QuestionCount {
		return errors.New("3290,There must be [" + strconv.Itoa(vote.QuestionCount) + "] answers.")
	}

	if voterData, err = mtc.getAddressInfo(stub, Voter); err != nil {
		return err
	}

	if _, err = ecdsaSignVerify(voterData.Password,
		strings.Join([]string{Voter, mrc030id}, "|"),
		sign); err != nil {
		return errors.New("2010,Invalid voter signature")
	}

	if vote.IsNeedSign == 1 {
		if voteCreatorData, err = mtc.getAddressInfo(stub, vote.Creator); err != nil {
			return err
		}
		if _, err = ecdsaSignVerify(voteCreatorData.Password,
			strings.Join([]string{Voter, mrc030id}, "|"),
			voteCreatorSign); err != nil {
			return errors.New("2010,Invalid vote creator signature")
		}
	}

	fmt.Printf("Reward : [%s]  vote.RewardType : [%d]\n", vote.Reward, vote.RewardType)
	if vote.RewardType == 10 {
		if vote.Reward != "0" {
			fmt.Printf("addToken : [%s]  vote.RewardType : [%s]\n", Voter, vote.Reward)
			if err = mtc.addToken(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
				return err
			}
			if err = mtc.setAddressInfo(stub, Voter, voterData, "mrc030reward", []string{mrc030id, Voter, vote.Reward, strconv.Itoa(vote.RewardToken), sign, "0", "", "", ""}); err != nil {
				return err
			}
		}
		vote.Voter[Voter] = 1
	} else {
		vote.Voter[Voter] = 0
	}
	// from, to, amount, tokenid, sign, unlockdate, tag, memo, tkey
	if data, err = json.Marshal([]string{mrc030id, Voter, vote.Reward, strconv.Itoa(vote.RewardToken), sign, "0", "", "", "", voteCreatorSign}); err == nil {
		voting.JobArgs = string(data)
	}
	if data, err = json.Marshal(voting); err != nil {
		return errors.New("4204,Invalid MRC031 data format")
	}
	if err = stub.PutState(mrc031key, data); err != nil {
		return err
	}

	vote.JobType = "mrc030update"
	if data, err = json.Marshal(vote); err != nil {
		return errors.New("4204,Invalid MRC031 data format")
	}
	if err = stub.PutState(mrc030id, data); err != nil {
		return err
	}

	return nil
}

// MRC030Finish  Vote join
func (mtc *MetaCoin) MRC030Finish(stub shim.ChaincodeStubInterface, mrc030id string, args []string) error {
	var err error
	var voterData, CreatorData MetaWallet
	var vote MRC030
	var data []byte
	var JoinerList []string
	var decRefund decimal.Decimal
	var i int
	var key string
	if vote, err = mtc.getMRC030(stub, mrc030id); err != nil {
		return err
	}
	if vote.IsFinish != 0 {
		return errors.New("4922,This vote has already ended")
	}

	nowTime := time.Now().Unix()
	if vote.RewardType == 20 {
		if vote.EndDate > nowTime {
			return errors.New("4922,This is an ongoing vote")
		}

		fmt.Printf("Finish Reward : [%s]  vote.RewardType : [%d]\n", vote.Reward, vote.RewardType)
		// 추첨
		JoinerList = make([]string, len(vote.Voter))

		if len(vote.Voter) < vote.MaxRewardRecipient { // 모두 추첨 대상
			fmt.Printf("Reward ALL\n")

			for key = range vote.Voter {
				fmt.Printf("Address : [%s]\n", key)
				if voterData, err = mtc.getAddressInfo(stub, key); err != nil {
					continue
				}
				fmt.Printf("AddToken : [%s], Token %d, Amount %s\n", key, vote.RewardToken, vote.Reward)
				if err = mtc.addToken(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
					continue
				}
				fmt.Printf("Address : %s\n", key)
				if err = mtc.setAddressInfo(stub, key, voterData, "mrc030reward", []string{mrc030id, key, vote.Reward, strconv.Itoa(vote.RewardToken), "", "0", "", "", ""}); err != nil {
					continue
				}
				fmt.Printf("Voter SET : %s\n", key)
				vote.Voter[key] = 1
			}
		} else if len(vote.Voter) < (vote.MaxRewardRecipient * 2) { // 받지 못할 사람을 지정
			fmt.Printf("Reward Except\n")
			for key = range vote.Voter {
				JoinerList[i] = key
				i++
			}

			for vote.MaxRewardRecipient < len(JoinerList) {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(len(JoinerList))))
				if err != nil {
					return err
				}
				JoinerList = remove(JoinerList, int(n.Int64()))
			}
			for _, key = range JoinerList {
				fmt.Printf("Address : [%s]\n", key)
				if voterData, err = mtc.getAddressInfo(stub, key); err != nil {
					continue
				}
				fmt.Printf("AddToken : [%s], Token %d, Amount %s\n", key, vote.RewardToken, vote.Reward)
				if err = mtc.addToken(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
					continue
				}
				fmt.Printf("setAddressInfo : %s\n", key)
				if err = mtc.setAddressInfo(stub, key, voterData, "mrc030reward", []string{mrc030id, key, vote.Reward, strconv.Itoa(vote.RewardToken), "", "0", "", "", ""}); err != nil {
					continue
				}
				fmt.Printf("Voter SET : %s\n", key)
				vote.Voter[key] = 1
			}
		} else { // 받을 사람을 지정
			fmt.Printf("Reward Select\n")
			for key := range vote.Voter {
				JoinerList[i] = key
				i++
			}
			i = 0
			for i < vote.MaxRewardRecipient {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(len(JoinerList))))
				if err != nil {
					return err
				}
				key = JoinerList[int(n.Int64())]
				fmt.Printf("Address : [%s]\n", key)
				JoinerList = remove(JoinerList, int(n.Int64()))
				if voterData, err = mtc.getAddressInfo(stub, key); err != nil {
					continue
				}
				fmt.Printf("AddToken : [%s], Token %d, Amount %s\n", key, vote.RewardToken, vote.Reward)
				if err = mtc.addToken(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
					continue
				}
				fmt.Printf("setAddressInfo : %s\n", key)
				if err = mtc.setAddressInfo(stub, key, voterData, "mrc030reward", []string{mrc030id, key, vote.Reward, strconv.Itoa(vote.RewardToken), "", "0", "", "", ""}); err != nil {
					continue
				}
				fmt.Printf("Voter SET : %s\n", key)
				vote.Voter[key] = 1
				i++
			}
		}
	} else {
		if vote.MaxRewardRecipient > len(vote.Voter) {
			if vote.EndDate > nowTime {
				return errors.New("4922,This is an ongoing vote")
			}
		}
	}

	if vote.MaxRewardRecipient > len(vote.Voter) {
		// 미 투표분 환불
		iReward, _ := strconv.ParseInt(vote.Reward, 10, 64)
		decRefund = decimal.New(int64(vote.MaxRewardRecipient-len(vote.Voter)), 0).Mul(decimal.New(iReward, 0))
		if decRefund.IsPositive() {
			CreatorData, err = mtc.getAddressInfo(stub, vote.Creator)
			mtc.addToken(stub, &CreatorData, strconv.Itoa(vote.RewardToken), decRefund.String(), 0)
			mtc.setAddressInfo(stub, vote.Creator, CreatorData, "mrc030refund", []string{mrc030id, vote.Creator, decRefund.String(), strconv.Itoa(vote.RewardToken), "", "0", "", "", ""})
		}
	}
	vote.JobType = "mrc030finish"
	vote.IsFinish = 1
	if data, err = json.Marshal(vote); err != nil {
		return errors.New("4204,Invalid MRC031 data format")
	}
	if err = stub.PutState(mrc030id, data); err != nil {
		return err
	}

	return nil
}

// Mrc100Reward : Game reward
func (mtc *MetaCoin) Mrc100Reward(stub shim.ChaincodeStubInterface, from, TokenID, userlist, gameid, gamememo, sign, tkey string, args []string) error {
	var err error
	var ownerData, playerData MetaWallet
	var playerList []MRC100Reward
	var checkList []string

	if ownerData, err = mtc.getAddressInfo(stub, from); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(userlist), &playerList); err != nil {
		println(userlist)
		println(err.Error())
		return errors.New("4209,Invalid UserLIst data")
	}
	if len(playerList) < 0 {
		return errors.New("2007,Playerlist need more than one")
	}
	if len(playerList) > 32 {
		return errors.New("2007,Playerlist should be less than 32")
	}

	checkList = append(checkList, TokenID)

	for _, elements := range playerList {
		if elements.Amount == "" {
			return errors.New("1107,Amount is must integer")
		}
		checkList = append(checkList, elements.Address, elements.Amount, elements.Tag)
	}
	checkList = append(checkList, tkey)

	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join(checkList, "|"),
		sign); err != nil {
		return err
	}

	for _, elements := range playerList {
		if playerData, err = mtc.getAddressInfo(stub, elements.Address); err != nil {
			return err
		}
		if elements.Amount != "" && elements.Amount != "0" {
			if err = mtc.moveToken(stub, &ownerData, &playerData, TokenID, elements.Amount, 0); err != nil {
				return err
			}
		}

		if err = mtc.setAddressInfo(stub, elements.Address, playerData, "mrc030reward",
			[]string{from, elements.Address, elements.Amount, TokenID, sign, "", elements.Tag, elements.Memo, ""}); err != nil {
			return err
		}
	}

	if err = mtc.setAddressInfo(stub, from, ownerData, "mrc030payment", args); err != nil {
		return err
	}
	return nil
}

// Mrc100Log Game log
func (mtc *MetaCoin) Mrc100Log(stub shim.ChaincodeStubInterface, key, token, logger, log, signature, tkey string, args []string) (string, error) {
	var err error
	var tk Token
	var ownerData MetaWallet
	var mrcLog MRC100Log
	var dat []byte

	if tk, _, err = mtc.getToken(stub, token); err != nil {
		return "", err
	}

	if tk.Owner != logger {
		if _, exists := tk.Logger[logger]; exists == false {
			return "", errors.New("6030,you do not have permission to log this token")
		}
	}
	if ownerData, err = mtc.getAddressInfo(stub, logger); err != nil {
		return "", err
	}

	if tk.Type != "100" && tk.Type != "101" {
		return "", errors.New("6032,This token cannot log")
	}
	if _, err = ecdsaSignVerify(ownerData.Password,
		strings.Join([]string{token, logger, log, tkey}, "|"),
		signature); err != nil {
		return "", err
	}

	mrcLog = MRC100Log{Regdate: time.Now().Unix(),
		Token:   tk.Token,
		Logger:  logger,
		JobType: "MRC100LOG",
		JobArgs: log}

	dat, err = stub.GetState(key)
	if err == nil && dat != nil {
		return "", errors.New("6013,MRC100 already exists")
	}

	dat, _ = json.Marshal(mrcLog)
	if err := stub.PutState(key, dat); err != nil {
		return "", errors.New("8600,Hyperledger internal error - " + err.Error())
	}

	fmt.Printf("MRC100Log [%s] [%s]\n", key, token)
	return key, nil
}

// Mrc100get - MRC-100 Protocol Add
func (mtc *MetaCoin) Mrc100get(stub shim.ChaincodeStubInterface, mrc100Key string) (string, error) {
	var dat []byte
	var err error

	if strings.Index(mrc100Key, "MRC100_") != 0 {
		return "", errors.New("6102,invalid MRC100 data address")
	}

	dat, err = stub.GetState(mrc100Key)
	if err != nil {
		return "", errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return "", errors.New("6004,MRC100 data not exist")
	}
	return string(dat), nil
}

// Init is a no-op
func (mtc *MetaCoin) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke has two functions
// put - takes two arguments, a key and value, and stores them in the state
// remove - takes one argument, a key, and removes if from the state
func (mtc *MetaCoin) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	var tkey string
	var value string
	var err error

	function, args := stub.GetFunctionAndParameters()
	for idx, arg := range args {
		args[idx] = strings.TrimSpace(arg)
	}
	switch function {
	case "dummy":
		if len(args) < 1 {
			return shim.Error("1000,get operation must include one arguments, index")
		}
		if args[0] < "0" || args[0] > "9" {
			return shim.Error("1100,index is must 0 to 9")
		}

		if err = stub.PutState("DUMMY_IDX_"+args[0], []byte(args[0])); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "get":
		if len(args) < 1 {
			return shim.Error("1000,get operation must include one arguments, address")
		}

		if strings.Index(args[0], "MRC") == 0 {
			return shim.Error("1000,invalid address")
		}

		valuet, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		if valuet == nil {
			return shim.Error("1000,Key not exist")
		}

		return shim.Success([]byte(valuet))

	case "totalSupply":
		value, err = mtc.BalanceOf(stub, CoinName)

		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "balanceOf":
		if len(args) < 1 {
			return shim.Error("1000,balanceOf operation must include one argument : address")
		}
		address := args[0]
		if IsInvalidID(address) {
			return shim.Error("Invalid address format")
		}
		value, err = mtc.BalanceOf(stub, address)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "newwallet":
		if len(args) < 3 {
			return shim.Error("1000,balanceOf operation must include five arguments : walletID, publicKey, addinfo")
		}

		walletid := args[0]
		publicKey := args[1]
		addinfo := args[2]

		err = mtc.NewWallet(stub, walletid, publicKey, addinfo, args)
		if err != nil {
			return shim.Error(err.Error())
		}
		break

	case "transfer":
		if len(args) < 9 {
			return shim.Error("1000,transfer operation must include four arguments : fromAddr, toAddr, amount, tokenID, signature, unlockdate, tag, memo, tkey")
		}

		fromAddr := args[0]
		toAddr := args[1]
		amount := args[2]
		tokenID := args[3]
		sign := args[4]
		unlockdate := args[5]
		//tag := args[6]
		//memo := args[7]
		tkey = args[8]

		err = mtc.Transfer(stub, fromAddr, toAddr, amount, tokenID, unlockdate, sign, tkey, args)
		if err != nil {
			return shim.Error(err.Error())
		}
		break

	case "exchange":
		if len(args) < 20 {
			return shim.Error("1000,exchange operation must include four arguments : " +
				"fromAddr, fromAmount, fromToken, fromFeeSendto, fromFee, fromFeeToken, " +
				"fromTag, fromMemo, fromSign, " +
				"toAddr, toAmount, toToken, toFeeSendto, toFee, toFeeToken, " +
				"toTag, toMemo, toSign, " +
				"fromTKey, toTKey")
		}

		//
		fromAddr := args[0]
		fromAmount := args[1]
		fromToken := args[2]
		fromFeeSendto := args[3]
		fromFee := args[4]
		fromFeeToken := args[5]
		// fromtag := args[6]
		// frommemo := args[7]
		fromSign := args[8]
		toAddr := args[9]
		toAmount := args[10]
		toToken := args[11]
		toFeeSendto := args[12]
		toFee := args[13]
		toFeeToken := args[14]
		// totag := args[15]
		// tomemo := args[16]
		toSign := args[17]
		fromTKey := args[18]
		toTKey := args[19]
		if err = mtc.Exchange(stub,
			fromAddr, fromAmount, fromToken, fromFeeSendto, fromFee, fromFeeToken, fromTKey, fromSign,
			toAddr, toAmount, toToken, toFeeSendto, toFee, toFeeToken, toTKey, toSign,
			args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "tokenRegister":
		if len(args) < 3 {
			return shim.Error("1000,tokenRegister must include one arguments : tokeninfo, sign, tkey")
		}
		data := args[0]
		sign := args[1]
		tkey := args[2]
		value, err = mtc.TokenRegister(stub, data, sign, tkey)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc020":
		if len(args) < 8 {
			return shim.Error("1000,mrc20 operation must include four arguments : owner, algorithm, data, publickey, opendata, referencekey, sign, tkey")
		}

		owner := args[0]
		algorithm := args[1]
		data := args[2]
		publickey := args[3]
		opendate := args[4]
		referencekey := args[5]
		sign := args[6]
		tkey := args[7]

		if value, err = mtc.Mrc020set(stub, owner, algorithm, data, publickey, opendate, referencekey, sign, tkey); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc020get":
		if len(args) < 1 {
			return shim.Error("1000,mrc20get operation must include four arguments : mrc020Key")
		}

		mrc020Key := args[0]

		if value, err = mtc.Mrc020get(stub, mrc020Key); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc040get":
		if len(args) < 1 {
			return shim.Error("1000,mrc040get operation must include four arguments : mrc040Key")
		}

		mrc040Key := args[0]

		if value, err = mtc.Mrc040get(stub, mrc040Key); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "stodexRegister":
		if len(args) < 9 {
			return shim.Error("1000,stodexRegister operation must include four arguments : owner, side, BaseToken, TargetToken, price, qtt, exchangeItemPK, sign, tkey")
		}
		owner := args[0]
		side := args[1]
		BaseToken := args[2]
		TargetToken := args[3]
		price := args[4]
		qtt := args[5]
		exchangeItemPK := args[6]
		sign := args[7]
		tkey := args[8]

		if err = mtc.StodexRegister(stub, owner, side, BaseToken, TargetToken, price, qtt, exchangeItemPK, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "stodexUnRegister":
		if len(args) < 4 {
			return shim.Error("1000,stodexUnRegister operation must include four arguments : owner, exchangeItemPK, sign, tkey")
		}
		owner := args[0]
		exchangeItemPK := args[1]
		sign := args[2]
		tkey := args[3]

		if err = mtc.StodexUnRegister(stub, owner, exchangeItemPK, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "stodexExchange":
		if len(args) < 6 {
			return shim.Error("1000,stodexExchange operation must include four arguments : requester, qtt, ExchangeItemPK, ExchangePK, sign, tkey")
		}
		requester := args[0]
		qtt := args[1]
		exchangeItemPK := args[2]
		exchangePK := args[3]
		sign := args[4]
		tkey := args[5]

		if err = mtc.StodexExchange(stub, requester, qtt, exchangeItemPK, exchangePK, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "tokenSetBase":
		if len(args) < 4 {
			return shim.Error("1000,tokenSetBase operation must include four arguments : TokenID, BaseTokenSN, sign, tkey")
		}
		TokenID := args[0]
		BaseTokenSN := args[1]
		sign := args[2]
		tkey := args[3]

		if err = mtc.TokenSetBase(stub, TokenID, BaseTokenSN, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "tokenAddTarget":
		if len(args) < 4 {
			return shim.Error("1000,tokenAddTarget operation must include four arguments : TokenID, TargetTokenSN, sign, tkey")
		}
		TokenID := args[0]
		TargetTokenSN := args[1]
		sign := args[2]
		tkey := args[3]

		if err = mtc.TokenAddTarget(stub, TokenID, TargetTokenSN, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "tokenRemoveTarget":
		if len(args) < 2 {
			return shim.Error("1000,tokenRemoveTarget operation must include four arguments : TokenID, TargetTokenSN, sign, tkey")
		}
		TokenID := args[0]
		TargetTokenSN := args[1]
		sign := args[2]
		tkey := args[3]

		if err = mtc.TokenRemoveTarget(stub, TokenID, TargetTokenSN, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

		break

	case "tokenUpdate":
		if len(args) < 6 {
			return shim.Error("1000,tokenRemoveTarget operation must include four arguments : TokenID, url, info, image, sign, tkey")
		}
		TokenID := args[0]
		url := args[1]
		info := args[2]
		image := args[3]
		sign := args[4]
		tkey := args[5]
		if err = mtc.TokenUpdate(stub, TokenID, url, info, image, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

		break

	case "tokenBurning":
		if len(args) < 4 {
			return shim.Error("1000,tokenBurn operation must include four arguments : TokenID, amount, sign, tkey")
		}
		TokenID := args[0]
		amount := args[1]
		sign := args[2]
		tkey = args[3]
		if err = mtc.TokenBurning(stub, TokenID, amount, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

		break

	case "tokenIncrease":
		if len(args) < 4 {
			return shim.Error("1000,tokenIncrease operation must include four arguments : TokenID, amount, sign, tkey")
		}
		TokenID := args[0]
		amount := args[1]
		sign := args[2]
		tkey = args[3]
		if err = mtc.TokenIncrease(stub, TokenID, amount, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

		break

	case "tokenAddLogger":
		if len(args) < 4 {
			return shim.Error("1000,tokenAddLogger operation must include four arguments : TokenID, logger, sign, tkey")
		}
		TokenID := args[0]
		logger := args[1]
		sign := args[2]
		tkey := args[3]

		if err = mtc.TokenAddLogger(stub, TokenID, logger, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "tokenRemoveLogger":
		if len(args) < 4 {
			return shim.Error("1000,tokenRemoveLogger operation must include four arguments : TokenID, logger, sign, tkey")
		}
		TokenID := args[0]
		logger := args[1]
		sign := args[2]
		tkey := args[3]

		if err = mtc.TokenRemoveLogger(stub, TokenID, logger, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

		break

	case "mrc011create":
		if len(args) < 13 {
			return shim.Error("1000,mrc011create operation must include four arguments : mrc011id, creator, name, totalsupply, validitytype, istransfer, startdate, enddate, term, code, data, signature, tkey")
		}
		mrc011id := args[0]
		creator := args[1]
		name := args[2]
		totalsupply := args[3]
		validitytype := args[4]
		istransfer := args[5]
		startdate := args[6]
		enddate := args[7]
		term := args[8]
		code := args[9]
		data := args[10]
		signature := args[11]
		tkey := args[12]

		if err = mtc.MRC011Create(stub, mrc011id, creator, name, totalsupply, validitytype, istransfer, startdate, enddate, term, code, data, signature, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

		break

	case "mrc100Payment":
		if len(args) < 6 {
			return shim.Error("1000,mrc100Payment operation must include four arguments : to, TokenID, tag, userlist, gameid, gamememo")
		}
		to := args[0]
		TokenID := args[1]
		tag := args[2]
		userlist := args[3]
		gameid := args[4]
		gamememo := args[5]
		if err = mtc.Mrc100Payment(stub, to, TokenID, tag, userlist, gameid, gamememo, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "mrc100Reward":
		if len(args) < 7 {
			return shim.Error("1000,mrc100Reward operation must include four arguments : from, TokenID, userlist, gameid, gamememo, sign, tkey")
		}
		from := args[0]
		TokenID := args[1]
		userlist := args[2]
		gameid := args[3]
		gamememo := args[4]
		sign := args[5]
		tkey := args[6]
		if err = mtc.Mrc100Reward(stub, from, TokenID, userlist, gameid, gamememo, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "mrc100Log":
		if len(args) < 6 {
			return shim.Error("1000,mrc100Log operation must include four arguments : key, TokenID, logger, log, sign, tkey")
		}
		mrc100Key := args[0]
		TokenID := args[1]
		logger := args[2]
		log := args[3]
		sign := args[4]
		tkey := args[5]
		if value, err = mtc.Mrc100Log(stub, mrc100Key, TokenID, logger, log, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "mrc100get":
		if len(args) < 1 {
			return shim.Error("1000,mrc100get operation must include four arguments : mrc100Key")
		}

		mrc100Key := args[0]

		if value, err = mtc.Mrc100get(stub, mrc100Key); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc030create":
		if len(args) < 15 {
			return shim.Error("1000,MRC030Create operation must include four arguments : Creator,  mrc030id, Title, Description, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType, URL, Question, SignNeed, sign, tkey")
		}
		Creator := args[0]
		mrc030id := args[1]
		Title := args[2]
		Description := args[3]
		StartDate := args[4]
		EndDate := args[5]
		Reward := args[6]
		RewardToken := args[7]
		MaxRewardRecipient := args[8]
		RewardType := args[9]
		URL := args[10]
		Question := args[11]
		SignNeed := args[12]
		sign := args[13]
		tkey := args[14]
		if err = mtc.MRC030Create(stub, mrc030id, Creator, Title, Description, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType, URL, Question, SignNeed, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "mrc030join":
		if len(args) < 5 {
			return shim.Error("1000,MRC030Create operation must include four arguments : mrc030id, Voter, Answer, sign, voteCreatorSign")
		}
		mrc030id := args[0]
		Voter := args[1]
		Answer := args[2]
		voteCreatorSign := args[3]
		sign := args[4]
		if err = mtc.MRC030Join(stub, mrc030id, Voter, Answer, voteCreatorSign, sign, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "mrc030get":
		if len(args) < 1 {
			return shim.Error("1000,MRC030Get operation must include four arguments : mrc030id")
		}
		mrc030id := args[0]

		if _, err = mtc.getMRC030(stub, mrc030id); err != nil {
			return shim.Error(err.Error())
		}

		valuet, err := stub.GetState(mrc030id)
		if err != nil {
			return shim.Error(err.Error())
		}

		if valuet == nil {
			return shim.Error("1000,Key not exist")
		}

		return shim.Success([]byte(valuet))

	case "mrc030finish":
		if len(args) < 1 {
			return shim.Error("1000,MRC030Finish operation must include four arguments : mrc030id")
		}
		mrc030id := args[0]
		if err = mtc.MRC030Finish(stub, mrc030id, args); err != nil {
			return shim.Error(err.Error())
		}
		break

	case "mrc031get":
		if len(args) < 1 {
			return shim.Error("1000,MRC031 operation must include four arguments : mrc031id")
		}
		mrc031id := args[0]
		if _, err = mtc.getMRC031(stub, mrc031id); err != nil {
			return shim.Error(err.Error())
		}

		valuet, err := stub.GetState(mrc031id)
		if err != nil {
			return shim.Error(err.Error())
		}

		if valuet == nil {
			return shim.Error("1000,Key not exist")
		}

		return shim.Success([]byte(valuet))
	default:
		return shim.Error(fmt.Sprintf("Unsupported operation [%s]", function))
	}

	return shim.Success(nil)
}

func main() {
	err := shim.Start(new(MetaCoin))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s\n", err)
	}
}
