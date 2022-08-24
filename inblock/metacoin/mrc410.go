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

// TMRC410 : Coupon/Ticket base
type TMRC410 struct {
	Creator    string `json:"creator"`
	Name       string `json:"name"`
	IsTransfer int    `json:"is_transfer"`
	StartDate  int64  `json:"start_date"`
	EndDate    int64  `json:"end_date"`
	Term       int    `json:"term"`
	Code       string `json:"code"`
	Data       string `json:"data"`
	JobType    string `json:"job_type"`
	JobArgs    string `json:"job_args"`
	JobDate    int64  `json:"jobdate"`
}

// TMRC411 : Coupon/Ticket
type TMRC411 struct {
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

// setMRC410 : save MRC410
func setMRC410(stub shim.ChaincodeStubInterface, MRC410ID string, tk TMRC410, JobType string, args []string) error {
	var dat []byte
	var err error
	if len(MRC410ID) != 40 {
		return errors.New("4202,MRC410 id length is must be 40")
	}

	if strings.Index(MRC410ID, "MRC410_") != 0 {
		return errors.New("4204,Invalid ID")
	}

	tk.JobType = JobType
	tk.JobDate = time.Now().Unix()
	if len(args) > 0 {
		if dat, err = json.Marshal(args); err == nil {
			tk.JobArgs = string(dat)
		}
	} else {
		tk.JobArgs = ""
	}

	if dat, err = json.Marshal(tk); err != nil {
		return errors.New("4204,Invalid MRC410 data format")
	}
	if err = stub.PutState(MRC410ID, dat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// GetMRC410 : get MRC410
func GetMRC410(stub shim.ChaincodeStubInterface, MRC410ID string) (TMRC410, error) {
	var data []byte
	var tk TMRC410
	var err error

	if len(MRC410ID) != 40 {
		return tk, errors.New("4202,MRC410 id length is must be 40")
	}
	if strings.Index(MRC410ID, "MRC410_") != 0 {
		return tk, errors.New("4204,Invalid ID")
	}

	if data, err = stub.GetState(MRC410ID); err != nil {
		return tk, errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data == nil {
		return tk, errors.New("4201,MRC410 " + MRC410ID + " not exists")
	}
	if err = json.Unmarshal(data, &tk); err != nil {
		return tk, errors.New("4204,Invalid MRC410 data format")
	}
	return tk, nil
}

// SetMRC411 : save MRC411
func SetMRC411(stub shim.ChaincodeStubInterface, MRC411ID string, tk TMRC411, JobType string, args []string) error {
	var dat []byte
	var err error
	if len(MRC411ID) != 40 {
		return errors.New("4202,MRC411 id length is must be 40")
	}
	if strings.Index(MRC411ID, "MRC411_") != 0 {
		return errors.New("4204,Invalid ID")
	}

	tk.JobType = JobType
	tk.JobDate = time.Now().Unix()
	if len(args) > 0 {
		if dat, err = json.Marshal(args); err == nil {
			tk.JobArgs = string(dat)
		}
	} else {
		tk.JobArgs = ""
	}

	if dat, err = json.Marshal(tk); err != nil {
		return errors.New("4204,Invalid MRC411 data format")
	}
	if err = stub.PutState(MRC411ID, dat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// MRC410Create - MRC410 Creator
func MRC410Create(stub shim.ChaincodeStubInterface, creator, name, validitytype, istransfer, startdate, enddate, term, code, data, signature, tkey string, args []string) error {
	var err error
	var i int
	var j int64
	var mrc410 TMRC410
	var creatorData mtc.TWallet

	if creatorData, err = GetAddressInfo(stub, creator); err != nil {
		return err
	}

	mrc410.Creator = creator
	mrc410.Name = name
	mrc410.StartDate = 0
	mrc410.EndDate = 0
	mrc410.Term = 0

	if validitytype == "term" {
		if j, err = util.Strtoint64(startdate); err != nil {
			return errors.New("1003,The Start_date must be integer")
		}
		mrc410.StartDate = j
		if j, err = util.Strtoint64(enddate); err != nil {
			return errors.New("1003,The end_date must be integer")
		}
		if j < time.Now().Unix() {
			return errors.New("1403,The end_date must be bigger then current timestamp")
		}
		mrc410.EndDate = j
	} else if validitytype == "duration" {
		if i, err = util.Strtoint(term); err != nil {
			return errors.New("1003,The Term must be integer")
		}
		if i < 1 {
			return errors.New("1003,The term must be bigger then 0")
		}
		mrc410.Term = i
	} else {
		return errors.New("1003,The Validity_type must be term or duration")
	}

	if istransfer == "0" || istransfer == "" {
		mrc410.IsTransfer = 0
	} else {
		mrc410.IsTransfer = 1
	}

	if err = NonceCheck(&creatorData, tkey, strings.Join([]string{creator, name, validitytype, istransfer, code, data, tkey}, "|"), signature); err != nil {
		return err
	}

	mrc410.Code = code
	mrc410.Data = data

	return setMRC410(stub, "1", mrc410, "mrc410create", args)

}
