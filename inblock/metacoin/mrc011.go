package metacoin

import (
	"errors"
	"strings"
	"time"

	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

// Mrc011set : save MRC011
func Mrc011set(stub shim.ChaincodeStubInterface, MRC011ID string, tk mtc.MRC011, JobType string, args []string) error {
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
		return errors.New("4204,Invalid MRC011 data format")
	}
	if err = stub.PutState(MRC011ID, dat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// Mrc011get : get MRC011
func Mrc011get(stub shim.ChaincodeStubInterface, MRC011ID string) (mtc.MRC011, error) {
	var data []byte
	var tk mtc.MRC011
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

// SetMRC012 : save MRC012
func SetMRC012(stub shim.ChaincodeStubInterface, MRC012ID string, tk mtc.MRC012, JobType string, args []string) error {
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
		return errors.New("4204,Invalid MRC012 data format")
	}
	if err = stub.PutState(MRC012ID, dat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// MRC011Create - MRC011 Creator
func MRC011Create(stub shim.ChaincodeStubInterface, mrc011id, creator, name, totalsupply, validitytype, istransfer, startdate, enddate, term, code, data, signature, tkey string, args []string) error {
	var err error
	var i int
	var j int64
	var mrc011 mtc.MRC011
	var creatorData mtc.MetaWallet

	if creatorData, err = GetAddressInfo(stub, creator); err != nil {
		return err
	}

	mrc011.Creator = creator
	mrc011.Name = name
	if i, err = util.Strtoint(totalsupply); err != nil {
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
		if j, err = util.Strtoint64(startdate); err != nil {
			return errors.New("1003,The Start_date must be integer")
		}
		mrc011.StartDate = j
		if j, err = util.Strtoint64(enddate); err != nil {
			return errors.New("1003,The end_date must be integer")
		}
		if j < time.Now().Unix() {
			return errors.New("1403,The end_date must be bigger then current timestamp")
		}
		mrc011.EndDate = j
	} else if validitytype == "duration" {
		if i, err = util.Strtoint(term); err != nil {
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

	if err = NonceCheck(&creatorData, tkey, strings.Join([]string{creator, name, totalsupply, validitytype, istransfer, code, data, tkey}, "|"), signature); err != nil {
		return err
	}

	mrc011.Code = code
	mrc011.Data = data

	return Mrc011set(stub, mrc011id, mrc011, "mrc011create", args)
}
