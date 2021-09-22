package state

import (
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

type dposList struct {
	// DPoSAccount DPoS账户 64
	dPoSAccounts []DPoSAccount
	// DPoSCandidateAccount DPoS候选账户 64
	dPoSCandidateAccounts []DPoSCandidateAccount

	// DPoSAccount DPoS账户 64
	oldDPoSAccounts []DPoSAccount
	// DPoSCandidateAccount DPoS候选账户 64
	oldDPoSCandidateAccounts []DPoSCandidateAccount
	lock                     sync.RWMutex
}

func newDposList() *dposList {
	return &dposList{
		dPoSAccounts:             make([]DPoSAccount, 64),
		dPoSCandidateAccounts:    make([]DPoSCandidateAccount, 64),
		oldDPoSAccounts:          make([]DPoSAccount, 64),
		oldDPoSCandidateAccounts: make([]DPoSCandidateAccount, 64),
	}
}

func (s *dposList) GetAllDPos() []DPoSAccount {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.dPoSAccounts
}

func (s *dposList) AddDPos(dDoSAccount DPoSAccount) {
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

	return s.dPoSCandidateAccounts
}

func (s *dposList) AddDPoSCandidate(account DPoSCandidateAccount) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.dPoSCandidateAccounts = append(s.dPoSCandidateAccounts, account)
	//sort.Sort(accountsByURL(liveList))
}

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
}

func (s *dposList) GetAllOldDPoS() []DPoSAccount {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.oldDPoSAccounts
}

func (s *dposList) AddOldDPoS(account DPoSAccount) {
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
