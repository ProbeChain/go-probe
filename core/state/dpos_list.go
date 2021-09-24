package state

import (
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

type dposList struct {
	// DPoSAccount DPoS账户 64
	dPoSAccounts []common.DPoSAccount
	// DPoSCandidateAccount DPoS候选账户 64
	dPoSCandidateAccounts *SortedLinkedList

	// DPoSAccount DPoS账户 64
	oldDPoSAccounts []common.DPoSAccount
	lock            sync.RWMutex
}

func newDposList() *dposList {
	return &dposList{
		dPoSAccounts:          make([]common.DPoSAccount, 64),
		dPoSCandidateAccounts: NewSortedLinkedList(64, compareValue),
		oldDPoSAccounts:       make([]common.DPoSAccount, 64),
	}
}

func (s *dposList) GetAllDPos() []common.DPoSAccount {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.dPoSAccounts
}

func (s *dposList) AddDPos(dDoSAccount common.DPoSAccount) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.dPoSAccounts = append(s.dPoSAccounts, dDoSAccount)
	//sort.Sort(accountsByURL(liveList))
}

func (s *dposList) DeleteDPosByAddr(addr common.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var i int
	for j, d := range s.dPoSAccounts {
		if d.Owner == addr {
			i = j
		}
	}
	s.dPoSAccounts = append(s.dPoSAccounts[:i], s.dPoSAccounts[i+1:]...)
}

func (s *dposList) GetAllDPoSCandidate() []DPoSCandidateAccount {
	s.lock.Lock()
	defer s.lock.Unlock()
	var dPoSCandidateAccounts = make([]DPoSCandidateAccount, s.dPoSCandidateAccounts.Limit)
	i := 0
	for element := s.dPoSCandidateAccounts.List.Front(); element != nil; element = element.Next() {
		dPoSCandidateAccounts[i] = element.Value.(DPoSCandidateAccount)
		i++
	}
	return dPoSCandidateAccounts
}

func (s *StateDB) getNextDPOSList() []common.DPoSAccount {
	var dPoSAccounts = make([]common.DPoSAccount, s.dPoSCandidateList.Limit)
	i := 0
	for element := s.dPoSCandidateList.List.Front(); element != nil; element = element.Next() {
		dPoSCandidateAccount := element.Value.(DPoSCandidateAccount)
		dPoSAccount := &common.DPoSAccount{dPoSCandidateAccount.Enode, dPoSCandidateAccount.Owner}
		dPoSAccounts[i] = *dPoSAccount
		i++
	}
	return dPoSAccounts
}

func (s *dposList) AddDPoSCandidate(account DPoSCandidateAccount) {
	s.dPoSCandidateAccounts.PutOnTop(account)
}

/*
func (s *dposList) DeleteDPoSCandidateByAddr(addr common.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var i int
	for j, d := range s.dPoSCandidateAccounts {
		if d.Owner == addr {
			i = j
		}
	}
	s.dPoSCandidateAccounts = append(s.dPoSCandidateAccounts[:i], s.dPoSCandidateAccounts[i+1:]...)
}

func (s *dposList) GetAllOldDPoSCandidate() []DPoSCandidateAccount {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.oldDPoSCandidateAccounts
}

func (s *dposList) AddOldDPoSCandidate(account DPoSCandidateAccount) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.oldDPoSCandidateAccounts = append(s.oldDPoSCandidateAccounts, account)
	//sort.Sort(accountsByURL(liveList))
}

func (s *dposList) DeleteOldDPoSCandidateByAddr(addr common.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var i int
	for j, d := range s.oldDPoSCandidateAccounts {
		if d.Owner == addr {
			i = j
		}
	}
	s.oldDPoSCandidateAccounts = append(s.oldDPoSCandidateAccounts[:i], s.oldDPoSCandidateAccounts[i+1:]...)
}*/

func (s *dposList) GetAllOldDPoS() []common.DPoSAccount {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.oldDPoSAccounts
}

func (s *dposList) AddOldDPoS(account common.DPoSAccount) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.oldDPoSAccounts = append(s.oldDPoSAccounts, account)
	//sort.Sort(accountsByURL(liveList))
}

func (s *dposList) DeleteOldDPoSByAddr(addr common.Address) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var i int
	for j, d := range s.oldDPoSAccounts {
		if d.Owner == addr {
			i = j
		}
	}
	s.oldDPoSAccounts = append(s.oldDPoSAccounts[:i], s.oldDPoSAccounts[i+1:]...)
}
