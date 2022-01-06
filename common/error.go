// Copyright 2014 The go-probeum Authors
// This file is part of the go-probeum library.
//
// The go-probeum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-probeum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-probeum library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"errors"
)

var (
	//ErrReservedAddress is returned if use system reserved address
	ErrReservedAddress = errors.New("system reserved address")

	//ErrIndexOutOfBounds is returned if index out of bounds
	ErrIndexOutOfBounds = errors.New("index out of bounds")
)