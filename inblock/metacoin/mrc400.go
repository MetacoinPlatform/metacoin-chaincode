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

// logDataBuy multi payyment info
type logDataBuy struct {
	FromAddr string `json:"from_addr"`
	ToAddr   string `json:"to_addr"`
	Amount   string `json:"amount"`
	TokenID  string `json:"token"`
	PayType  string `json:"type"`
}

// Mrc400Create create MRC400 Item
func Mrc400Create(stub shim.ChaincodeStubInterface, owner, name, url, imageurl, allowtoken, category, description, itemurl, itemimageurl, data, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.MetaWallet
	var mrc400id string
	var MRC400ProjectData mtc.MRC400
	var argdat []byte

	MRC400ProjectData = mtc.MRC400{
		CreateDate: time.Now().Unix(),
	}

	if err = util.DataAssign(owner, &MRC400ProjectData.Owner, "address", 40, 40, false); err != nil {
		return errors.New("3005,Data must be 1 to 4096 characters long")
	}
	if err = util.DataAssign(name, &MRC400ProjectData.Name, "string", 1, 128, false); err != nil {
		return errors.New("3005,Name must be 1 to 128 characters long")
	}
	if err = util.DataAssign(url, &MRC400ProjectData.URL, "string", 1, 1024, false); err != nil {
		return errors.New("3005,Url must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(imageurl, &MRC400ProjectData.ImageURL, "url", 1, 256, false); err != nil {
		return errors.New("3005,ImageURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(category, &MRC400ProjectData.Category, "string", 1, 64, false); err != nil {
		return errors.New("3005,Category must be 1 to 64 characters long")
	}
	if err = util.DataAssign(description, &MRC400ProjectData.Description, "string", 1, 4096, false); err != nil {
		return errors.New("3005,Description must be 1 to 4096 characters long")
	}
	if err = util.DataAssign(itemurl, &MRC400ProjectData.ItemURL, "url", 1, 256, false); err != nil {
		return errors.New("3005,ItemURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(itemimageurl, &MRC400ProjectData.ItemImageURL, "url", 1, 256, false); err != nil {
		return errors.New("3005,ItemImageURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(allowtoken, &MRC400ProjectData.AllowToken, "string", 1, 128, false); err != nil {
		return errors.New("3005,AllowToken must be 1 to 128 characters long")
	}

	// allow token error
	if _, _, err = GetToken(stub, allowtoken); err != nil {
		return errors.New("3005,Token id " + allowtoken + " error : " + err.Error())
	}

	if ownerWallet, err = GetAddressInfo(stub, owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{owner, name, url, imageurl, category, itemurl, itemimageurl, data, tkey}, "|"),
		signature); err != nil {
		return err
	}

	var isSuccess = false
	temp := util.GenerateKey("MRC400_", []string{owner, name, url, imageurl, category, itemurl, itemimageurl, data, tkey})
	for i := 0; i < 10; i++ {
		mrc400id = fmt.Sprintf("%39s%1d", temp, i)
		argdat, err = stub.GetState(mrc400id)
		if err != nil {
			fmt.Printf("setAddressInfo stub.GetState(key) [%s] Error %s\n", mrc400id, err)
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if argdat != nil { // key already exists
			continue
		} else {
			isSuccess = true
			break
		}
	}

	if isSuccess == false {
		return errors.New("3005,Data generate error, retry again")
	}

	MRC400ProjectData.JobType = "mrc400_create"
	MRC400ProjectData.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal([]string{mrc400id, owner, name, url, imageurl, allowtoken, category, description, itemurl, itemimageurl, data, signature, tkey}); err == nil {
		MRC400ProjectData.JobArgs = string(argdat)
	}
	if argdat, err = json.Marshal(MRC400ProjectData); err != nil {
		return errors.New("3209,Invalid mrc400 data format")
	}
	if err := stub.PutState(mrc400id, argdat); err != nil {
		fmt.Printf("setAddressInfo stub.PutState(key, dat) [%s] Error %s\n", mrc400id, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error() + mrc400id)
	}

	if err = SetAddressInfo(stub, owner, ownerWallet, "mrc400create", []string{mrc400id, owner, name, url, imageurl, allowtoken, category, description, itemurl, itemimageurl, data, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc400Update for MTC110 token update.
func Mrc400Update(stub shim.ChaincodeStubInterface, mrc400id, name, url, imageurl, allowtoken, category, description, itemurl, itemimageurl, data, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.MetaWallet
	var MRC400ProjectData mtc.MRC400
	var argdat []byte
	var mrc400 string

	if mrc400, err = Mrc400get(stub, mrc400id); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(mrc400), &MRC400ProjectData); err != nil {
		return errors.New("6205,MRC400 [" + mrc400id + "] is in the wrong data")
	}

	if err = util.DataAssign(name, &MRC400ProjectData.Name, "string", 1, 128, true); err != nil {
		return errors.New("3005,Name must be 1 to 128 characters long")
	}
	if err = util.DataAssign(url, &MRC400ProjectData.URL, "url", 1, 255, true); err != nil {
		return errors.New("3005,Url must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(imageurl, &MRC400ProjectData.ImageURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,ImageURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(category, &MRC400ProjectData.Category, "string", 1, 64, true); err != nil {
		return errors.New("3005,Category must be 1 to 64 characters long")
	}
	if err = util.DataAssign(description, &MRC400ProjectData.Description, "string", 1, 4096, true); err != nil {
		return errors.New("3005,Description must be 1 to 4096 characters long")
	}
	if err = util.DataAssign(itemurl, &MRC400ProjectData.ItemURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,ItemURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(itemimageurl, &MRC400ProjectData.ItemImageURL, "url", 1, 255, true); err != nil {
		return errors.New("3005,ItemImageURL must be 1 to 1024 characters long URL")
	}
	if err = util.DataAssign(data, &MRC400ProjectData.Data, "string", 1, 4096, true); err != nil {
		return errors.New("3005,Data must be 1 to 4096 characters long")
	}
	if err = util.DataAssign(allowtoken, &MRC400ProjectData.AllowToken, "string", 1, 128, false); err != nil {
		return errors.New("3005,AllowToken must be 1 to 128 characters long")
	}

	// allow token error
	if _, _, err = GetToken(stub, allowtoken); err != nil {
		return errors.New("3005,Token id " + allowtoken + " error : " + err.Error())
	}

	if ownerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
		return err
	}

	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{mrc400id, name, url, imageurl, category, description, itemurl, itemimageurl, data, tkey}, "|"),
		signature); err != nil {
		return err
	}

	MRC400ProjectData.JobType = "mrc400_update"
	MRC400ProjectData.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal([]string{mrc400id, MRC400ProjectData.Owner, name, url, imageurl, allowtoken, category, description, itemurl, itemimageurl, data, signature, tkey}); err == nil {
		MRC400ProjectData.JobArgs = string(argdat)
	}

	if argdat, err = json.Marshal(MRC400ProjectData); err != nil {
		fmt.Printf("setAddressInfo json.Marshal(mcData) [%s] Marshal error %s\n", mrc400id, err)
		return errors.New("3209,Invalid address data format")
	}
	if err := stub.PutState(mrc400id, argdat); err != nil {
		fmt.Printf("setAddressInfo stub.PutState(key, dat) [%s] Error %s\n", mrc400id, err)
		return errors.New("8600,Hyperledger internal error - " + err.Error() + mrc400id)
	}

	if err = SetAddressInfo(stub, MRC400ProjectData.Owner, ownerWallet, "mrc400update", []string{mrc400id, MRC400ProjectData.Owner, name, url, imageurl, allowtoken, category, description, itemurl, itemimageurl, data, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc400get - MRC-110 Protocol Add
func Mrc400get(stub shim.ChaincodeStubInterface, mrc400Key string) (string, error) {
	var dat []byte
	var err error

	if strings.Index(mrc400Key, "MRC400_") != 0 || len(mrc400Key) != 40 {
		return "", errors.New("6102,invalid MRC400 data address")
	}

	dat, err = stub.GetState(mrc400Key)
	if err != nil {
		return "", errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return "", errors.New("6004,MRC400 [" + mrc400Key + "] not exist")
	}
	return string(dat), nil
}

// Mrc401get - MRC-110 Protocol Add
func Mrc401get(stub shim.ChaincodeStubInterface, mrc401Key string) (string, error) {
	var dat []byte
	var err error

	if strings.Index(mrc401Key, "MRC400_") != 0 || len(mrc401Key) != 81 {
		fmt.Printf("Key : [%s], len : %d\n", mrc401Key, len(mrc401Key))
		return "", errors.New("6102,invalid MRC401 data address")
	}

	dat, err = stub.GetState(mrc401Key)
	if err != nil {
		return "", errors.New("8110,Hyperledger internal error - " + err.Error())
	}
	if dat == nil {
		return "", errors.New("6004,MRC401 [" + mrc401Key + "] not exist")
	}
	return string(dat), nil
}

func mrc401set(stub shim.ChaincodeStubInterface, mrc401Key string, MRC401ItemData mtc.MRC401, jobType string, jobArgs []string) error {
	var err error
	var argdat []byte

	if strings.Index(mrc401Key, "MRC400_") != 0 || len(mrc401Key) != 81 {
		return errors.New("6102,invalid MRC401 data address")
	}

	MRC401ItemData.JobType = jobType
	MRC401ItemData.JobDate = time.Now().Unix()
	if argdat, err = json.Marshal(jobArgs); err == nil {
		MRC401ItemData.JobArgs = string(argdat)
	}

	if argdat, err = json.Marshal(MRC401ItemData); err != nil {
		return errors.New("3209,Invalid MRC401ItemData data format")
	}

	if err := stub.PutState(mrc401Key, argdat); err != nil {
		fmt.Printf("Mrc401Create stub.PutState [%s] Error %s\n", mrc401Key, err.Error())
		return errors.New("8600,Mrc401Create stub.PutState [" + mrc401Key + "] Error " + err.Error())
	}
	return nil
}

// Mrc401Create MRC401 create
func Mrc401Create(stub shim.ChaincodeStubInterface, mrc400id, itemData, signature, tkey string, args []string) error {
	var err error
	var projectOwnerWallet mtc.MetaWallet
	var now int64
	var MRC400ProjectData mtc.MRC400
	var mrc400 string
	var createTotal map[string]decimal.Decimal
	var tempPrice decimal.Decimal
	var MRC401Job []mtc.MRC401job
	var MRC401ItemData mtc.MRC401
	var data []byte

	var logData []mtc.MRC401Sell

	// get project(mrc400)
	if mrc400, err = Mrc400get(stub, mrc400id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(mrc400), &MRC400ProjectData); err != nil {
		return errors.New("3004,MRC400 [" + mrc400id + "] is in the wrong data")
	}

	// get project owner
	if projectOwnerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
		return err
	}
	// sign check
	if err = NonceCheck(&projectOwnerWallet, tkey,
		strings.Join([]string{mrc400id, itemData, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(itemData), &MRC401Job); err != nil {
		return errors.New("6205,MRC401 Data is in the wrong data - " + err.Error())
	}
	if len(MRC401Job) > 100 {
		return errors.New("3002,There must be 100 or fewer create item")
	}

	createTotal = make(map[string]decimal.Decimal)
	now = time.Now().Unix()
	logData = make([]mtc.MRC401Sell, 0, 100)
	for index := range MRC401Job {

		// item check & save
		fmt.Printf("Key exists check [%s]\n", mrc400id+"_"+MRC401Job[index].ItemID)

		data, err = stub.GetState(mrc400id + "_" + MRC401Job[index].ItemID)
		if err != nil {
			fmt.Printf("Mrc401Create stub.GetState(key) [%s] Error %s\n", mrc400id+"_"+MRC401Job[index].ItemID, err)
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if data != nil {
			return errors.New("8600,Item ID " + MRC401Job[index].ItemID + " already exists in project " + mrc400id)
		}

		// init data
		MRC401ItemData = mtc.MRC401{
			MRC400: mrc400id,
			Owner:  MRC400ProjectData.Owner,
			// ItemURL,
			// ItemImageURL
			// GroupID
			CreateDate: now,
			// InititalReserve
			// InititalToken
			// MeltingFee
			// Transferable
			// SellFee
			// SellPrice
			// SellToken
			SellDate:    0,
			MeltingDate: 0,
			JobType:     "mrc401_create",
			JobArgs:     "",
			JobDate:     now,
		}

		// param check
		if err = util.DataAssign(MRC401Job[index].ItemID, &MRC401Job[index].ItemID, "id", 40, 40, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item ItemID error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].ItemURL, &MRC401ItemData.ItemURL, "url", 1, 255, true); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item ItemURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].ItemImageURL, &MRC401ItemData.ItemImageURL, "url", 1, 255, true); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item ItemImageURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].GroupID, &MRC401ItemData.GroupID, "string", 1, 40, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item GroupID error : " + err.Error())
		}

		if err = util.NumericDataCheck(MRC401Job[index].InititalReserve, &MRC401ItemData.InititalReserve, "0", "9999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item InititalReserve error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].InititalToken, &MRC401ItemData.InititalToken, "string", 1, 40, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item InititalToken error : " + err.Error())
		}

		if err = util.NumericDataCheck(MRC401Job[index].MeltingFee, &MRC401ItemData.MeltingFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item MeltingFee error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].Transferable, &MRC401ItemData.Transferable, "string", 1, 128, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item Transferable error : " + err.Error())
		}

		if MRC401ItemData.Transferable != "Permanent" && MRC401ItemData.Transferable != "Bound" && MRC401ItemData.Transferable != "Temprary" {
			return errors.New("3005," + util.GetOrdNumber(index) + " item Transferable value is Permanent, Bound, Temprary ")
		}

		if err = util.NumericDataCheck(MRC401Job[index].SellFee, &MRC401ItemData.SellFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellFee error : " + err.Error())
		}

		// Initital token check
		if MRC401ItemData.InititalToken != MRC400ProjectData.AllowToken && MRC401ItemData.InititalToken != "0" {
			if MRC400ProjectData.AllowToken != "0" {
				return errors.New("3005," + util.GetOrdNumber(index) + " item Token is must " + MRC400ProjectData.AllowToken + " or metacoin")
			}
			return errors.New("3005," + util.GetOrdNumber(index) + " item Token is must " + MRC400ProjectData.AllowToken)
		}

		if tempPrice, err = decimal.NewFromString(MRC401ItemData.InititalReserve); err != nil {
			return errors.New("3209," + util.GetOrdNumber(index) + " item Invalid InititalReserve")
		}
		if tempPrice.IsPositive() {
			createTotal[MRC401ItemData.InititalToken] = createTotal[MRC401ItemData.InititalToken].Add(tempPrice).Truncate(0)
		}

		if err = mrc401set(stub, mrc400id+"_"+MRC401Job[index].ItemID, MRC401ItemData, "mrc401_create", []string{mrc400id + "_" + MRC401Job[index].ItemID, MRC400ProjectData.Owner, MRC401ItemData.InititalReserve, MRC401ItemData.InititalToken, signature, tkey}); err != nil {
			return err
		}
		logData = append(logData, mtc.MRC401Sell{MRC401Job[index].ItemID, MRC401ItemData.InititalReserve, MRC401ItemData.InititalToken})
	}

	// subtract token for item initial price
	for token, totPrice := range createTotal {
		if totPrice.IsPositive() {
			if err = SubtractToken(stub, &projectOwnerWallet, token, totPrice.String()); err != nil {
				return err
			}
		}
	}

	// save create info
	// - for update balance
	// - for nonce update
	if err = SetAddressInfo(stub, MRC400ProjectData.Owner, projectOwnerWallet, "mrc401create", []string{mrc400id, MRC400ProjectData.Owner,
		util.JSONEncode(logData), signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401Update MRC401 update
func Mrc401Update(stub shim.ChaincodeStubInterface, mrc400id, itemData, signature, tkey string, args []string) error {
	var err error
	var projectOwnerWallet mtc.MetaWallet
	var MRC400ProjectData mtc.MRC400
	var mrc400 string
	var createTotal map[string]decimal.Decimal
	var MRC401Job []mtc.MRC401job
	var MRC401ItemData mtc.MRC401
	var data []byte
	var logData []string

	// get project(mrc400)
	if mrc400, err = Mrc400get(stub, mrc400id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(mrc400), &MRC400ProjectData); err != nil {
		return errors.New("3004,MRC400 [" + mrc400id + "] is in the wrong data")
	}

	// get project owner
	if projectOwnerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
		return err
	}
	// sign check
	if err = NonceCheck(&projectOwnerWallet, tkey,
		strings.Join([]string{mrc400id, itemData, tkey}, "|"),
		signature); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(itemData), &MRC401Job); err != nil {
		return errors.New("6205,MRC401 Data is in the wrong data - " + err.Error())
	}
	if len(MRC401Job) > 100 {
		return errors.New("3002,There must be 100 or fewer create item")
	}

	logData = make([]string, 0, 100)
	for index := range MRC401Job {
		data, err = stub.GetState(mrc400id + "_" + MRC401Job[index].ItemID)
		if err != nil {
			fmt.Printf("Mrc401Create stub.GetState(key) [%s] Error %s\n", mrc400id+"_"+MRC401Job[index].ItemID, err)
			return errors.New("8600,Hyperledger internal error - " + err.Error())
		}

		if data == nil {
			return errors.New("8600,Item ID " + MRC401Job[index].ItemID + " not exists in project " + mrc400id)
		}
		if err = json.Unmarshal([]byte(data), &MRC401ItemData); err != nil {
			return errors.New("6205,MRC401 Data is in the wrong data - " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].ItemURL, &MRC401ItemData.ItemURL, "url", 0, 255, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item ItemURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].ItemImageURL, &MRC401ItemData.ItemImageURL, "url", 0, 255, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item ItemImageURL error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].GroupID, &MRC401ItemData.GroupID, "string", 0, 40, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item GroupID error : " + err.Error())
		}

		if MRC401Job[index].SellFee != MRC401ItemData.SellFee {
			if MRC401ItemData.SellDate > 0 {
				return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already sale")
			}

			if MRC401ItemData.AuctionDate > 0 {
				return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already auction")
			}
		}

		if err = util.NumericDataCheck(MRC401Job[index].SellFee, &MRC401ItemData.SellFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item SellFee error : " + err.Error())
		}

		if MRC401Job[index].MeltingFee != MRC401ItemData.MeltingFee {
			if MRC401ItemData.SellDate > 0 {
				return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already sale")
			}

			if MRC401ItemData.AuctionDate > 0 {
				return errors.New("3004,MRC401 [" + MRC401Job[index].ItemID + "] is already auction")
			}
		}

		if err = util.NumericDataCheck(MRC401Job[index].MeltingFee, &MRC401ItemData.MeltingFee, "0", "99.9999", 4, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item MeltingFee error : " + err.Error())
		}

		if err = util.DataAssign(MRC401Job[index].Transferable, &MRC401ItemData.Transferable, "string", 1, 128, false); err != nil {
			return errors.New("3005," + MRC401Job[index].ItemID + " item Transferable error : " + err.Error())
		}

		if MRC401ItemData.Transferable != "Permanent" && MRC401ItemData.Transferable != "Bound" && MRC401ItemData.Transferable != "Temprary" {
			return errors.New("3005," + MRC401Job[index].ItemID + " item Transferable value is Permanent, Bound, Temprary ")
		}

		if err = mrc401set(stub, mrc400id+"_"+MRC401Job[index].ItemID, MRC401ItemData, "mrc401_update", []string{mrc400id + "_" + MRC401Job[index].ItemID, MRC400ProjectData.Owner, MRC401ItemData.InititalReserve, MRC401ItemData.InititalToken, signature, tkey}); err != nil {
			return err
		}
		logData = append(logData, MRC401Job[index].ItemID)
	}

	// subtract token for item initial price
	for token, totPrice := range createTotal {
		if totPrice.IsPositive() {
			if err = SubtractToken(stub, &projectOwnerWallet, token, totPrice.String()); err != nil {
				return err
			}
		}
	}

	// save create info
	// - for update balance
	// - for nonce update

	if err = SetAddressInfo(stub, MRC400ProjectData.Owner, projectOwnerWallet, "mrc401create", []string{mrc400id, MRC400ProjectData.Owner,
		util.JSONEncode(logData),
		signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401Transfer transfer
func Mrc401Transfer(stub shim.ChaincodeStubInterface, mrc401id, fromAddr, toAddr, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.MetaWallet
	var MRC401ItemData mtc.MRC401
	var mrc401 string

	// get item
	if mrc401, err = Mrc401get(stub, mrc401id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(mrc401), &MRC401ItemData); err != nil {
		return errors.New("3004,MRC401 [" + mrc401id + "] is in the wrong data")
	}

	// item transferable ?
	if MRC401ItemData.Transferable == "Bound" {
		return errors.New("5002,MRC401 [" + mrc401id + "] is not transferable")
	}
	if MRC401ItemData.SellDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already sale")
	}

	if MRC401ItemData.AuctionDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already auction")
	}

	if MRC401ItemData.Owner != fromAddr {
		return errors.New("3004,MRC401 [" + mrc401id + "] is not your item")
	}
	// get owner info
	if ownerWallet, err = GetAddressInfo(stub, fromAddr); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{fromAddr, toAddr, mrc401id, tkey}, "|"),
		signature); err != nil {
		return err
	}

	// item owner change
	MRC401ItemData.Owner = toAddr
	if err := mrc401set(stub, mrc401id, MRC401ItemData, "mrc401_transfer", args); err != nil {
		return err
	}

	// save prev owner info for nonce update
	if err = SetAddressInfo(stub, fromAddr, ownerWallet, "mrc401transfer", []string{mrc401id, fromAddr, toAddr, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401Sell Mrc401Sell
func Mrc401Sell(stub shim.ChaincodeStubInterface, seller, mrc400id, itemData, signature, tkey string, args []string) error {
	var err error
	var sellerData mtc.MetaWallet
	var now int64
	var buf string
	var MRC401SellData []mtc.MRC401Sell
	var MRC401ItemData mtc.MRC401

	var MRC400ProjectData mtc.MRC400
	var mrc400 string
	var logData []mtc.MRC401Sell
	var keyCheck map[string]int

	if err = json.Unmarshal([]byte(itemData), &MRC401SellData); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401SellData) > 100 {
		return errors.New("3002,There must be 100 or fewer sell item")
	}
	// get seller info
	if sellerData, err = GetAddressInfo(stub, seller); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&sellerData, tkey,
		strings.Join([]string{seller, itemData, tkey}, "|"),
		signature); err != nil {
		return err
	}

	logData = make([]mtc.MRC401Sell, len(MRC401SellData))
	keyCheck = make(map[string]int, len(MRC401SellData))
	now = time.Now().Unix()
	for index := range MRC401SellData {
		if _, exists := keyCheck[MRC401SellData[index].ItemID]; exists != false {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is duplicate")
		}
		keyCheck[MRC401SellData[index].ItemID] = 0

		if buf, err = Mrc401get(stub, MRC401SellData[index].ItemID); err != nil {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] - " + err.Error())
		}

		if err = json.Unmarshal([]byte(buf), &MRC401ItemData); err != nil {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is in the wrong data - " + err.Error())
		}

		if mrc400id != MRC401ItemData.MRC400 {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is not MRC400 " + mrc400id + " item")
		}

		// get project
		if mrc400, err = Mrc400get(stub, MRC401ItemData.MRC400); err != nil {
			return err
		}
		if err = json.Unmarshal([]byte(mrc400), &MRC400ProjectData); err != nil {
			return errors.New("3004,MRC400 [" + MRC401ItemData.MRC400 + "] is in the wrong data")
		}

		// item owner check.
		if MRC401ItemData.Owner != seller {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is not your item")
		}

		if MRC401ItemData.SellDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is already sale")
		}
		if MRC401ItemData.AuctionDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401SellData[index].ItemID + "] is already auction")
		}
		// sell price check
		if err = util.NumericDataCheck(MRC401SellData[index].SellPrice, &MRC401ItemData.SellPrice, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellPrice error : " + err.Error())
		}

		//  token check
		if MRC401SellData[index].SellToken != MRC400ProjectData.AllowToken && MRC401SellData[index].SellToken != "0" {
			if MRC400ProjectData.AllowToken != "0" {
				return errors.New("3005," + util.GetOrdNumber(index) + " item SellToken is must " + MRC400ProjectData.AllowToken + " or metacoin")
			}
			return errors.New("3005," + util.GetOrdNumber(index) + " item SellToken is must " + MRC400ProjectData.AllowToken)
		}
		MRC401ItemData.SellToken = MRC401SellData[index].SellToken

		// save item
		MRC401ItemData.SellDate = now
		if err = mrc401set(stub, MRC401SellData[index].ItemID, MRC401ItemData, "mrc401_sell", []string{MRC401SellData[index].ItemID, seller, MRC401SellData[index].SellPrice, MRC401SellData[index].SellToken, signature, tkey}); err != nil {
			return err
		}

		logData = append(logData, mtc.MRC401Sell{MRC401SellData[index].ItemID, MRC401SellData[index].SellPrice, MRC401SellData[index].SellToken})

	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, seller, sellerData, "mrc401sell", []string{util.JSONEncode(logData), seller, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401UnSell Mrc401UnSell
func Mrc401UnSell(stub shim.ChaincodeStubInterface, seller, mrc400id, itemData, signature, tkey string, args []string) error {
	var err error
	var ownerWallet mtc.MetaWallet
	var buf string
	var MRC401list []string
	var MRC401ItemData mtc.MRC401
	var logData []string
	var keyCheck map[string]int

	if err = json.Unmarshal([]byte(itemData), &MRC401list); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401list) > 100 {
		return errors.New("3002,There must be 100 or fewer unsell item")
	}

	// get seller info
	if ownerWallet, err = GetAddressInfo(stub, seller); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&ownerWallet, tkey,
		strings.Join([]string{seller, itemData, tkey}, "|"),
		signature); err != nil {
		return err
	}

	logData = make([]string, len(MRC401list))
	keyCheck = make(map[string]int, len(MRC401list))
	for index := range MRC401list {

		if _, exists := keyCheck[MRC401list[index]]; exists != false {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is duplicate")
		}
		keyCheck[MRC401list[index]] = 0

		if buf, err = Mrc401get(stub, MRC401list[index]); err != nil {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] - " + err.Error())
		}

		if err = json.Unmarshal([]byte(buf), &MRC401ItemData); err != nil {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is in the wrong data")
		}

		if mrc400id != MRC401ItemData.MRC400 {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not MRC400 " + mrc400id + " item")
		}

		// item owner check.
		if MRC401ItemData.Owner != seller {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not your item")
		}

		if MRC401ItemData.SellDate == 0 {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not sale")
		}

		// save item
		MRC401ItemData.SellDate = 0
		MRC401ItemData.SellPrice = "0"
		MRC401ItemData.SellToken = "0"
		if err = mrc401set(stub, MRC401list[index], MRC401ItemData, "mrc401_unsell", []string{MRC401list[index], seller, signature, tkey}); err != nil {
			return err
		}
		logData = append(logData, MRC401list[index])
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, seller, ownerWallet, "mrc401unsell", []string{util.JSONEncode(logData), seller, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401Buy Mrc401Buy
func Mrc401Buy(stub shim.ChaincodeStubInterface, buyer, mrc401id, signature, tkey string, args []string) error {
	var err error
	var buffer string

	var sellerWallet mtc.MetaWallet
	var buyerWallet mtc.MetaWallet
	var projectOwnerWallet mtc.MetaWallet
	var MRC401ItemData mtc.MRC401
	var MRC400ProjectData mtc.MRC400
	var seller string

	var payPrice decimal.Decimal     // Trade price
	var feeRate decimal.Decimal      // Melting Fee(percents)    100% == 100
	var Percent decimal.Decimal      // "100"  (Price * feeRate / Percent)
	var receivePrice decimal.Decimal // The amount the owner will receive
	var feePrice decimal.Decimal     // The amount the creator will receive

	var PaymentInfo []logDataBuy
	PaymentInfo = make([]logDataBuy, 0, 3)

	// get item info
	if buffer, err = Mrc401get(stub, mrc401id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(buffer), &MRC401ItemData); err != nil {
		return errors.New("3004,MRC401 [" + mrc401id + "] is in the wrong data")
	}
	if MRC401ItemData.SellDate == 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is not for sale")
	}
	seller = MRC401ItemData.Owner

	// sign check
	if buyerWallet, err = GetAddressInfo(stub, buyer); err != nil {
		return err
	}
	if err = NonceCheck(&buyerWallet, tkey,
		strings.Join([]string{mrc401id, tkey}, "|"),
		signature); err != nil {
		return err
	}

	payPrice, _ = decimal.NewFromString(MRC401ItemData.SellPrice)
	if err = SubtractToken(stub, &buyerWallet, MRC401ItemData.SellToken, payPrice.String()); err != nil {
		return err
	}

	// save buyer info
	if err = SetAddressInfo(stub, buyer, buyerWallet, "transfer_mrc401buy", []string{buyer, seller, payPrice.String(), MRC401ItemData.SellToken, signature, "0", "", mrc401id, tkey}); err != nil {
		return err
	}
	PaymentInfo = append(PaymentInfo, logDataBuy{buyer, seller, payPrice.String(), MRC401ItemData.SellToken, "mrc401_buy"})

	// get Project
	if buffer, err = Mrc400get(stub, MRC401ItemData.MRC400); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(buffer), &MRC400ProjectData); err != nil {
		return errors.New("6205,MRC400 [" + mrc401id + "] is in the wrong data")
	}

	// first sale ?
	if seller == MRC400ProjectData.Owner {
		feePrice = decimal.Zero
		receivePrice = payPrice
	} else {
		feeRate, _ = decimal.NewFromString(MRC401ItemData.MeltingFee)
		Percent, _ = decimal.NewFromString("100")
		feePrice = payPrice.Mul(feeRate).Div(Percent).Floor()
		receivePrice = payPrice.Sub(feePrice)
	}

	if feePrice.IsPositive() {
		// get Proejct Owner
		if projectOwnerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
			return err
		}

		// Add trade fee
		if err = AddToken(stub, &projectOwnerWallet, MRC401ItemData.SellToken, feePrice.String(), 0); err != nil {
			return err
		}
		// Save Project Owner
		if err = SetAddressInfo(stub, MRC400ProjectData.Owner, projectOwnerWallet, "receive_mrc401fee", []string{seller, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.SellToken, signature, "0", "", mrc401id, tkey}); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, logDataBuy{seller, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.SellToken, "mrc401_recv_fee"})
	}

	if receivePrice.IsPositive() {
		// get owner data for trade price recv
		if sellerWallet, err = GetAddressInfo(stub, seller); err != nil {
			return err
		}

		// add remain price
		if err = AddToken(stub, &sellerWallet, MRC401ItemData.SellToken, receivePrice.String(), 0); err != nil {
			return err
		}

		// save owner info
		if err = SetAddressInfo(stub, seller, sellerWallet, "receive_mrc401sell", []string{buyer, seller, receivePrice.String(), MRC401ItemData.SellToken, signature, "0", "", mrc401id, tkey}); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, logDataBuy{buyer, seller, receivePrice.String(), MRC401ItemData.SellToken, "mrc401_recv_sell"})
	}

	// item owner change for Buy
	MRC401ItemData.Owner = buyer
	MRC401ItemData.SellDate = 0
	if err = mrc401set(stub, mrc401id, MRC401ItemData, "mrc401_buy", []string{mrc401id, seller, buyer, util.JSONEncode(PaymentInfo), signature, tkey}); err != nil {
		return err
	}

	return nil
}

