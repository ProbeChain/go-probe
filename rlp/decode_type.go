package rlp

import (
	"bytes"
	"errors"
	"github.com/probeum/go-probeum/log"
)

func ParseTypeByHead(b []byte) (byte, error) {
	s := NewStream(bytes.NewReader(b), 0)
	_, l, err := s.Kind()

	if err != nil {
		log.Error("ParseTypeByHead %d: Kind returned error: %v", l, err)
		return 0, errors.New("Parse type error!")
	}
	hs := uint64(len(b)) - l

	return b[hs : hs+1][0], nil
}

func ParseTypeByEnd(b []byte) (byte, error) {
	if len(b) > 0 {
		return b[len(b)-1 : len(b)][0], nil
	}
	return 0, errors.New(" ParseTypeByEnd Parse type error!")
}
