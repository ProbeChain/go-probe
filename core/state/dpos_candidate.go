package state

import (
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/crypto"
	"github.com/probeum/go-probeum/log"
	"math/big"
	"sort"
	"sync"
)

type DPosCandidate struct {
	lock                  *sync.RWMutex
	dPosCandidateAccounts dPosCandidateAccountList
}

type dPosCandidateAccountList []common.DPoSCandidateAccount

var dPosCandidate *DPosCandidate

func init() {
	log.Info("DPosCandidate init")
	dPosCandidate = &DPosCandidate{
		lock:                  new(sync.RWMutex),
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
		cmpRet = d[i].Owner.Hash().Big().Cmp(d[j].Owner.Hash().Big())
		//cmpRet = d[i].Weight.Cmp(d[j].Weight)
	}
	return cmpRet > 0
}

func (d *DPosCandidate) GetDPosCandidateAccounts() []common.DPoSCandidateAccount {
	d.lock.RLock()
	defer d.lock.RUnlock()
	return d.dPosCandidateAccounts
}

func (d *DPosCandidate) GetPresetDPosAccounts() []common.DPoSAccount {
	d.lock.Lock()
	defer d.lock.Unlock()
	sort.Sort(d.dPosCandidateAccounts)
	presetLen := 0
	flag := 1
	presetDPoSAccountMap := make(map[common.DposEnode]*int)
	presetDPoSAccounts := make([]common.DPoSAccount, 0)
	for i, dPosCandidate := range d.dPosCandidateAccounts {
		if len(presetDPoSAccountMap) >= common.DposNodeLength {
			break
		}
		existDPosCandidate := presetDPoSAccountMap[dPosCandidate.Enode]
		if existDPosCandidate == nil {
			presetDPoSAccountMap[dPosCandidate.Enode] = &flag
			presetDPoSAccounts = append(presetDPoSAccounts, common.DPoSAccount{dPosCandidate.Enode, dPosCandidate.Owner})
		}
		presetLen = i
	}
	if d.dPosCandidateAccounts.Len() > 0 {
		d.dPosCandidateAccounts = d.dPosCandidateAccounts[presetLen+1:]
	}
	if len(presetDPoSAccountMap) == 0 {
		return nil
	}
	return presetDPoSAccounts
}

func (d *DPosCandidate) AddDPosCandidate(curNode common.DPoSCandidateAccount) {
	d.lock.Lock()
	defer d.lock.Unlock()
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
}

func (d *DPosCandidate) UpdateDPosCandidate(curNode common.DPoSCandidateAccount) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.dPosCandidateAccounts.Len() > 0 {
		for i, node := range d.dPosCandidateAccounts {
			if node.Vote == curNode.Vote {
				d.dPosCandidateAccounts[i] = curNode
				break
			}
		}
	}
}

func (d *DPosCandidate) DeleteDPosCandidate(curNode common.DPoSCandidateAccount) {
	d.lock.Lock()
	defer d.lock.Unlock()
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
	}
}

func BuildHashForDPos(accounts []common.DPoSAccount) common.Hash {
	num := big.NewInt(0)
	for _, account := range accounts {
		curNum := new(big.Int).SetBytes(crypto.Keccak512(append(account.Enode[:], account.Owner.Bytes()...)))
		num = new(big.Int).Xor(curNum, num)
	}
	return crypto.Keccak256Hash(num.Bytes())
}
