package state

import (
	"github.com/probeum/go-probeum/common"
)

//dPosCandidateAccounts  dPos candidate account definition of array
type dPosCandidateAccounts []common.DPoSCandidateAccount

//Swap swap element
func (d dPosCandidateAccounts) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

//Len return the element length
func (d dPosCandidateAccounts) Len() int { return len(d) }

//Less compare element
func (d dPosCandidateAccounts) Less(i, j int) bool {
	if d[i].VoteValue == nil && d[j].VoteValue != nil {
		return false
	}
	if d[i].VoteValue != nil && d[j].VoteValue == nil {
		return true
	}
	cmpRet := d[i].VoteValue.Cmp(d[j].VoteValue)
	if cmpRet == 0 {
		cmpRet = d[i].Owner.Hash().Big().Cmp(d[j].Owner.Hash().Big())
	}
	return cmpRet > 0
}

//GetPresetDPosAccounts return preset dPos node information
func (d dPosCandidateAccounts) GetPresetDPosAccounts() []*common.DPoSAccount {
	flag := byte(0)
	presetDPoSAccountMap := make(map[common.DposEnode]*byte)
	presetDPoSAccounts := make([]*common.DPoSAccount, 0)
	for _, dPosCandidate := range d {
		if len(presetDPoSAccountMap) == common.DPosNodeLength {
			break
		}
		existDPosCandidate := presetDPoSAccountMap[dPosCandidate.Enode]
		if existDPosCandidate == nil {
			presetDPoSAccountMap[dPosCandidate.Enode] = &flag
			presetDPoSAccounts = append(presetDPoSAccounts, &common.DPoSAccount{dPosCandidate.Enode, dPosCandidate.Owner})
		}
	}
	return presetDPoSAccounts
}