// Mrc401Melt Mrc401Melt
func Mrc401Melt(stub shim.ChaincodeStubInterface, mrc401id, signature, tkey string, args []string) error {
	var err error
	var itemOwnerWallet mtc.MetaWallet
	var projectOwnerWallet mtc.MetaWallet
	var MRC401ItemData mtc.MRC401
	var buffer string
	var MRC400ProjectData mtc.MRC400
	var itemOwner string

	var PaymentInfo []logDataBuy

	var InititalPrice decimal.Decimal //  The amount given by the creator when creating an item
	var feeRate decimal.Decimal       // Melting Fee(percents)    100% == 100
	var Percent decimal.Decimal       // "100"  (Price * feeRate / Percent)
	var receivePrice decimal.Decimal  // The amount the owner will receive
	var feePrice decimal.Decimal      // The amount the creator will receive

	PaymentInfo = make([]logDataBuy, 0, 2)

	// get item info
	if buffer, err = Mrc401get(stub, mrc401id); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(buffer), &MRC401ItemData); err != nil {
		return errors.New("3004,MRC401 [" + mrc401id + "] is in the wrong data")
	}

	if MRC401ItemData.SellDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already sale")
	}

	if MRC401ItemData.AuctionDate > 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already auction")
	}

	itemOwner = MRC401ItemData.Owner
	if itemOwner == "MELTED" {
		return errors.New("3004,MRC401 [" + mrc401id + "] is already melted")
	}

	// get item owner info
	if itemOwnerWallet, err = GetAddressInfo(stub, itemOwner); err != nil {
		return err
	}

	// sign check.
	if err = NonceCheck(&itemOwnerWallet, tkey,
		strings.Join([]string{mrc401id, tkey}, "|"),
		signature); err != nil {
		return err
	}

	// InititalPrice process
	InititalPrice, _ = decimal.NewFromString(MRC401ItemData.InititalReserve)
	if InititalPrice.IsPositive() {
		feeRate, _ = decimal.NewFromString(MRC401ItemData.MeltingFee)
		Percent, _ = decimal.NewFromString("100")
		feePrice = InititalPrice.Mul(feeRate).Div(Percent).Floor()
		receivePrice = InititalPrice.Sub(feePrice)

		if feePrice.IsPositive() {
			// get Project
			if buffer, err = Mrc400get(stub, MRC401ItemData.MRC400); err != nil {
				return err
			}

			if err = json.Unmarshal([]byte(buffer), &MRC400ProjectData); err != nil {
				return errors.New("6205,MRC400 [" + mrc401id + "] is in the wrong data")
			}

			// get Proejct Owner
			if projectOwnerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
				return err
			}

			// Add melt fee
			if err = AddToken(stub, &projectOwnerWallet, MRC401ItemData.InititalToken, feePrice.String(), 0); err != nil {
				return err
			}
			// Save Project Owner
			PaymentInfo = append(PaymentInfo, logDataBuy{mrc401id, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.InititalToken, "mrc401_recv_meltfee"})
			if err = SetAddressInfo(stub, MRC400ProjectData.Owner, projectOwnerWallet, "receive_meltfee", []string{mrc401id, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.InititalToken, signature, "0", "", mrc401id, tkey}); err != nil {
				return err
			}
		}

		if receivePrice.IsPositive() {
			// add remain price
			if err = AddToken(stub, &itemOwnerWallet, MRC401ItemData.InititalToken, receivePrice.String(), 0); err != nil {
				return err
			}

			// save owner info
			PaymentInfo = append(PaymentInfo, logDataBuy{mrc401id, itemOwner, receivePrice.String(), MRC401ItemData.InititalToken, "mrc401_recv_melt"})
			if err = SetAddressInfo(stub, itemOwner, itemOwnerWallet, "receive_melt", []string{mrc401id, itemOwner, receivePrice.String(), MRC401ItemData.InititalToken, signature, "0", "", mrc401id, tkey}); err != nil {
				return err
			}
		}
	}

	// item owner change for MELTED
	MRC401ItemData.Owner = "MELTED"
	MRC401ItemData.MeltingDate = time.Now().Unix()
	if err = mrc401set(stub, mrc401id, MRC401ItemData, "mrc401_melt", []string{mrc401id, itemOwner, util.JSONEncode(PaymentInfo), signature, tkey}); err != nil {
		return err
	}

	return nil
}

