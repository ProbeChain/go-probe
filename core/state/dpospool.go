package state

import (
	"container/list"
	"github.com/probeum/go-probeum/common"
	"math/big"
	"strings"
	"sync"
)

type SortedLinkedList struct {
	*list.List
	Limit       int
	compareFunc func(old, new interface{}) bool
	lock        *sync.RWMutex
}

func NewSortedLinkedList(limit int, compare func(old, new interface{}) bool) *SortedLinkedList {
	return &SortedLinkedList{list.New(), limit, compare, new(sync.RWMutex)}
}

func (this SortedLinkedList) findInsertPlaceElement(value interface{}) *list.Element {
	for element := this.Front(); element != nil; element = element.Next() {
		tempValue := element.Value
		if this.compareFunc(tempValue, value) {
			return element
		}
	}
	return nil
}
func (this SortedLinkedList) PutOnTop(value interface{}) {
	defer this.lock.Unlock()
	this.lock.Lock()
	if this.List.Len() == 0 {
		this.PushFront(value)
		return
	}
	if this.List.Len() < this.Limit && this.compareFunc(value, this.Back().Value) {
		this.PushBack(value)
		return
	}
	if this.compareFunc(this.List.Front().Value, value) {
		this.PushFront(value)
	} else if this.compareFunc(this.List.Back().Value, value) && this.compareFunc(value, this.Front().Value) {
		element := this.findInsertPlaceElement(value)
		if element != nil {
			this.InsertBefore(value, element)
		}
	}
	if this.Len() > this.Limit {
		this.Remove(this.Back())
	}
}

func (this SortedLinkedList) remove(value interface{}) {
	defer this.lock.Unlock()
	this.lock.Lock()
	if this.List.Len() == 0 {
		return
	}
	var next *list.Element
	for e := this.List.Front(); e != nil; e = next {
		next = e.Next()
		if strings.EqualFold(e.Value.(DPoSCandidateAccount).Owner.Hex(), value.(DPoSCandidateAccount).Owner.Hex()) {
			this.Remove(e)
		}
	}
}

func (this SortedLinkedList) removeByHeigh(heigh *big.Int) {
	defer this.lock.Unlock()
	this.lock.Lock()
	if this.List.Len() == 0 {
		return
	}
	var next *list.Element
	for e := this.List.Front(); e != nil; e = next {
		next = e.Next()
		if e.Value.(DPoSCandidateAccount).Height.Cmp(heigh) < 0 {
			this.Remove(e)
		}
	}
}

func (this SortedLinkedList) Update(value interface{}) {
	defer this.lock.Unlock()
	this.lock.Lock()
	if this.List.Len() == 0 {
		return
	}
	var next *list.Element
	for e := this.List.Front(); e != nil; e = next {
		next = e.Next()
		if strings.EqualFold(e.Value.(DPoSCandidateAccount).Owner.Hex(), value.(DPoSCandidateAccount).Owner.Hex()) {
			//this.Remove(e)
			//this.PutOnTop(value)
			e.Value = value
		}
	}
}

func (this SortedLinkedList) GetDpostList() []common.DPoSAccount {
	defer this.lock.Unlock()
	this.lock.Lock()
	var dPoSAccounts = make([]common.DPoSAccount, this.Limit)
	i := 0
	for element := this.Front(); element != nil; element = element.Next() {
		dPoSCandidateAccount := element.Value.(DPoSCandidateAccount)
		dPoSAccount := &common.DPoSAccount{dPoSCandidateAccount.Enode, dPoSCandidateAccount.Owner}
		dPoSAccounts[i] = *dPoSAccount
		i++
	}
	return dPoSAccounts
}

func compareValue(old, new interface{}) bool {
	cmpRet := new.(DPoSCandidateAccount).DelegateValue.Cmp(old.(DPoSCandidateAccount).DelegateValue)
	if cmpRet == 0 {
		return new.(DPoSCandidateAccount).Weight.Cmp(old.(DPoSCandidateAccount).Weight) > 0
	}
	return cmpRet > 0
}
