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

// BizType is probe business transaction type
const (
	TRANSFER              = byte(1)  //Transfer transaction
	CONTRACT_DEPLOY       = byte(2)  //Contract deployment
	REGISTER_PNS          = byte(3)  //Register PNS account
	REGISTER_AUTHORIZE    = byte(4)  //Registered authorized account
	REGISTER_LOSE         = byte(5)  //Registered loss reporting account
	CANCELLATION          = byte(6)  //Cancellation of account
	VOTE                  = byte(7)  //vote
	APPLY_TO_BE_DPOS_NODE = byte(8)  //Apply to become a dpos node
	REDEMPTION            = byte(9)  //Redemption vote
	SEND_LOSS_REPORT      = byte(10) //Application for loss reporting
	REVEAL_LOSS_REPORT    = byte(11) //Loss report announcement
	TRANSFER_LOST_ACCOUNT = byte(12) //Transfer assets of loss reporting account
	REMOVE_LOSS_REPORT    = byte(13) //Delete the loss report application initiated without revealing the contents
	REJECT_LOSS_REPORT    = byte(14) //Refusal to report loss
	MODIFY_PNS_OWNER      = byte(15) //Modify PNS account owner
	MODIFY_PNS_CONTENT    = byte(16) //Modify PNS content
)

const (
	SPECIAL_ADDRESS_FOR_REGISTER_PNS          = "0x0000000000000000000000000000000000000001"
	SPECIAL_ADDRESS_FOR_REGISTER_AUTHORIZE    = "0x0000000000000000000000000000000000000002"
	SPECIAL_ADDRESS_FOR_REGISTER_LOSE         = "0x0000000000000000000000000000000000000003"
	SPECIAL_ADDRESS_FOR_CANCELLATION          = "0x0000000000000000000000000000000000000004"
	SPECIAL_ADDRESS_FOR_VOTE                  = "0x0000000000000000000000000000000000000005"
	SPECIAL_ADDRESS_FOR_APPLY_TO_BE_DPOS_NODE = "0x0000000000000000000000000000000000000006"
	SPECIAL_ADDRESS_FOR_REDEMPTION            = "0x0000000000000000000000000000000000000007"
	SPECIAL_ADDRESS_FOR_SEND_LOSS_REPORT      = "0x0000000000000000000000000000000000000008"
	SPECIAL_ADDRESS_FOR_REVEAL_LOSS_REPORT    = "0x0000000000000000000000000000000000000009"
	SPECIAL_ADDRESS_FOR_TRANSFER_LOST_ACCOUNT = "0x000000000000000000000000000000000000000a"
	SPECIAL_ADDRESS_FOR_REMOVE_LOSS_REPORT    = "0x000000000000000000000000000000000000000b"
	SPECIAL_ADDRESS_FOR_REJECT_LOSS_REPORT    = "0x000000000000000000000000000000000000000c"
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_OWNER      = "0x000000000000000000000000000000000000000d"
	SPECIAL_ADDRESS_FOR_MODIFY_PNS_CONTENT    = "0x000000000000000000000000000000000000000e"
)

// account type of Probe
// 6 kinds
const (
	ACC_TYPE_OF_GENERAL   = byte(1)   //General account
	ACC_TYPE_OF_PNS       = byte(2)   //PNS account
	ACC_TYPE_OF_CONTRACT  = byte(3)   //Contract account
	ACC_TYPE_OF_AUTHORIZE = byte(4)   //Authorized account
	ACC_TYPE_OF_LOSE      = byte(5)   //Loss reporting account
	ACC_TYPE_OF_UNKNOWN   = byte(100) //Unknown account
)

const (
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR     uint64 = 10000000000000000    //0.01 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS         uint64 = 50000000000000000    //0.05 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_CONTRACT    uint64 = 100000000000000000   //0.1 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_VOTING      uint64 = 10000000000000000000 //10 PRO
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS_REPORT uint64 = 1000000000000000000  //1 PRO

	MIN_PERCENTAGE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT uint64 = 10 //The minimum reported loss amount is the percentage of the original account balance
	CYCLE_HEIGHT_OF_LOSS_TYPE                          uint64 = 1  //1 loss cycle height: (5760/day)*30day*3month=518400 blocks
	THRESHOLD_HEIGHT_OF_REMOVE_LOSS_REPORT             uint64 = 1  //Initiate the loss report without revealing the content, delete it, and the height threshold
)

const (
	LOSS_STATE_OF_INIT    = byte(0)
	LOSS_STATE_OF_APPLY   = byte(1)
	LOSS_STATE_OF_NOTICE  = byte(2)
	LOSS_STATE_OF_SUCCESS = byte(3)
)

// Check business transaction type

func CheckBizType(bizType byte) bool {

	var contain bool = false
	switch bizType {
	case REGISTER_PNS:
		contain = true
	case REGISTER_AUTHORIZE:
		contain = true
	case REGISTER_LOSE:
		contain = true
	case CANCELLATION:
		contain = true
	case TRANSFER:
		contain = true
	case CONTRACT_DEPLOY:
		contain = true
	case VOTE:
		contain = true
	case APPLY_TO_BE_DPOS_NODE:
		contain = true
	case REDEMPTION:
		contain = true
	case SEND_LOSS_REPORT:
		contain = false //The current version does not support
	case REVEAL_LOSS_REPORT:
		contain = false //The current version does not support
	case TRANSFER_LOST_ACCOUNT:
		contain = false //The current version does not support
	case REMOVE_LOSS_REPORT:
		contain = false //The current version does not support
	case REJECT_LOSS_REPORT:
		contain = false //The current version does not support
	case MODIFY_PNS_OWNER:
		contain = true
	case MODIFY_PNS_CONTENT:
		contain = true
	default:
		contain = false
	}
	return contain
}

// CheckLossType check loss report type
func CheckLossType(accType byte) bool {
	return byte(0) <= accType && accType <= byte(15)
}

// CheckRegisterAccType check allow register account type
func CheckRegisterAccType(accType byte) bool {
	switch accType {
	case ACC_TYPE_OF_GENERAL:
		return true
	case ACC_TYPE_OF_PNS:
		return true
		/*	case ACC_TYPE_OF_LOSE:
			return true*/
	case ACC_TYPE_OF_AUTHORIZE:
		return true
	default:
		return false
	}
}

// AmountOfPledgeForCreateAccount amount of pledge for create a account
func AmountOfPledgeForCreateAccount(accType byte) uint64 {
	switch accType {
	case ACC_TYPE_OF_GENERAL:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR
	case ACC_TYPE_OF_PNS:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS
	case ACC_TYPE_OF_CONTRACT:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_CONTRACT
	case ACC_TYPE_OF_AUTHORIZE:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_VOTING
	case ACC_TYPE_OF_LOSE:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS_REPORT
	default:
		return 0
	}
}