// Mrc401Auction Mrc401Sell
func Mrc401Auction(stub shim.ChaincodeStubInterface, seller, mrc400id, itemData, signature, tkey string, args []string) error {
	var err error
	var now int64
	var buffer string

	var sellerWallet mtc.MetaWallet

	var MRC401AuctionData []mtc.MRC401Auction
	var MRC401ItemData mtc.MRC401
	var MRC400ProjectData mtc.MRC400

	var auctionStart, auctionBuynow decimal.Decimal

	var keyCheck map[string]int

	if err = json.Unmarshal([]byte(itemData), &MRC401AuctionData); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401AuctionData) > 100 {
		return errors.New("3002,There must be 100 or fewer sell item")
	}
	// get seller info
	if sellerWallet, err = GetAddressInfo(stub, seller); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&sellerWallet, tkey,
		strings.Join([]string{seller, itemData, tkey}, "|"),
		signature); err != nil {
		return err
	}
	// auction item check.
	now = time.Now().Unix()

	keyCheck = make(map[string]int, len(MRC401AuctionData))
	for index := range MRC401AuctionData {
		if _, exists := keyCheck[MRC401AuctionData[index].ItemID]; exists != false {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is duplicate")
		}
		keyCheck[MRC401AuctionData[index].ItemID] = 0

		if buffer, err = Mrc401get(stub, MRC401AuctionData[index].ItemID); err != nil {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] - " + err.Error())
		}

		if err = json.Unmarshal([]byte(buffer), &MRC401ItemData); err != nil {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is in the wrong data - " + err.Error())
		}

		if mrc400id != MRC401ItemData.MRC400 {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is not MRC400 " + mrc400id + " item")
		}

		// get project
		if buffer, err = Mrc400get(stub, MRC401ItemData.MRC400); err != nil {
			return err
		}

		if err = json.Unmarshal([]byte(buffer), &MRC400ProjectData); err != nil {
			return errors.New("3004,MRC400 [" + MRC401ItemData.MRC400 + "] is in the wrong data")
		}
		// item owner check.
		if MRC401ItemData.Owner != seller {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is not your item")
		}

		// sale or auction item ?
		if MRC401ItemData.SellDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is already sale")
		}
		if MRC401ItemData.AuctionDate > 0 {
			return errors.New("3004,MRC401 [" + MRC401AuctionData[index].ItemID + "] is already auction")
		}

		// start price check
		if err = util.NumericDataCheck(MRC401AuctionData[index].AuctionStartPrice, &MRC401ItemData.AuctionStartPrice, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_start_price error : " + err.Error())
		}

		// buynow price check
		if err = util.NumericDataCheck(MRC401AuctionData[index].AuctionBuyNowPrice, &MRC401ItemData.AuctionBuyNowPrice, "0", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, true); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_buynow_price error : " + err.Error())
		}

		// bidding unit price check
		if err = util.NumericDataCheck(MRC401AuctionData[index].AuctionBiddingUnit, &MRC401ItemData.AuctionBiddingUnit, "1", "99999999999999999999999999999999999999999999999999999999999999999999999999999999", 0, false); err != nil {
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_bidding_unit error : " + err.Error())
		}

		auctionStart, _ = decimal.NewFromString(MRC401ItemData.AuctionStartPrice)
		auctionBuynow, _ = decimal.NewFromString(MRC401ItemData.AuctionBuyNowPrice)
		if auctionBuynow.IsZero() == false && auctionBuynow.Cmp(auctionStart) < 0 {
			return errors.New("3005," + util.GetOrdNumber(index) + " item buynow price is must be greater then auction start price")
		}

		//  token check
		if MRC401AuctionData[index].AuctionToken != MRC400ProjectData.AllowToken && MRC401AuctionData[index].AuctionToken != "0" {
			if MRC400ProjectData.AllowToken != "0" {
				return errors.New("3005," + util.GetOrdNumber(index) + " item auction_token is must " + MRC400ProjectData.AllowToken + " or metacoin")
			}
			return errors.New("3005," + util.GetOrdNumber(index) + " item auction_token is must " + MRC400ProjectData.AllowToken)
		}
		MRC401ItemData.AuctionToken = MRC401AuctionData[index].AuctionToken

		MRC401ItemData.AuctionEnd = MRC401AuctionData[index].AuctionEnd
		if (MRC401AuctionData[index].AuctionEnd - now) < 3600 {
			MRC401ItemData.AuctionEnd = now + 3600
		} else if (MRC401AuctionData[index].AuctionEnd - now) > 1814400 {
			MRC401ItemData.AuctionEnd = now + 1814400
		}
		// save item
		MRC401ItemData.AuctionCurrentPrice = "0"
		MRC401ItemData.AuctionCurrentBidder = ""
		MRC401ItemData.AuctionDate = now
		mrc401set(stub, MRC401AuctionData[index].ItemID, MRC401ItemData, "mrc401_auction", []string{MRC401AuctionData[index].ItemID, seller, MRC401ItemData.AuctionStartPrice, MRC401ItemData.AuctionToken, MRC401ItemData.AuctionBuyNowPrice, MRC401ItemData.AuctionBiddingUnit, signature, tkey})
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, seller, sellerWallet, "mrc401auction", []string{itemData, seller, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401UnAuction Mrc401UnAuction
func Mrc401UnAuction(stub shim.ChaincodeStubInterface, seller, mrc400id, itemData, signature, tkey string, args []string) error {
	var err error
	var sellerWallet mtc.MetaWallet
	var buffer string
	var MRC401list []string
	var MRC401ItemData mtc.MRC401
	var keyCheck map[string]int

	if err = json.Unmarshal([]byte(itemData), &MRC401list); err != nil {
		return errors.New("3004,Selldata is in the wrong data " + err.Error())
	}
	if len(MRC401list) > 100 {
		return errors.New("3002,There must be 100 or fewer unauction item")
	}

	// get seller info
	if sellerWallet, err = GetAddressInfo(stub, seller); err != nil {
		return err
	}

	// sign check
	if err = NonceCheck(&sellerWallet, tkey,
		strings.Join([]string{seller, itemData, tkey}, "|"),
		signature); err != nil {
		return err
	}

	keyCheck = make(map[string]int, len(MRC401list))
	for index := range MRC401list {
		if _, exists := keyCheck[MRC401list[index]]; exists != false {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is duplicate")
		}
		keyCheck[MRC401list[index]] = 0

		if buffer, err = Mrc401get(stub, MRC401list[index]); err != nil {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] - " + err.Error())
		}

		if err = json.Unmarshal([]byte(buffer), &MRC401ItemData); err != nil {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is in the wrong data")
		}

		if mrc400id != MRC401ItemData.MRC400 {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not MRC400 " + mrc400id + " item")
		}

		// item owner check.
		if MRC401ItemData.Owner != seller {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not your item")
		}

		// is auction item ?
		if MRC401ItemData.AuctionDate == 0 {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] is not auction item")
		}

		// bidder exists ?
		if MRC401ItemData.AuctionCurrentBidder != "" {
			return errors.New("3004,MRC401 [" + MRC401list[index] + "] there is a bidder, so the auction cannot be canceled")
		}

		// save item
		MRC401ItemData.AuctionDate = 0
		MRC401ItemData.AuctionEnd = 0
		MRC401ItemData.AuctionToken = ""
		MRC401ItemData.AuctionBiddingUnit = "0"
		MRC401ItemData.AuctionStartPrice = "0"
		MRC401ItemData.AuctionBuyNowPrice = "0"
		MRC401ItemData.AuctionCurrentPrice = "0"
		MRC401ItemData.AuctionCurrentBidder = ""
		mrc401set(stub, MRC401list[index], MRC401ItemData, "mrc401_unauction", []string{MRC401list[index], seller, signature, tkey})
	}

	// save owner info for nonce update
	if err = SetAddressInfo(stub, seller, sellerWallet, "mrc401unauction", []string{itemData, seller, signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401AuctionBid Mrc401AuctionBid
func Mrc401AuctionBid(stub shim.ChaincodeStubInterface, buyer, mrc401id, amount, token, signature, tkey string, args []string) error {
	var err error
	var now int64
	var buffer string

	var buyerWallet, currentBidderWallet mtc.MetaWallet
	var MRC401ItemData mtc.MRC401

	var currentPrice, bidAmount, bidUnit, buyNow, diffPrice decimal.Decimal

	var PaymentInfo []logDataBuy

	now = time.Now().Unix()
	PaymentInfo = make([]logDataBuy, 0, 4)

	// get item info
	if buffer, err = Mrc401get(stub, mrc401id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(buffer), &MRC401ItemData); err != nil {
		return errors.New("3004,MRC401 [" + mrc401id + "] is in the wrong data")
	}

	// is auction ?
	if MRC401ItemData.AuctionDate == 0 {
		return errors.New("3004,MRC401 [" + mrc401id + "] is not for auction")
	}
	if MRC401ItemData.AuctionEnd < now {
		return errors.New("3004,MRC401 [" + mrc401id + "] has completed auction")
	}

	// buyer check.
	if MRC401ItemData.AuctionCurrentBidder == buyer {
		return errors.New("3004,You are already the highest bidder")
	}

	// sign check
	if buyerWallet, err = GetAddressInfo(stub, buyer); err != nil {
		return err
	}
	if err = NonceCheck(&buyerWallet, tkey,
		strings.Join([]string{mrc401id, amount, token, tkey}, "|"),
		signature); err != nil {
		return err
	}

	// token check.
	if MRC401ItemData.AuctionToken != token {
		return errors.New("3004,Only " + MRC401ItemData.AuctionToken + " tokens can be bid")
	}

	// sell price check
	if _, err = util.ParsePositive(amount); err != nil {
		return err
	}

	if bidAmount, err = decimal.NewFromString(amount); err != nil {
		return errors.New("3004,The bid amount is incorrect. " + err.Error())
	}

	// get current price

	buyNow, _ = decimal.NewFromString(MRC401ItemData.AuctionBuyNowPrice)

	// first bidding ?
	if MRC401ItemData.AuctionCurrentBidder == "" {
		currentPrice, _ = decimal.NewFromString(MRC401ItemData.AuctionStartPrice)
		if bidAmount.Cmp(currentPrice) < 0 {
			return errors.New("3004,The bid amount must be equal to or greater than the starting price")
		}
	} else {
		currentPrice, _ = decimal.NewFromString(MRC401ItemData.AuctionCurrentPrice)
		if bidAmount.Cmp(currentPrice) < 1 {
			return errors.New("3004,The bid amount must be greater than the current price")
		}
	}

	// check new bid price
	if buyNow.IsZero() == false {
		if bidAmount.Cmp(buyNow) > 0 {
			return errors.New("3004,The bid amount must be less than or equal to the purchase buynow price")
		}
	}

	// buyer token subtract & save
	if err = SubtractToken(stub, &buyerWallet, MRC401ItemData.AuctionToken, amount); err != nil {
		return err
	}

	if err = SetAddressInfo(stub, buyer, buyerWallet, "transfer_mrc401bid", []string{buyer, mrc401id, amount, MRC401ItemData.AuctionToken, signature, "0", "", mrc401id, tkey}); err != nil {
		return err
	}
	PaymentInfo = append(PaymentInfo, logDataBuy{buyer, mrc401id, amount, MRC401ItemData.AuctionToken, "mrc401_bid"})

	// refund current bidder
	if MRC401ItemData.AuctionCurrentBidder != "" {
		if currentBidderWallet, err = GetAddressInfo(stub, MRC401ItemData.AuctionCurrentBidder); err != nil {
			return err
		}
		if err = AddToken(stub, &currentBidderWallet, MRC401ItemData.AuctionToken, MRC401ItemData.AuctionCurrentPrice, 0); err != nil {
			return err
		}
		if err = SetAddressInfo(stub, MRC401ItemData.AuctionCurrentBidder, currentBidderWallet, "receive_mrc401refund", []string{mrc401id, MRC401ItemData.AuctionCurrentBidder, MRC401ItemData.AuctionCurrentPrice, MRC401ItemData.AuctionToken, signature, "0", "", mrc401id, tkey}); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, logDataBuy{mrc401id, MRC401ItemData.AuctionCurrentBidder, MRC401ItemData.AuctionCurrentPrice, MRC401ItemData.AuctionToken, "mrc401_recv_refund"})
	}

	// set new bidder
	MRC401ItemData.AuctionCurrentPrice = amount
	MRC401ItemData.AuctionCurrentBidder = buyer

	// buynow
	if buyNow.IsZero() == false && bidAmount.Cmp(buyNow) == 0 {
		return auctionFinish(stub, mrc401id, MRC401ItemData, PaymentInfo, true)
	}

	// save bid info
	bidUnit, _ = decimal.NewFromString(MRC401ItemData.AuctionBiddingUnit)
	diffPrice = bidAmount.Sub(currentPrice)
	if diffPrice.Div(bidUnit).Floor().Mul(bidUnit).Cmp(diffPrice) != 0 {
		return errors.New("3004,The bid amount must be greater than the current amount plus the bid units")
	}

	if err = mrc401set(stub, mrc401id, MRC401ItemData, "mrc401_auctionbid", []string{mrc401id, buyer, util.JSONEncode(PaymentInfo), signature, tkey}); err != nil {
		return err
	}
	return nil
}

// Mrc401AuctionFinish Mrc401AuctionFinish
func Mrc401AuctionFinish(stub shim.ChaincodeStubInterface, mrc401id string) error {
	var err error
	var buffer string

	var MRC401ItemData mtc.MRC401
	var PaymentInfo []logDataBuy

	// get item info
	if buffer, err = Mrc401get(stub, mrc401id); err != nil {
		return err
	}
	if err = json.Unmarshal([]byte(buffer), &MRC401ItemData); err != nil {
		return errors.New("3004,MRC401 [" + mrc401id + "] is in the wrong data")
	}
	PaymentInfo = make([]logDataBuy, 0, 2)
	return auctionFinish(stub, mrc401id, MRC401ItemData, PaymentInfo, false)
}

// auctionFinish auction finish or winningbid process
func auctionFinish(stub shim.ChaincodeStubInterface, mrc401id string, MRC401ItemData mtc.MRC401, PaymentInfo []logDataBuy, isBuynow bool) error {
	var err error
	var buffer string

	var projectOwnerWallet mtc.MetaWallet
	var sellerWallet mtc.MetaWallet

	var MRC400ProjectData mtc.MRC400
	var now int64
	var seller, buyer string
	var jobType string

	var payPrice decimal.Decimal     // Trade price
	var feeRate decimal.Decimal      // Melting Fee(percents)    100% == 100
	var Percent decimal.Decimal      // "100"  (Price * feeRate / Percent)
	var receivePrice decimal.Decimal // The amount the owner will receive
	var feePrice decimal.Decimal     // The amount the creator will receive

	now = time.Now().Unix()

	if isBuynow == false {
		// auction not expire ?
		if MRC401ItemData.AuctionEnd > now {
			return errors.New("3004,MRC401 [" + mrc401id + "] is under auction.")
		}
	} else {
		if MRC401ItemData.AuctionBuyNowPrice == "0" || MRC401ItemData.AuctionBuyNowPrice != MRC401ItemData.AuctionCurrentPrice {
			return errors.New("3004,MRC401 [" + mrc401id + "] is not buynow item.")
		}
	}

	// save seller, buyer
	seller = MRC401ItemData.Owner
	buyer = MRC401ItemData.AuctionCurrentBidder

	// auction fail.
	if buyer == "" {
		if MRC401ItemData.AuctionDate == 0 || MRC401ItemData.AuctionEnd == 0 {
			return errors.New("3004,MRC401 [" + mrc401id + "] is not auction")
		}
		MRC401ItemData.AuctionDate = 0
		MRC401ItemData.AuctionEnd = 0
		MRC401ItemData.AuctionToken = ""
		MRC401ItemData.AuctionBiddingUnit = "0"
		MRC401ItemData.AuctionStartPrice = "0"
		MRC401ItemData.AuctionBuyNowPrice = "0"
		MRC401ItemData.AuctionCurrentPrice = "0"
		MRC401ItemData.AuctionCurrentBidder = ""
		if err = mrc401set(stub, mrc401id, MRC401ItemData, "mrc401_auctionfailure", []string{mrc401id, seller, "", util.JSONEncode(PaymentInfo), "", ""}); err != nil {
			return err
		}
		return nil
	}

	// get Project
	if buffer, err = Mrc400get(stub, MRC401ItemData.MRC400); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(buffer), &MRC400ProjectData); err != nil {
		return errors.New("6205,MRC400 [" + mrc401id + "] is in the wrong data")
	}

	// owner sale ?
	payPrice, _ = decimal.NewFromString(MRC401ItemData.AuctionCurrentPrice)
	if seller == MRC400ProjectData.Owner {
		feePrice = decimal.Zero
		receivePrice = payPrice
	} else {
		feeRate, _ = decimal.NewFromString(MRC401ItemData.SellFee)
		Percent, _ = decimal.NewFromString("100")
		feePrice = payPrice.Mul(feeRate).Div(Percent).Floor()
		receivePrice = payPrice.Sub(feePrice)
	}

	if feePrice.IsPositive() {
		// get Proejct Owner
		if projectOwnerWallet, err = GetAddressInfo(stub, MRC400ProjectData.Owner); err != nil {
			return err
		}

		// Add trade fee
		if err = AddToken(stub, &projectOwnerWallet, MRC401ItemData.AuctionToken, feePrice.String(), 0); err != nil {
			return err
		}
		// Save Project Owner
		if err = SetAddressInfo(stub, MRC400ProjectData.Owner, projectOwnerWallet, "receive_fee", []string{buyer, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.SellToken, "", "0", "", mrc401id, ""}); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, logDataBuy{seller, MRC400ProjectData.Owner, feePrice.String(), MRC401ItemData.AuctionToken, "mrc401_recv_fee"})
	}

	if receivePrice.IsPositive() {
		// get owner data for trade price recv
		if sellerWallet, err = GetAddressInfo(stub, seller); err != nil {
			return err
		}

		// add remain price
		if err = AddToken(stub, &sellerWallet, MRC401ItemData.AuctionToken, receivePrice.String(), 0); err != nil {
			return err
		}

		// save owner info
		if err = SetAddressInfo(stub, seller, sellerWallet, "receive_mrc401auction", []string{buyer, seller, receivePrice.String(), MRC401ItemData.SellToken, "", "0", "", mrc401id, ""}); err != nil {
			return err
		}
		PaymentInfo = append(PaymentInfo, logDataBuy{buyer, seller, receivePrice.String(), MRC401ItemData.AuctionToken, "mrc401_recv_auction"})
	}

	MRC401ItemData.Owner = buyer
	MRC401ItemData.AuctionDate = 0
	MRC401ItemData.AuctionEnd = 0
	MRC401ItemData.AuctionToken = ""
	MRC401ItemData.AuctionBiddingUnit = "0"
	MRC401ItemData.AuctionStartPrice = "0"
	MRC401ItemData.AuctionBuyNowPrice = "0"
	MRC401ItemData.AuctionCurrentPrice = "0"
	MRC401ItemData.AuctionCurrentBidder = ""

	if isBuynow {
		jobType = "mrc401_auctionbuynow"
	} else {
		jobType = "mrc401_auctionwinning"
	}
	if err = mrc401set(stub, mrc401id, MRC401ItemData, jobType, []string{mrc401id, buyer, util.JSONEncode(PaymentInfo), "", ""}); err != nil {
		return err
	}
	return nil
}
