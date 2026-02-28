package pob

import (
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/consensus"
	"github.com/probechain/go-probe/core/types"
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
