package state

import (
	"github.com/probechain/go-probe/common"
)

//validatorCandidates  validator candidate account definition of array
type validatorCandidates []common.DPoSCandidateAccount

//Swap swap element
func (d validatorCandidates) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

//Len return the element length
func (d validatorCandidates) Len() int { return len(d) }

//Less compare element
func (d validatorCandidates) Less(i, j int) bool {
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

//GetPresetDPosAccounts return preset validator node information
func (d validatorCandidates) GetPresetDPosAccounts() []common.Validator {
	if d.Len() > 0 {
		presetValidators := make([]common.Validator, d.Len())
		for i, dPosCandidate := range d {
			presetValidators[i] = common.Validator{Enode: dPosCandidate.Enode, Owner: dPosCandidate.Owner}
		}
		return presetValidators
	}
	return nil
}
