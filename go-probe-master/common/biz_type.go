// Copyright 2015 The ProbeChain Authors
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

package common

var (
	//special address for business type (Address typed for byte-level comparison)
	SPECIAL_ADDRESS_FOR_REGISTER_PNS                    = BytesToAddress(FromHex("0x0000000000000000000000000000000000000101"))
	SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE              = BytesToAddress(FromHex("0x0000000000000000000000000000000000000102"))
	SPECIAL_ADDRESS_FOR_REGISTER_LOSE                   = BytesToAddress(FromHex("0x0000000000000000000000000000000000000103"))
	SPECIAL_ADDRESS_FOR_CANCELLATION                    = BytesToAddress(FromHex("0x0000000000000000000000000000000000000104"))
	SPECIAL_ADDRESS_FOR_VOTE                            = BytesToAddress(FromHex("0x0000000000000000000000000000000000000105"))
	SPECIAL_ADDRESS_FOR_APPLY_TO_BE_DPOS_NODE           = BytesToAddress(FromHex("0x0000000000000000000000000000000000000106"))
	SPECIAL_ADDRESS_FOR_REDEMPTION                      = BytesToAddress(FromHex("0x0000000000000000000000000000000000000107"))
	SPECIAL_ADDRESS_FOR_CANCELLATION_LOST_ACCOUNT       = BytesToAddress(FromHex("0x0000000000000000000000000000000000000108"))
	SPECIAL_ADDRESS_FOR_REVEAL_LOSS_REPORT              = BytesToAddress(FromHex("0x0000000000000000000000000000000000000109"))
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_BALANCE   = BytesToAddress(FromHex("0x0000000000000000000000000000000000000110"))
	SPECIAL_ADDRESS_FOR_REMOVE_LOSS_REPORT              = BytesToAddress(FromHex("0x0000000000000000000000000000000000000111"))
	SPECIAL_ADDRESS_FOR_REJECT_LOSS_REPORT              = BytesToAddress(FromHex("0x0000000000000000000000000000000000000112"))
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_OWNER                = BytesToAddress(FromHex("0x0000000000000000000000000000000000000113"))
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_CONTENT              = BytesToAddress(FromHex("0x0000000000000000000000000000000000000114"))
	SPECIAL_ADDRESS_FOR_MODIFY_LOSS_TYPE                = BytesToAddress(FromHex("0x0000000000000000000000000000000000000115"))
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_PNS       = BytesToAddress(FromHex("0x0000000000000000000000000000000000000116"))
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_AUTHORIZE = BytesToAddress(FromHex("0x0000000000000000000000000000000000000117"))
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_ASSET     = BytesToAddress(FromHex("0x0000000000000000000000000000000000000118"))
	SPECIAL_ADDRESS_FOR_DPOS                            = BytesToAddress(FromHex("0x0000000000000000000000000000000000000119"))
	SPECIAL_ADDRESS_FOR_DEX_SETTLEMENT                  = BytesToAddress(FromHex("0x000000000000000000000000000000000000011a"))
)

const (
	// account type
	ACC_TYPE_OF_REGULAR   = uint8(1)
	ACC_TYPE_OF_PNS       = uint8(2)
	ACC_TYPE_OF_CONTRACT  = uint8(3)
	ACC_TYPE_OF_AUTHORIZE = uint8(4)
	ACC_TYPE_OF_LOSS      = uint8(5)
	ACC_TYPE_OF_LOSS_MARK = uint8(6)
	ACC_TYPE_OF_DPOS      = uint8(7)
	ACC_TYPE_OF_UNKNOWN   = uint8(100)

	// pledge amount when register
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR   uint64 = 10000000000000000    //0.01 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS       uint64 = 50000000000000000    //0.05 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_CONTRACT  uint64 = 100000000000000000   //0.1 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_AUTHORIZE uint64 = 10000000000000000000 //10 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS      uint64 = 1000000000000000000  //1 PRO

	// loss reporing
	MIN_PERCENTAGE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT uint8  = 10     //min percentage of pledge for retrieve lost account
	UNSUPPORTED_OF_LOSS_TYPE                           uint8  = 0      //loss reporting is not supported
	MAX_CYCLE_HEIGHT_OF_LOSS_TYPE                      uint8  = 127    //max cycle height
	LOSS_MARK_OF_LOSS_TYPE                             bool   = true   //loss reporting mark
	CYCLE_HEIGHT_BLOCKS_OF_LOSS_TYPE                   uint64 = 172800 //1 loss cycle height: (5760/day)*30day=172800 blocks
	THRESHOLD_HEIGHT_OF_REMOVE_LOSS_REPORT             uint64 = 11520  //threshold height of remove loss report when loss report not reveal, 2 days height

	//loss state
	LOSS_STATE_OF_APPLY   uint8 = 0
	LOSS_STATE_OF_REVEAL  uint8 = 1
	LOSS_STATE_OF_SUCCESS uint8 = 2
)

// specialAddresses is the set of system-reserved addresses for byte-level lookup.
var specialAddresses = map[Address]bool{
	SPECIAL_ADDRESS_FOR_REGISTER_PNS:                    true,
	SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE:              true,
	SPECIAL_ADDRESS_FOR_REGISTER_LOSE:                   true,
	SPECIAL_ADDRESS_FOR_CANCELLATION:                    true,
	SPECIAL_ADDRESS_FOR_VOTE:                            true,
	SPECIAL_ADDRESS_FOR_APPLY_TO_BE_DPOS_NODE:           true,
	SPECIAL_ADDRESS_FOR_REDEMPTION:                      true,
	SPECIAL_ADDRESS_FOR_CANCELLATION_LOST_ACCOUNT:       true,
	SPECIAL_ADDRESS_FOR_REVEAL_LOSS_REPORT:              true,
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_BALANCE:   true,
	SPECIAL_ADDRESS_FOR_REMOVE_LOSS_REPORT:              true,
	SPECIAL_ADDRESS_FOR_REJECT_LOSS_REPORT:              true,
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_OWNER:                true,
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_CONTENT:              true,
	SPECIAL_ADDRESS_FOR_MODIFY_LOSS_TYPE:                true,
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_PNS:       true,
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_AUTHORIZE: true,
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_ASSET:     true,
	SPECIAL_ADDRESS_FOR_DPOS:                            true,
	SPECIAL_ADDRESS_FOR_DEX_SETTLEMENT:                  true,
}

//IsSpecialAddress judges system reserved address. Accepts Address type for byte-level comparison.
func IsSpecialAddress(addr Address) bool {
	return specialAddresses[addr]
}

//IsSpecialAddressString judges system reserved address from a string (hex or Bech32).
func IsSpecialAddressString(addrStr string) bool {
	return specialAddresses[HexToAddress(addrStr)]
}

//IsReservedAddress judge system reserved address
func IsReservedAddress(address Address) bool {
	return address.Hash().Big().Uint64() <= 512
}
