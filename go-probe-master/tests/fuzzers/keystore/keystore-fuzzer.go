// Copyright 2019 The ProbeChain Authors
// This file is part of the ProbeChain.
//
// The ProbeChain is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The ProbeChain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the ProbeChain. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"os"

	"github.com/probechain/go-probe/accounts/keystore"
)

func Fuzz(input []byte) int {
	ks := keystore.NewKeyStore("/tmp/ks", keystore.LightScryptN, keystore.LightScryptP)

	a, err := ks.NewAccount(string(input))
	if err != nil {
		panic(err)
	}
	if err := ks.Unlock(a, string(input)); err != nil {
		panic(err)
	}
	os.Remove(a.URL.Path)
	return 1
}
