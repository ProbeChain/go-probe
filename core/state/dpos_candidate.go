package state

import (
	"bytes"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/log"
	"math/big"
	"regexp"
	"sort"
)

type DPosCandidate struct {
	// DPoSCandidateAccount DPoS候选账户 64
	dPosCandidateAccounts dPosCandidateAccountList
}

type dPosCandidateAccountList []DPoSCandidateAccount

var dPosCandidate *DPosCandidate

func init() {
	log.Info("DPosCandidate init")
	dPosCandidate = &DPosCandidate{
		dPosCandidateAccounts: make([]DPoSCandidateAccount, 0),
	}
}

func GetDPosCandidates() *DPosCandidate {
	return dPosCandidate
}

func (d dPosCandidateAccountList) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d dPosCandidateAccountList) Len() int { return len(d) }

func (d dPosCandidateAccountList) Less(i, j int) bool {
	if d[i].DelegateValue == nil && d[j].DelegateValue != nil {
		return false
	}
	if d[i].DelegateValue != nil && d[j].DelegateValue == nil {
		return true
	}
	cmpRet := d[i].DelegateValue.Cmp(d[j].DelegateValue)
	if cmpRet == 0 {
		cmpRet = d[i].Weight.Cmp(d[j].Weight)
	}
	return cmpRet > 0
}

func (d *DPosCandidate) GetDPosCandidateAccounts() *dPosCandidateAccountList {
	return &d.dPosCandidateAccounts
}

func (d *DPosCandidate) GetPresetDPosAccounts(del bool) ([]common.DPoSAccount, bool) {
	presetLen := d.dPosCandidateAccounts.Len()
	if presetLen > common.DposNodeLength {
		presetLen = common.DposNodeLength
	}
	hasNew := false
	presetDPoSAccounts := make([]common.DPoSAccount, presetLen)
	for i := 0; i < presetLen; i++ {
		dPosCandidate := d.dPosCandidateAccounts[i]
		if dPosCandidate.Mark == byte(0) {
			hasNew = true
			d.dPosCandidateAccounts[i].Mark = byte(1)
		}
		presetDPoSAccounts[i] = common.DPoSAccount{dPosCandidate.Enode, dPosCandidate.Owner}
	}
	if del {
		d.dPosCandidateAccounts = d.dPosCandidateAccounts[presetLen:]
	}
	if presetLen == 0 {
		presetDPoSAccounts = nil
	}
	return presetDPoSAccounts, hasNew
}

func (d *DPosCandidate) ConvertToDPosCandidate(dposList []common.DPoSAccount) {
	if len(dposList) == 0 {
		return
	}
	dPosCandidateAccounts := make([]DPoSCandidateAccount, len(dposList))
	for i, dposAccount := range dposList {
		var dposCandidateAccount DPoSCandidateAccount
		dposCandidateAccount.Enode = dposAccount.Enode
		dposCandidateAccount.Owner = dposAccount.Owner
		dposEnode := bytes.Trim(dposAccount.Enode[:], "\x00")
		dposStr := string(dposEnode[:])
		reg := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
		remoteIp := reg.FindAllString(string(dposStr), -1)[0]
		dposCandidateAccount.Weight = common.InetAtoN(remoteIp)
		dposCandidateAccount.Mark = byte(0)
		dposCandidateAccount.DelegateValue = new(big.Int).SetUint64(0)

		dPosCandidateAccounts[i] = dposCandidateAccount
	}
	d.dPosCandidateAccounts = append(d.dPosCandidateAccounts, dPosCandidateAccounts...)
	sort.Stable(d.dPosCandidateAccounts)
}

func (d *DPosCandidate) AddDPosCandidate(dPosCandidate DPoSCandidateAccount) {
	d.dPosCandidateAccounts = append(d.dPosCandidateAccounts, dPosCandidate)
	sort.Stable(d.dPosCandidateAccounts)
}

func BuildHashForDPos(accounts []common.DPoSAccount) common.Hash {
	num := big.NewInt(0)
	for _, account := range accounts {
		curNum := new(big.Int).SetBytes(crypto.Keccak512(append(account.Enode[:], account.Owner.Bytes()...)))
		num = new(big.Int).Xor(curNum, num)
	}
	return crypto.Keccak256Hash(num.Bytes())
}

func BuildHashForDPosCandidate(accounts []DPoSCandidateAccount) common.Hash {
	num := big.NewInt(0)
	for _, account := range accounts {
		bytes1 := append(account.Enode[:], account.Owner.Bytes()...)
		bytes2 := append(account.Weight.Bytes(), account.DelegateValue.Bytes()...)
		bytes3 := append(bytes1, bytes2...)
		curNum := new(big.Int).SetBytes(crypto.Keccak512(bytes3))
		num = new(big.Int).Xor(curNum, num)
	}
	//hash := make([]byte, 32, 64)        // 哈希出来的长度为32byte
	//hash = append(hash, num.Bytes()...) // 前面不足的补0，一共返回32位
	//
	//var ret [32]byte
	//copy(ret[:], hash[32:64])

	return crypto.Keccak256Hash(num.Bytes())
}
