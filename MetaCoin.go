package main

/*
version 1.6     2020-11-06
version 2.0		2020-11-07
				cleanup!
				split module
				tkey => nonce
				address generate built-in


version 2.1		2021-02-14

version 2.2     MRC 400, 401, 402 - NFT

vertion 2.3	MRC 010(Token DEX)

*/
import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/peer"

	"inblock/metacoin"
	"inblock/metacoin/util"
)

type serverConfig struct {
	CCID    string
	Address string
}

// MetacoinChainCode dummy struct for init
type MetacoinChainCode struct {
}

// Invoke has two functions
// put - takes two arguments, a key and value, and stores them in the state
// remove - takes one argument, a key, and removes if from the state
func (t *MetacoinChainCode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	var tkey string
	var value string
	var err error
	var bytes []byte

	function, args := stub.GetFunctionAndParameters()
	for idx, arg := range args {
		args[idx] = strings.TrimSpace(arg)
	}

	switch function {

	// Function to quickly fill blocks to avoid collisions
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

	// Simple GET funhction
	case "get":
		if len(args) < 1 {
			return shim.Error("1000,get operation must include one arguments, address")
		}

		valuet, err := stub.GetState(args[0])
		if err != nil {
			return shim.Error(err.Error())
		}

		if valuet == nil {
			return shim.Error("1000,Key not exist")
		}

		return shim.Success([]byte(valuet))

	case "newwallet":
		if len(args) < 2 {
			return shim.Error("1000,newwallet operation must include five arguments : publicKey, addinfo")
		}

		publicKey := args[0]
		addinfo := args[1]

		// base.go
		value, err = metacoin.NewWallet(stub, publicKey, addinfo)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "getNonce":
		if len(args) < 1 {
			return shim.Error("1000,getNonce operation must include four arguments : address")
		}
		address := args[0]
		if value, err = metacoin.GetNonce(stub, address); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "balanceOf":
		if len(args) < 1 {
			return shim.Error("1000,balanceOf operation must include one argument : address")
		}
		address := args[0]
		if !util.IsAddress(address) {
			return shim.Error("Invalid address format")
		}

		// base.go
		value, err = metacoin.BalanceOf(stub, address)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

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

		// base.go
		err = metacoin.Transfer(stub, fromAddr, toAddr, amount, tokenID, unlockdate, sign, tkey, args)
		if err != nil {
			return shim.Error(err.Error())
		}

	case "signcheck":
		if len(args) < 3 {
			return shim.Error("1000,signcheck operation must include four arguments : address, data, sign")
		}

		address := args[0]
		data := args[1]
		sign := args[2]

		// base.go
		err = metacoin.SignCheck(stub, address, data, sign)
		if err != nil {
			return shim.Error(err.Error())
		}

	case "multitransfer":
		if len(args) < 5 {
			return shim.Error("1000,multitransfer operation must include four arguments : fromAddr, transferlist, tokenID, signature, tkey")
		}

		fromAddr := args[0]
		transferlist := args[1]
		tokenID := args[2]
		sign := args[3]
		tkey = args[4]

		// base.go
		err = metacoin.MultiTransfer(stub, fromAddr, transferlist, tokenID, sign, tkey, args)
		if err != nil {
			return shim.Error(err.Error())
		}

	case "tokenRegister":
		if len(args) < 3 {
			return shim.Error("1000,tokenRegister must include one arguments : tokeninfo, sign, tkey")
		}
		data := args[0]
		sign := args[1]
		tkey := args[2]

		// token.go
		value, err = metacoin.TokenRegister(stub, data, sign, tkey)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

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
		if err = metacoin.TokenUpdate(stub, TokenID, url, info, image, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "tokenIncrease":
		if err = metacoin.TokenIncrease(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "tokenBurning":
		if err = metacoin.TokenBurning(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010sell":
		if err = metacoin.Mrc010Sell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010unsell":
		if err = metacoin.Mrc010UnSell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010buy":
		if err = metacoin.Mrc010Buy(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010reqsell":
		if err = metacoin.Mrc010ReqSell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010unreqsell":
		if err = metacoin.Mrc010UnReqSell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010acceptreqsell":
		if err = metacoin.Mrc010AcceptReqSell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010auction":
		if err = metacoin.Mrc010Auction(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010unauction":
		if err = metacoin.Mrc010UnAuction(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010bid":
		if err = metacoin.Mrc010AuctionBid(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc010auctionfinish":
		if err = metacoin.Mrc010AuctionFinish(stub, args); err != nil {
			return shim.Error(err.Error())
		}

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

		if value, err = metacoin.Mrc020set(stub, owner, algorithm, data, publickey, opendate, referencekey, sign, tkey); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc020get":
		if len(args) < 1 {
			return shim.Error("1000,mrc20get operation must include four arguments : mrc020Key")
		}

		mrc020Key := args[0]

		if value, err = metacoin.Mrc020get(stub, mrc020Key); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc030create":
		if len(args) < 15 {
			return shim.Error("1000,MRC030Create operation must include four arguments : Creator, mrc030id, Title, Description, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType, URL, Question, SignNeed, sign, tkey")
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
		if err = metacoin.MRC030Create(stub, mrc030id, Creator, Title, Description, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType, URL, Question, SignNeed, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc030join":
		if len(args) < 5 {
			return shim.Error("1000,MRC030Create operation must include four arguments : mrc030id, Voter, Answer, sign, voteCreatorSign")
		}
		mrc030id := args[0]
		Voter := args[1]
		Answer := args[2]
		voteCreatorSign := args[3]
		sign := args[4]
		if err = metacoin.MRC030Join(stub, mrc030id, Voter, Answer, voteCreatorSign, sign, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc030get":
		if len(args) < 1 {
			return shim.Error("1000,MRC030Get operation must include four arguments : mrc030id")
		}
		mrc030id := args[0]

		if _, err = metacoin.Mrc030get(stub, mrc030id); err != nil {
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
		if err = metacoin.MRC030Finish(stub, mrc030id, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc031get":
		if len(args) < 1 {
			return shim.Error("1000,MRC031 operation must include four arguments : mrc031id")
		}
		mrc031id := args[0]
		if _, err = metacoin.Mrc031get(stub, mrc031id); err != nil {
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
		if err = metacoin.Mrc100Payment(stub, to, TokenID, tag, userlist, gameid, gamememo, args); err != nil {
			return shim.Error(err.Error())
		}

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
		if err = metacoin.Mrc100Reward(stub, from, TokenID, userlist, gameid, gamememo, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

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
		if value, err = metacoin.Mrc100Log(stub, mrc100Key, TokenID, logger, log, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc100get":
		if len(args) < 1 {
			return shim.Error("1000,mrc100get operation must include four arguments : mrc100Key")
		}

		mrc100Key := args[0]
		if value, err = metacoin.Mrc100get(stub, mrc100Key); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success([]byte(value))

	case "mrc400get":
		if len(args) < 1 {
			return shim.Error("1000,mrc400get operation must include four arguments : mrc400Key")
		}
		if _, bytes, err = metacoin.GetMRC400(stub, args[0]); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(bytes)

	case "mrc400create":
		if len(args) < 12 {
			return shim.Error("1000,mrc400create operation must include four arguments : owner, name, url, imageurl, allowtoken, itemurl, itemimageurl, category, description, data, sign, tkey, args")
		}
		owner := args[0]
		name := args[1]
		url := args[2]
		imageurl := args[3]
		allowtoken := args[4]
		itemurl := args[5]
		itemimageurl := args[6]
		category := args[7]
		description := args[8]
		data := args[9]
		sign := args[10]
		tkey := args[11]

		if err = metacoin.Mrc400Create(stub, owner, name, url, imageurl, allowtoken, itemurl, itemimageurl, category, description, data, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc400update":
		if len(args) < 12 {
			return shim.Error("1000,mrc400update operation must include four arguments : mrc400id, name, url, imageurl, allowtoken, itemurl, itemimageurl, category, description, data, sign, tkey")
		}
		mrc400id := args[0]
		name := args[1]
		url := args[2]
		imageurl := args[3]
		allowtoken := args[4]
		itemurl := args[5]
		itemimageurl := args[6]
		category := args[7]
		description := args[8]
		data := args[9]
		sign := args[10]
		tkey := args[11]
		if err = metacoin.Mrc400Update(stub, mrc400id, name, url, imageurl, allowtoken, itemurl, itemimageurl, category, description, data, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401get":
		if len(args) < 1 {
			return shim.Error("1000,mrc401get operation must include four arguments : mrc401Key")
		}

		if _, bytes, err = metacoin.GetMRC401(stub, args[0]); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(bytes)

	case "mrc401create":
		if len(args) < 4 {
			return shim.Error("1000,mrc401create operation must include four arguments : mrc400id, itemData, sign, tkey")
		}
		mrc400id := args[0]
		itemData := args[1]
		sign := args[2]
		tkey := args[3]

		if err = metacoin.Mrc401Create(stub, mrc400id, itemData, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401update":
		if len(args) < 4 {
			return shim.Error("1000,mrc401update operation must include four arguments : mrc400id, itemData, sign, tkey")
		}
		mrc400id := args[0]
		itemData := args[1]
		sign := args[2]
		tkey := args[3]

		if err = metacoin.Mrc401Update(stub, mrc400id, itemData, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401transfer":
		if len(args) < 5 {
			return shim.Error("1000,mrc401transfer operation must include four arguments : mrc401id, fromAddr, toAddr, sign, tkey")
		}
		mrc401id := args[0]
		fromAddr := args[1]
		toAddr := args[2]
		sign := args[3]
		tkey := args[4]

		if err = metacoin.Mrc401Transfer(stub, mrc401id, fromAddr, toAddr, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401sell":
		if len(args) < 5 {
			return shim.Error("1000,mrc401sell operation must include four arguments : seller, mrc400id, itemData, sign, tkey")
		}
		seller := args[0]
		mrc400id := args[1]
		itemData := args[2]
		sign := args[3]
		tkey := args[4]

		if err = metacoin.Mrc401Sell(stub, seller, mrc400id, itemData, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401unsell":
		if len(args) < 5 {
			return shim.Error("1000,mrc401unsell operation must include four arguments : seller, mrc400id, itemData, sign, tkey")
		}
		seller := args[0]
		mrc400id := args[1]
		itemData := args[2]
		sign := args[3]
		tkey := args[4]

		if err = metacoin.Mrc401UnSell(stub, seller, mrc400id, itemData, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401buy":
		if len(args) < 4 {
			return shim.Error("1000,mrc401buy operation must include four arguments : buyer, mrc401id, sign, tkey")
		}
		buyer := args[0]
		mrc401id := args[1]
		sign := args[2]
		tkey := args[3]

		if err = metacoin.Mrc401Buy(stub, buyer, mrc401id, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401melt":
		if len(args) < 3 {
			return shim.Error("1000,mrc401melt operation must include four arguments : mrc401id, sign, tkey")
		}
		mrc401id := args[0]
		sign := args[1]
		tkey := args[2]

		if err = metacoin.Mrc401Melt(stub, mrc401id, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401auction":
		if len(args) < 5 {
			return shim.Error("1000,mrc401auction operation must include four arguments : seller, mrc400id, itemData, sign, tkey")
		}
		seller := args[0]
		mrc400id := args[1]
		itemData := args[2]
		sign := args[3]
		tkey := args[4]

		if err = metacoin.Mrc401Auction(stub, seller, mrc400id, itemData, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401unauction":
		if len(args) < 5 {
			return shim.Error("1000,mrc401unauction operation must include four arguments : seller, mrc400id, itemData, sign, tkey")
		}
		seller := args[0]
		mrc400id := args[1]
		itemData := args[2]
		sign := args[3]
		tkey := args[4]

		if err = metacoin.Mrc401UnAuction(stub, seller, mrc400id, itemData, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401bid":
		if len(args) < 6 {
			return shim.Error("1000,mrc401bid operation must include four arguments : buyer, mrc401id, amount, token, sign, tkey")
		}
		buyer := args[0]
		mrc401id := args[1]
		amount := args[2]
		token := args[3]
		sign := args[4]
		tkey := args[5]
		if err = metacoin.Mrc401AuctionBid(stub, buyer, mrc401id, amount, token, sign, tkey, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc401auctionfinish":
		if len(args) < 1 {
			return shim.Error("1000,mrc401auctionfinish operation must include four arguments : mrc401id")
		}
		mrc401id := args[0]

		if err = metacoin.Mrc401AuctionFinish(stub, mrc401id); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402create":
		if err = metacoin.Mrc402Create(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402get":
		if len(args) < 1 {
			return shim.Error("1000,mrc402get operation must include four arguments : MRC402ID")
		}
		if _, bytes, err = metacoin.GetMRC402(stub, args[0]); err != nil {
			return shim.Error(err.Error())
		}
		return shim.Success(bytes)

	case "mrc402update":
		if err = metacoin.Mrc402Update(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402transfer":
		if err = metacoin.Mrc402Transfer(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402mint":
		if err = metacoin.Mrc402Mint(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402burn":
		if err = metacoin.Mrc402Burn(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402melt":
		if err = metacoin.Mrc402Melt(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402sell":
		if err = metacoin.Mrc402Sell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402unsell":
		if err = metacoin.Mrc402UnSell(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402buy":
		if err = metacoin.Mrc402Buy(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402auction":
		if err = metacoin.Mrc402Auction(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402unauction":
		if err = metacoin.Mrc402UnAuction(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402bid":
		if err = metacoin.Mrc402AuctionBid(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	case "mrc402auctionfinish":
		if err = metacoin.Mrc402AuctionFinish(stub, args); err != nil {
			return shim.Error(err.Error())
		}

	default:
		return shim.Error(fmt.Sprintf("Unsupported operation [%s]", function))
	}

	return shim.Success(nil)
}

// Init function
func (t *MetacoinChainCode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func main() {
	// See chaincode.env.example
	config := serverConfig{
		CCID:    os.Getenv("CHAINCODE_ID"),
		Address: os.Getenv("CHAINCODE_SERVER_ADDRESS"),
	}

	server := &shim.ChaincodeServer{
		CCID:    config.CCID,
		Address: config.Address,
		CC:      new(MetacoinChainCode),
		TLSProps: shim.TLSProperties{
			Disabled: true,
		},
	}

	if err := server.Start(); err != nil {
		log.Panicf("error starting metacoin chaincode: %s", err)
	}
}
