package state

import (
	"github.com/probechain/go-probe/common"
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
func (d dPosCandidateAccounts) GetPresetDPosAccounts() []common.DPoSAccount {
	if d.Len() > 0 {
		presetDPoSAccounts := make([]common.DPoSAccount, d.Len())
		for i, dPosCandidate := range d {
			presetDPoSAccounts[i] = common.DPoSAccount{Enode: dPosCandidate.Enode, Owner: dPosCandidate.Owner}
		}
		return presetDPoSAccounts
	}
	return nil
}
