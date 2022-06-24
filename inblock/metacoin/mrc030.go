package metacoin

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/shopspring/decimal"

	"inblock/metacoin/mtc"
	"inblock/metacoin/util"
)

// MRC030Create  Vote create
func MRC030Create(stub shim.ChaincodeStubInterface, mrc030id, Creator, Title, Description, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType, URL, Question, SignNeed, signature, tkey string, args []string) error {
	var err error
	var CreatorData mtc.MetaWallet
	var decReward, totReward, decMaxRewardRecipient decimal.Decimal
	var iStartDate, iEndDate int64
	var iRewardType, iMaxRewardRecipient, iRewardToken int
	var vote mtc.MRC030
	var data []byte
	var q [20]mtc.MRC030Question

	if data, err = stub.GetState(mrc030id); err != nil {
		return errors.New("8100,Hyperledger internal error - " + err.Error())
	}
	if data != nil {
		return errors.New("6100,MRC030 [" + mrc030id + "] is already exists")
	}

	if CreatorData, err = GetAddressInfo(stub, Creator); err != nil {
		return err
	}

	if decMaxRewardRecipient, err = util.ParsePositive(MaxRewardRecipient); err != nil {
		return errors.New("1101,MaxRewardRecipient must be an integer string")
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

	if decReward, err = util.ParseNotNegative(Reward); err != nil {
		return errors.New("1101,Reward must be an integer string")
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

	if _, iRewardToken, err = GetToken(stub, RewardToken); err != nil {
		return err
	}

	if decReward.IsPositive() {
		totReward = decReward.Mul(decMaxRewardRecipient)
		if err = MRC010Subtract(stub, &CreatorData, RewardToken, totReward.String()); err != nil {
			return err
		}
	} else {
		totReward = decimal.Zero
	}

	if err = SetAddressInfo(stub, Creator, CreatorData, "mrc030create",
		[]string{Creator, mrc030id, totReward.String(), RewardToken, signature, "0", "", "", tkey}); err != nil {
		return err
	}

	if err = json.Unmarshal([]byte(Question), &q); err != nil {
		return errors.New("3290,Question is in the wrong data")
	}

	vote.QuestionCount = 0
	vote.Question = make([]mtc.MRC030Question, 0, 20)
	vote.QuestionInfo = make([]mtc.MRC030QuestionInfo, 0, 20)
	for index, ele := range q {
		if len(ele.Question) == 0 {
			break
		}
		vote.Question = append(vote.Question, ele)
		vote.QuestionInfo = append(vote.QuestionInfo, mtc.MRC030QuestionInfo{AnswerCount: 0, SubAnswerCount: []int{}})
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
				vote.Question[index].Item[idx2].SubItem = make([]mtc.MRC030SubItem, 0)
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

	if err = NonceCheck(&CreatorData, tkey,
		strings.Join([]string{Creator, Title, Reward, RewardToken, MaxRewardRecipient, RewardType, tkey}, "|"),
		signature); err != nil {
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

	if err = Mrc030set(stub, mrc030id, vote, "mrc030", []string{Creator, mrc030id, Title, StartDate, EndDate, Reward, RewardToken, MaxRewardRecipient, RewardType}); err != nil {
		return err
	}
	return nil
}

// MRC030Join  Vote join
func MRC030Join(stub shim.ChaincodeStubInterface, mrc030id, Voter, Answer, voteCreatorSign, signature string, args []string) error {
	var err error
	var voterData, voteCreatorData mtc.MetaWallet
	var vote mtc.MRC030
	var voting mtc.MRC031
	var mrc031key string
	var data []byte
	var AnswerTemp [20]mtc.MRC031Answer
	var currentAnswer int

	nowTime := time.Now().Unix()

	if vote, err = Mrc030get(stub, mrc030id); err != nil {
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

	if _, exists := vote.Voter[Voter]; exists {
		return errors.New("6100,MRC031 [" + mrc030id + "] is already voting")
	}

	if vote.Creator == Answer {
		return errors.New("6100,Vote creators cannot participate")
	}

	if vote.RewardType == 10 && len(vote.Voter) >= vote.MaxRewardRecipient {
		return errors.New("3290,No more voting")
	}

	mrc031key = mrc030id + "_" + Voter
	voting = mtc.MRC031{
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
	voting.Answer = make([]mtc.MRC031Answer, 0, 20)
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

	if voterData, err = GetAddressInfo(stub, Voter); err != nil {
		return err
	}

	if err = util.EcdsaSignVerify(voterData.Password,
		strings.Join([]string{Voter, mrc030id}, "|"),
		signature); err != nil {
		return err
	}

	if vote.IsNeedSign == 1 {
		if voteCreatorData, err = GetAddressInfo(stub, vote.Creator); err != nil {
			return err
		}
		if err = util.EcdsaSignVerify(voteCreatorData.Password,
			strings.Join([]string{Voter, mrc030id}, "|"),
			voteCreatorSign); err != nil {
			return err
		}
	}

	if vote.RewardType == 10 {
		if vote.Reward != "0" {
			if err = MRC010Add(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
				return err
			}
			if err = SetAddressInfo(stub, Voter, voterData, "mrc030reward", []string{mrc030id, Voter, vote.Reward, strconv.Itoa(vote.RewardToken), signature, "0", "", "", ""}); err != nil {
				return err
			}
		}
		vote.Voter[Voter] = 1
	} else {
		vote.Voter[Voter] = 0
	}
	// from, to, amount, tokenid, sign, unlockdate, tag, memo, tkey
	if data, err = json.Marshal([]string{mrc030id, Voter, vote.Reward, strconv.Itoa(vote.RewardToken), signature, "0", "", "", "", voteCreatorSign}); err == nil {
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
func MRC030Finish(stub shim.ChaincodeStubInterface, mrc030id string, args []string) error {
	var err error
	var voterData, CreatorData mtc.MetaWallet
	var vote mtc.MRC030
	var data []byte
	var JoinerList []string
	var decRefund decimal.Decimal
	var i int
	var key string
	if vote, err = Mrc030get(stub, mrc030id); err != nil {
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

		// 추첨
		JoinerList = make([]string, len(vote.Voter))

		if len(vote.Voter) < vote.MaxRewardRecipient { // 모두 추첨 대상

			for key = range vote.Voter {
				if voterData, err = GetAddressInfo(stub, key); err != nil {
					continue
				}
				if err = MRC010Add(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
					continue
				}
				if err = SetAddressInfo(stub, key, voterData, "mrc030reward", []string{mrc030id, key, vote.Reward, strconv.Itoa(vote.RewardToken), "", "0", "", "", ""}); err != nil {
					continue
				}
				vote.Voter[key] = 1
			}
		} else if len(vote.Voter) < (vote.MaxRewardRecipient * 2) { // 받지 못할 사람을 지정
			for key = range vote.Voter {
				JoinerList[i] = key
				i++
			}

			for vote.MaxRewardRecipient < len(JoinerList) {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(len(JoinerList))))
				if err != nil {
					return err
				}
				JoinerList = util.RemoveElement(JoinerList, int(n.Int64()))
			}
			for _, key = range JoinerList {
				if voterData, err = GetAddressInfo(stub, key); err != nil {
					continue
				}
				if err = MRC010Add(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
					continue
				}
				if err = SetAddressInfo(stub, key, voterData, "mrc030reward", []string{mrc030id, key, vote.Reward, strconv.Itoa(vote.RewardToken), "", "0", "", "", ""}); err != nil {
					continue
				}
				vote.Voter[key] = 1
			}
		} else { // 받을 사람을 지정
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
				JoinerList = util.RemoveElement(JoinerList, int(n.Int64()))
				if voterData, err = GetAddressInfo(stub, key); err != nil {
					continue
				}
				if err = MRC010Add(stub, &voterData, strconv.Itoa(vote.RewardToken), vote.Reward, 0); err != nil {
					continue
				}
				if err = SetAddressInfo(stub, key, voterData, "mrc030reward", []string{mrc030id, key, vote.Reward, strconv.Itoa(vote.RewardToken), "", "0", "", "", ""}); err != nil {
					continue
				}
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
			if CreatorData, err = GetAddressInfo(stub, vote.Creator); err != nil {
				return err
			}
			MRC010Add(stub, &CreatorData, strconv.Itoa(vote.RewardToken), decRefund.String(), 0)
			SetAddressInfo(stub, vote.Creator, CreatorData, "mrc030refund", []string{mrc030id, vote.Creator, decRefund.String(), strconv.Itoa(vote.RewardToken), "", "0", "", "", ""})
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

// Mrc030set : save Mrc030set
func Mrc030set(stub shim.ChaincodeStubInterface, MRC030ID string, tk mtc.MRC030, JobType string, args []string) error {
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
	if len(args) > 0 {
		if dat, err = json.Marshal(args); err == nil {
			tk.JobArgs = string(dat)
		}
	} else {
		tk.JobArgs = ""
	}

	if dat, err = json.Marshal(tk); err != nil {
		return errors.New("4204,Invalid MRC030 data format")
	}
	if err = stub.PutState(MRC030ID, dat); err != nil {
		return errors.New("8600,Hyperledger internal error - " + err.Error())
	}
	return nil
}

// Mrc030get : get MRC030
func Mrc030get(stub shim.ChaincodeStubInterface, MRC030ID string) (mtc.MRC030, error) {
	var data []byte
	var tk mtc.MRC030
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
		return tk, errors.New("4204,Invalid MRC030 data format")
	}
	return tk, nil
}

// Mrc031get : get MRC031
func Mrc031get(stub shim.ChaincodeStubInterface, MRC031ID string) (mtc.MRC031, error) {
	var data []byte
	var tk mtc.MRC031
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
