package pob

import (
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/consensus"
	"github.com/probeum/go-probeum/core/types"
)

var (
	ExtraVanity              = extraVanity
	ExtraSeal                = extraSeal
	DiffInTurn               = diffInTurn
	ErrUnauthorizedValidator = errUnauthorizedValidator
	ErrRecentlySigned        = errRecentlySigned
)

type ValidatorsAscending = validatorsAscending

func (c *ProofOfBehavior) SetFakeDiff(v bool) {
	c.fakeDiff = v
}

func (c *ProofOfBehavior) Snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	return c.snapshot(chain, number, hash, parents)
}

func SnapshotValidators(s *Snapshot) []common.Address {
	return s.validators()
}
