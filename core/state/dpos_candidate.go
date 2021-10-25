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

type dPosCandidateAccountList []common.DPoSCandidateAccount

var dPosCandidate *DPosCandidate

func init() {
	log.Info("DPosCandidate init")
	dPosCandidate = &DPosCandidate{
		dPosCandidateAccounts: make([]common.DPoSCandidateAccount, 0),
	}
}

func GetDPosCandidates() *DPosCandidate {
	return dPosCandidate
}

func (d dPosCandidateAccountList) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d dPosCandidateAccountList) Len() int { return len(d) }

func (d dPosCandidateAccountList) Less(i, j int) bool {
	if d[i].VoteValue == nil && d[j].VoteValue != nil {
		return false
	}
	if d[i].VoteValue != nil && d[j].VoteValue == nil {
		return true
	}
	cmpRet := d[i].VoteValue.Cmp(d[j].VoteValue)
	if cmpRet == 0 {
		cmpRet = d[i].Weight.Cmp(d[j].Weight)
	}
	return cmpRet > 0
}

func (d *DPosCandidate) GetDPosCandidateAccounts() []common.DPoSCandidateAccount {
	return d.dPosCandidateAccounts
}

func (d *DPosCandidate) GetPresetDPosAccounts() []common.DPoSAccount {
	presetLen := 0
	presetDPoSAccountMap := make(map[common.DposEnode]*common.DPoSAccount)
	for i, dPosCandidate := range d.dPosCandidateAccounts {
		if len(presetDPoSAccountMap) >= common.DposNodeLength {
			break
		}
		existDPosCandidate := presetDPoSAccountMap[dPosCandidate.Enode]
		if existDPosCandidate == nil {
			presetDPoSAccountMap[dPosCandidate.Enode] = &common.DPoSAccount{dPosCandidate.Enode, dPosCandidate.Owner}
		}
		presetLen = i
	}
	if d.dPosCandidateAccounts.Len() > 0 {
		d.dPosCandidateAccounts = d.dPosCandidateAccounts[presetLen+1:]
	}
	if len(presetDPoSAccountMap) == 0 {
		return nil
	}
	presetDPoSAccounts := make([]common.DPoSAccount, len(presetDPoSAccountMap))
	index := 0
	for _, dPoSAccount := range presetDPoSAccountMap {
		presetDPoSAccounts[index] = *dPoSAccount
		index++
	}
	return presetDPoSAccounts
}

func (d *DPosCandidate) ConvertToDPosCandidate(dposList []common.DPoSAccount) {
	if len(dposList) == 0 {
		return
	}
	dPosCandidateAccounts := make([]common.DPoSCandidateAccount, len(dposList))
	for i, dposAccount := range dposList {
		var dposCandidateAccount common.DPoSCandidateAccount
		dposCandidateAccount.Enode = dposAccount.Enode
		dposCandidateAccount.Owner = dposAccount.Owner
		dposEnode := bytes.Trim(dposAccount.Enode[:], "\x00")
		dposStr := string(dposEnode[:])
		reg := regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
		remoteIp := reg.FindAllString(string(dposStr), -1)[0]
		dposCandidateAccount.Weight = common.InetAtoN(remoteIp)
		dposCandidateAccount.VoteValue = new(big.Int).SetUint64(0)

		dPosCandidateAccounts[i] = dposCandidateAccount
	}
	d.dPosCandidateAccounts = append(d.dPosCandidateAccounts, dPosCandidateAccounts...)
	sort.Stable(d.dPosCandidateAccounts)
}

func (d *DPosCandidate) AddDPosCandidate(curNode common.DPoSCandidateAccount) {
	exist := false
	if d.dPosCandidateAccounts.Len() > 0 {
		for i, node := range d.dPosCandidateAccounts {
			if node.Vote == curNode.Vote {
				d.dPosCandidateAccounts[i] = curNode
				exist = true
				break
			}
		}
	}
	if !exist {
		d.dPosCandidateAccounts = append(d.dPosCandidateAccounts, curNode)
	}
	sort.Stable(d.dPosCandidateAccounts)
}

func (d *DPosCandidate) UpdateDPosCandidate(curNode common.DPoSCandidateAccount) {
	isUpdate := false
	if d.dPosCandidateAccounts.Len() > 0 {
		for i, node := range d.dPosCandidateAccounts {
			if node.Vote == curNode.Vote {
				d.dPosCandidateAccounts[i] = curNode
				isUpdate = true
				break
			}
		}
	}
	if isUpdate {
		sort.Stable(d.dPosCandidateAccounts)
	}
}

func (d *DPosCandidate) DeleteDPosCandidate(curNode common.DPoSCandidateAccount) {
	deleteIndex := -1
	if d.dPosCandidateAccounts.Len() > 0 {
		for i, node := range d.dPosCandidateAccounts {
			if node.Vote == curNode.Vote {
				deleteIndex = i
				break
			}
		}
	}
	if deleteIndex > -1 {
		d.dPosCandidateAccounts = append(d.dPosCandidateAccounts[:deleteIndex], d.dPosCandidateAccounts[deleteIndex+1:]...)
		//sort.Stable(d.dPosCandidateAccounts)
	}
}

func (d *DPosCandidate) compare(node1, node2 *common.DPoSCandidateAccount) int {
	if node1.Owner == node2.Owner && node1.Enode == node2.Enode {
		if node1.VoteValue == nil && node2.VoteValue != nil {
			return 1
		}
		if node1.VoteValue != nil && node2.VoteValue == nil {
			return -1
		}
		cmpRet := node1.VoteValue.Cmp(node2.VoteValue)
		if cmpRet == 0 {
			cmpRet = node1.Weight.Cmp(node2.Weight)
		}
		if cmpRet > 0 || cmpRet == 0 {
			return -1
		} else {
			return 1
		}
	}
	return 0
}

func BuildHashForDPos(accounts []common.DPoSAccount) common.Hash {
	num := big.NewInt(0)
	for _, account := range accounts {
		curNum := new(big.Int).SetBytes(crypto.Keccak512(append(account.Enode[:], account.Owner.Bytes()...)))
		num = new(big.Int).Xor(curNum, num)
	}
	return crypto.Keccak256Hash(num.Bytes())
}
