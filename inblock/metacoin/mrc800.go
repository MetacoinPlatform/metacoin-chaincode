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

/* MRC800 - point
There is no issue quantity.
MRC800 creators can give MRC800 Tokens to users without restrictions.
MRC800 creators can bring MRC800 Tokens to users without restrictions.
When the MRC800 token is in a transferable state, the owner of the MRC800 can transfer it to other users.
*/

// Mrc800Create create MRC800 Item
func Mrc800Create(stub shim.ChaincodeStubInterface, owner, name, url, imageurl, transferable, description, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.MetaWallet
	var mrc800id string
	var MRC800ProjectData mtc.MRC800
	var argdat []byte

	MRC800ProjectData = mtc.MRC800{
		CreateDate: time.Now().Unix(),
	}

	if err = util.DataAssign(owner, &MRC800ProjectData.Owner, "address", 40, 40, false); err != nil {
		return errors.New("3005,Data must be 1 to 4096 characters long")
	}
	if err = util.DataAssign(name, &MRC800ProjectData.Name, "string", 1, 128, false); err != nil {
		return errors.New("3005,Name must be 1 to 128 characters long")
	}
	if err = util.DataAssign(url, &MRC800ProjectData.URL, "string", 1, 1024, false); err != nil {
		return errors.New("3005,Url must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(imageurl, &MRC800ProjectData.ImageURL, "url", 1, 256, false); err != nil {
		return errors.New("3005,ImageURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(transferable, &MRC800ProjectData.Transferable, "bool", 1, 1, false); err != nil {
		return errors.New("3005,transferable must be '1' or '0'")
	}
	if err = util.DataAssign(description, &MRC800ProjectData.Description, "string", 1, 4096, false); err != nil {
		return errors.New("3005,Description must be 1 to 4096 characters long")
	}

	if ownerWallet, err = GetAddressInfo(stub, owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{owner, name, url, imageurl, tkey}, "|"),
		signature); err != nil {
		return err
	}

	var isSuccess = false
	temp := util.GenerateKey("MRC800_", []string{owner, name, url, imageurl, tkey})
	for i := 0; i < 10; i++ {
		mrc800id = fmt.Sprintf("%39s%1d", temp, i)
		argdat, err = stub.GetState(mrc800id)
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

	MRC800ProjectData.JobType = "mrc800_create"
	MRC800ProjectData.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal([]string{mrc800id, owner, name, url, imageurl, description, signature, tkey}); err == nil {
		MRC800ProjectData.JobArgs = string(argdat)
	}
	if argdat, err = json.Marshal(MRC800ProjectData); err != nil {
		return errors.New("3209,Invalid mrc800 data format")
	}
	if err := stub.PutState(mrc800id, argdat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error() + mrc800id)
	}

	if err = SetAddressInfo(stub, owner, ownerWallet, "mrc800create", []string{mrc800id, owner, name, url, imageurl, description, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc800Update for MTC110 token update.
func Mrc800Update(stub shim.ChaincodeStubInterface, mrc800id, name, url, imageurl, transferable, description, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.MetaWallet
	var MRC800ProjectData mtc.MRC800
	var argdat []byte
	var mrc800 string

	if mrc800, err = Mrc800get(stub, mrc800id); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(mrc800), &MRC800ProjectData); err != nil {
		return errors.New("6205,MRC800 [" + mrc800id + "] is in the wrong data")
	}

	if err = util.DataAssign(name, &MRC800ProjectData.Name, "string", 1, 128, true); err != nil {
		return errors.New("3005,Name must be 1 to 128 characters long")
	}
	if err = util.DataAssign(url, &MRC800ProjectData.URL, "url", 1, 255, true); err != nil {
		return errors.New("3005,Url must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(imageurl, &MRC800ProjectData.ImageURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,ImageURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(transferable, &MRC800ProjectData.Transferable, "bool", 1, 1, false); err != nil {
		return errors.New("3005,transferable must be '1' or '0'")
	}
	if err = util.DataAssign(description, &MRC800ProjectData.Description, "string", 1, 4096, true); err != nil {
		return errors.New("3005,Description must be 1 to 4096 characters long")
	}

	if ownerWallet, err = GetAddressInfo(stub, MRC800ProjectData.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{mrc800id, name, url, imageurl, description, tkey}, "|"),
		signature); err != nil {
		return err
	}

	MRC800ProjectData.JobType = "mrc800_update"
	MRC800ProjectData.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal([]string{mrc800id, MRC800ProjectData.Owner, name, url, imageurl, description, signature, tkey}); err == nil {
		MRC800ProjectData.JobArgs = string(argdat)
	}

	if argdat, err = json.Marshal(MRC800ProjectData); err != nil {
		return errors.New("3209,Invalid address data format")
	}
	if err := stub.PutState(mrc800id, argdat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error() + mrc800id)
	}

	if err = SetAddressInfo(stub, MRC800ProjectData.Owner, ownerWallet, "mrc800update", []string{mrc800id, MRC800ProjectData.Owner, name, url, imageurl, description, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc800get - return MRC800 token
func Mrc800get(stub shim.ChaincodeStubInterface, mrc800id string) (string, error) {
	var dat []byte
	var err error

	if strings.Index(mrc800id, "MRC800_") != 0 || len(mrc800id) != 40 {
		return "", errors.New("6102,invalid MRC800 data address")
	}

	dat, err = stub.GetState(mrc800id)
	if err != nil {
		return "", errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return "", errors.New("6004,MRC800 [" + mrc800id + "] not exist")
	}
	return string(dat), nil
}

func Mrc800give(stub shim.ChaincodeStubInterface, mrc800id, toAddr, amount, memo, signature, tkey string) error {
	var err error
	var tokenOwnerWallet, toAddrWallet mtc.MetaWallet
	var mrc800Token mtc.MRC800
	var mrc800 string
	var addAmount, currentAmount decimal.Decimal

	// get project(mrc800)
	if mrc800, err = Mrc800get(stub, mrc800id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(mrc800), &mrc800Token); err != nil {
		return errors.New("3004,MRC800 [" + mrc800id + "] is in the wrong data")
	}
	if tokenOwnerWallet, err = GetAddressInfo(stub, mrc800Token.Owner); err != nil {
		return err
	}
	if err = NonceCheck(&tokenOwnerWallet, tkey,
		strings.Join([]string{mrc800id, toAddr, amount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if toAddrWallet, err = GetAddressInfo(stub, toAddr); err != nil {
		return err
	}

	if toAddrWallet.MRC800 == nil {
		toAddrWallet.MRC800 = make(map[string]string, 0)
	}
	if addAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1101," + amount + " is not positive integer")
	}

	if _, exists := toAddrWallet.MRC800[mrc800id]; exists {
		toAddrWallet.MRC800[mrc800id] = addAmount.String()
	} else {
		currentAmount, _ = decimal.NewFromString(toAddrWallet.MRC800[mrc800id])
		toAddrWallet.MRC800[mrc800id] = currentAmount.Add(addAmount).String()
	}
	err = SetAddressInfo(stub, toAddr, toAddrWallet, "receive_mrc800give", []string{mrc800Token.Owner, toAddr, amount, mrc800id, signature, "0", "", memo, tkey})
	return err
}

func Mrc800take(stub shim.ChaincodeStubInterface, mrc800id, fromAddr, amount, memo, signature, tkey string) error {
	var err error
	var tokenOwnerWallet, fromAddrWallet mtc.MetaWallet
	var mrc800Token mtc.MRC800
	var mrc800 string
	var subAmount, currentAmount decimal.Decimal

	// get project(mrc800)
	if mrc800, err = Mrc800get(stub, mrc800id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(mrc800), &mrc800Token); err != nil {
		return errors.New("3004,MRC800 [" + mrc800id + "] is in the wrong data")
	}
	if tokenOwnerWallet, err = GetAddressInfo(stub, mrc800Token.Owner); err != nil {
		return err
	}
	if err = NonceCheck(&tokenOwnerWallet, tkey,
		strings.Join([]string{mrc800id, fromAddr, amount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if fromAddrWallet, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}

	if fromAddrWallet.MRC800 == nil {
		fromAddrWallet.MRC800 = make(map[string]string, 0)
	}
	if subAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1101," + amount + " is not positive integer")
	}

	if _, exists := fromAddrWallet.MRC800[mrc800id]; exists {
		return errors.New("5000,Not enough balance")
	}
	currentAmount, _ = decimal.NewFromString(fromAddrWallet.MRC800[mrc800id])
	if currentAmount.Cmp(subAmount) < 0 {
		return errors.New("5000,Not enough balance")
	}

	fromAddrWallet.MRC800[mrc800id] = currentAmount.Sub(subAmount).String()
	err = SetAddressInfo(stub, fromAddr, fromAddrWallet, "transfer_mrc800take", []string{fromAddr, mrc800Token.Owner, amount, mrc800id, signature, "0", "", memo, tkey})
	return err
}

func Mrc800transfer(stub shim.ChaincodeStubInterface, fromAddr, toAddr, mrc800id, amount, memo, signature, tkey string) error {
	var err error
	var fromAddrWallet, toAddrWallet mtc.MetaWallet
	var mrc800Token mtc.MRC800
	var mrc800 string
	var transferAmount, currentAmount decimal.Decimal

	// get project(mrc800)
	if mrc800, err = Mrc800get(stub, mrc800id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(mrc800), &mrc800Token); err != nil {
		return errors.New("3004,MRC800 [" + mrc800id + "] is in the wrong data")
	}
	if mrc800Token.Transferable == "0" {
		return errors.New("3004,MRC800 [" + mrc800id + "] is not transferable")
	}

	if fromAddrWallet, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}
	if err = NonceCheck(&fromAddrWallet, tkey,
		strings.Join([]string{mrc800id, fromAddr, toAddr, mrc800id, amount, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if fromAddrWallet.MRC800 == nil {
		fromAddrWallet.MRC800 = make(map[string]string, 0)
	}
	if transferAmount, err = util.ParsePositive(amount); err != nil {
		return errors.New("1101," + amount + " is not positive integer")
	}

	if _, exists := fromAddrWallet.MRC800[mrc800id]; exists {
		return errors.New("5000,Not enough balance")
	}
	currentAmount, _ = decimal.NewFromString(fromAddrWallet.MRC800[mrc800id])
	if currentAmount.Cmp(transferAmount) < 0 {
		return errors.New("5000,Not enough balance")
	}

	fromAddrWallet.MRC800[mrc800id] = currentAmount.Sub(transferAmount).String()
	if err = SetAddressInfo(stub, fromAddr, fromAddrWallet, "transfer_mrc800", []string{fromAddr, toAddr, amount, mrc800id, signature, "0", "", memo, tkey}); err != nil {
		return err
	}

	if toAddrWallet, err = GetAddressInfo(stub, toAddr); err != nil {
		return err
	}

	if _, exists := toAddrWallet.MRC800[mrc800id]; exists {
		toAddrWallet.MRC800[mrc800id] = transferAmount.String()
	} else {
		currentAmount, _ = decimal.NewFromString(toAddrWallet.MRC800[mrc800id])
		toAddrWallet.MRC800[mrc800id] = currentAmount.Add(transferAmount).String()
	}
	err = SetAddressInfo(stub, toAddr, toAddrWallet, "receive_mrc800", []string{fromAddr, toAddr, amount, mrc800id, signature, "0", "", memo, tkey})

	return err
}
