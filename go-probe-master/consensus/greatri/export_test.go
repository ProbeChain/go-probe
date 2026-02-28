package greatri

import (
	"github.com/probechain/go-probe/common"
	"github.com/probechain/go-probe/consensus"
	"github.com/probechain/go-probe/core/types"
)

var (
	ExtraVanity          = extraVanity
	ExtraSeal            = extraSeal
	DiffInTurn           = diffInTurn
	NonceAuthVote        = nonceAuthVote
	ErrUnauthorizedSigner = errUnauthorizedSigner
	ErrRecentlySigned    = errRecentlySigned
)

type SignersAscending = signersAscending

func (c *Greatri) SetFakeDiff(v bool) {
	c.fakeDiff = v
}

func (c *Greatri) Snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	return c.snapshot(chain, number, hash, parents)
}

func SnapshotSigners(s *Snapshot) []common.Address {
	return s.signers()
}
