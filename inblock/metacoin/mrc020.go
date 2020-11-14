package metacoin

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

// =========================================
// mrc020.go
// =========================================

// Mrc020set - MRC-020 Protocol set
func Mrc020set(stub shim.ChaincodeStubInterface, owner, algorithm, data, publickey, opendate, referencekey, signature, tkey string) (string, error) {
	var dat []byte
	var mrc020Key string
	var err error
	var mrc020 mtc.MRC020
	var currNo64 int64

	if currNo64, err = util.Strtoint64(opendate); err != nil {
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

	var mwOwner mtc.MetaWallet
	if mwOwner, err = GetAddressInfo(stub, owner); err != nil {
		return "", err
	}

	if err = NonceCheck(&mwOwner, tkey,
		strings.Join([]string{owner, data, opendate, referencekey, tkey}, "|"),
		signature); err != nil {
		return "", err
	}

	if err = stub.PutState(mrc020Key, dat); err != nil {
		return "", err
	}
	return mrc020Key, nil
}

// Mrc020get - MRC-020 Protocol Add
func Mrc020get(stub shim.ChaincodeStubInterface, mrc020Key string) (string, error) {
	var dat []byte
	var err error
	var mrc020 mtc.MRC020

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
