package metacoin

import (
	"errors"
	"strings"
	"time"

	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"

	"inblock/metacoin/mtc"
)

// Mrc100Payment : Game payment
func Mrc100Payment(stub shim.ChaincodeStubInterface, to, TokenID, tag, userlist, gameid, gamememo string, args []string) error {
	var err error
	var ownerData, playerData mtc.MetaWallet
	var playerList []mtc.MRC100Payment

	if ownerData, err = GetAddressInfo(stub, to); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(userlist), &playerList); err != nil {
		return errors.New("4209,Invalid UserLIst data")
	}
	if len(playerList) < 1 {
		return errors.New("2007,Playerlist need more than one")
	}
	if len(playerList) > 32 {
		return errors.New("2007,Playerlist should be less than 32")
	}

	for _, elements := range playerList {
		if playerData, err = GetAddressInfo(stub, elements.Address); err != nil {
			return err
		}

		if err = NonceCheck(&playerData, elements.TKey,
			strings.Join([]string{elements.Address, to, TokenID, elements.Amount, elements.TKey}, "|"),
			elements.Signature); err != nil {
			return err
		}

		if elements.Amount == "" {
			return errors.New("1107,The amount must be an integer")
		}

		if elements.Amount != "0" {
			if err = MoveToken(stub, &playerData, &ownerData, TokenID, elements.Amount, 0); err != nil {
				return err
			}
		}

		if err = SetAddressInfo(stub, elements.Address, playerData, "mrc100payment",
			[]string{elements.Address, to, elements.Amount, TokenID, elements.Signature, "", tag, elements.Memo, elements.TKey}); err != nil {
			return err
		}
	}

	if err = SetAddressInfo(stub, to, ownerData, "mrc100paymentrecv", args); err != nil {
		return err
	}
	return nil
}

// Mrc100Reward : Game reward
func Mrc100Reward(stub shim.ChaincodeStubInterface, from, TokenID, userlist, gameid, gamememo, signature, tkey string, args []string) error {
	var err error
	var ownerData, playerData mtc.MetaWallet
	var playerList []mtc.MRC100Reward
	var checkList []string

	if ownerData, err = GetAddressInfo(stub, from); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(userlist), &playerList); err != nil {
		return errors.New("4209,Invalid UserLIst data")
	}
	if len(playerList) == 0 {
		return errors.New("2007,Playerlist need more than one")
	}
	if len(playerList) > 32 {
		return errors.New("2007,Playerlist should be less than 32")
	}

	checkList = append(checkList, TokenID)

	for _, elements := range playerList {
		if elements.Amount == "" {
			return errors.New("1107,The amount must be an integer")
		}
		checkList = append(checkList, elements.Address, elements.Amount, elements.Tag)
	}
	checkList = append(checkList, tkey)

	if err = NonceCheck(&ownerData, tkey,
		strings.Join(checkList, "|"),
		signature); err != nil {
		return err
	}

	for _, elements := range playerList {
		if playerData, err = GetAddressInfo(stub, elements.Address); err != nil {
			return err
		}
		if elements.Amount != "" && elements.Amount != "0" {
			if err = MoveToken(stub, &ownerData, &playerData, TokenID, elements.Amount, 0); err != nil {
				return err
			}
		}

		if err = SetAddressInfo(stub, elements.Address, playerData, "mrc030reward",
			[]string{from, elements.Address, elements.Amount, TokenID, signature, "", elements.Tag, elements.Memo, ""}); err != nil {
			return err
		}
	}

	if err = SetAddressInfo(stub, from, ownerData, "mrc030payment", args); err != nil {
		return err
	}
	return nil
}

// Mrc100Log Game log
func Mrc100Log(stub shim.ChaincodeStubInterface, key, token, logger, log, signature, tkey string, args []string) (string, error) {
	var err error
	var tk mtc.Token
	var ownerData mtc.MetaWallet
	var mrcLog mtc.MRC100Log
	var dat []byte

	if tk, _, err = GetToken(stub, token); err != nil {
		return "", err
	}

	if tk.Owner != logger {
		if _, exists := tk.Logger[logger]; !exists {
			return "", errors.New("6030,you do not have permission to log this token")
		}
	}
	if ownerData, err = GetAddressInfo(stub, logger); err != nil {
		return "", err
	}

	if tk.Type != "100" && tk.Type != "101" {
		return "", errors.New("6032,This token cannot log")
	}

	if err = NonceCheck(&ownerData, tkey,
		strings.Join([]string{token, logger, log, tkey}, "|"),
		signature); err != nil {
		return "", err
	}

	mrcLog = mtc.MRC100Log{Regdate: time.Now().Unix(),
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

	return key, nil
}

// Mrc100get - MRC-100 Protocol Add
func Mrc100get(stub shim.ChaincodeStubInterface, mrc100Key string) (string, error) {
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
