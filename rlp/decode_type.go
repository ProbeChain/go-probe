package rlp

import (
	"bytes"
	"errors"
	"github.com/probeum/go-probeum/common"
	"github.com/probeum/go-probeum/log"
)

func ParseTypeByHead(b []byte) (byte, error) {
	s := NewStream(bytes.NewReader(b), 0)
	_, l, err := s.Kind()

	if err != nil {
		log.Error("ParseTypeByHead %d: Kind returned error: %v", l, err)
		return common.ACC_TYPE_OF_UNKNOWN, errors.New("unsupported account type")
	}
	hs := uint64(len(b)) - l

	return b[hs : hs+1][0], nil
}

func ParseTypeByEnd(b []byte) (byte, error) {
	if len(b) > 0 {
		return b[len(b)-1 : len(b)][0], nil
	}
	return common.ACC_TYPE_OF_UNKNOWN, errors.New("unsupported account type")
}
