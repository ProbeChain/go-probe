// Copyright 2015 The go-probeum Authors
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

const (

	//special address for business type
	SPECIAL_ADDRESS_FOR_REGISTER_PNS                    = "0x0000000000000000000000000000000000000001"
	SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE              = "0x0000000000000000000000000000000000000002"
	SPECIAL_ADDRESS_FOR_REGISTER_LOSE                   = "0x0000000000000000000000000000000000000003"
	SPECIAL_ADDRESS_FOR_CANCELLATION                    = "0x0000000000000000000000000000000000000004"
	SPECIAL_ADDRESS_FOR_VOTE                            = "0x0000000000000000000000000000000000000005"
	SPECIAL_ADDRESS_FOR_APPLY_TO_BE_DPOS_NODE           = "0x0000000000000000000000000000000000000006"
	SPECIAL_ADDRESS_FOR_REDEMPTION                      = "0x0000000000000000000000000000000000000007"
	SPECIAL_ADDRESS_FOR_CANCELLATION_LOST_ACCOUNT       = "0x0000000000000000000000000000000000000008"
	SPECIAL_ADDRESS_FOR_REVEAL_LOSS_REPORT              = "0x0000000000000000000000000000000000000009"
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_BALANCE   = "0x0000000000000000000000000000000000000010"
	SPECIAL_ADDRESS_FOR_REMOVE_LOSS_REPORT              = "0x0000000000000000000000000000000000000011"
	SPECIAL_ADDRESS_FOR_REJECT_LOSS_REPORT              = "0x0000000000000000000000000000000000000012"
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_OWNER                = "0x0000000000000000000000000000000000000013"
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_CONTENT              = "0x0000000000000000000000000000000000000014"
	SPECIAL_ADDRESS_FOR_MODIFY_LOSS_TYPE                = "0x0000000000000000000000000000000000000015"
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_PNS       = "0x0000000000000000000000000000000000000016"
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_AUTHORIZE = "0x0000000000000000000000000000000000000017"
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT_ASSET     = "0x0000000000000000000000000000000000000018"

	// account type
	ACC_TYPE_OF_REGULAR   = uint8(1)
	ACC_TYPE_OF_PNS       = uint8(2)
	ACC_TYPE_OF_CONTRACT  = uint8(3)
	ACC_TYPE_OF_AUTHORIZE = uint8(4)
	ACC_TYPE_OF_LOSS      = uint8(5)
	ACC_TYPE_OF_LOSS_MARK = uint8(6)
	ACC_TYPE_OF_UNKNOWN   = uint8(100)

	// pledge amount when register
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR   uint64 = 10000000000000000    //0.01 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS       uint64 = 50000000000000000    //0.05 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_CONTRACT  uint64 = 100000000000000000   //0.1 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_AUTHORIZE uint64 = 10000000000000000000 //10 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS      uint64 = 1000000000000000000  //1 PRO

	// loss reporing
	MIN_PERCENTAGE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT uint8  = 10   //min percentage of pledge for retrieve lost account
	UNSUPPORTED_OF_LOSS_TYPE                           uint8  = 0    //loss reporting is not supported
	MAX_CYCLE_HEIGHT_OF_LOSS_TYPE                      uint8  = 127  //max cycle height
	LOSS_MARK_OF_LOSS_TYPE                             bool   = true //loss reporting mark
	CYCLE_HEIGHT_BLOCKS_OF_LOSS_TYPE                   uint64 = 50   //1 loss cycle height: (5760/day)*30day=172800 blocks  todo
	THRESHOLD_HEIGHT_OF_REMOVE_LOSS_REPORT             uint64 = 20   //threshold height of remove loss report when loss report not reveal todo 两天的高度

	//loss state
	LOSS_STATE_OF_APPLY   uint8 = 0
	LOSS_STATE_OF_REVEAL  uint8 = 1
	LOSS_STATE_OF_SUCCESS uint8 = 2
)
