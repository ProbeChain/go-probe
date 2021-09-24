// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package common

// BizType is probe business transaction type
const (
	Register                 = byte(0x00) //注册账户
	Cancellation             = byte(0xff) //注销账户
	RevokeCancellation       = byte(0xfe) //撤销注销账户操作
	Transfer                 = byte(0x01) //转账交易
	ContractCall             = byte(0x02) //合约调用
	ExchangeAsset            = byte(0x11) //资产兑换
	Vote                     = byte(0x21) //投票
	ApplyToBeDPoSNode        = byte(0x22) //申请成为DPoS节点
	UpdatingVotesOrData      = byte(0x23) //更新投票数据
	Redemption               = byte(0x24) //赎回投票
	SendLossReport           = byte(0x31) //申请挂失
	RevealLossReport         = byte(0x32) //挂失公告
	TransferLostAccount      = byte(0x33) //转移挂失账号的资产
	TransferLostAssetAccount = byte(0x34) //转移挂失账号的数字证券资产
	RemoveLossReport         = byte(0x3f) //发起挂失不揭示内容删除掉
	RejectLossReport         = byte(0x3e) //拒绝挂失报告
	ModifyLossType           = byte(0x30) //修改挂失类型
	ModifyPnsOwner           = byte(0x25) //修改PNS账号所有者
	ModifyPnsContent         = byte(0x26) //修改PNS内容
)

// account type of Probe
// 6 kinds
const (
	ACC_TYPE_OF_GENERAL        = byte(0) //普通账户
	ACC_TYPE_OF_PNS            = byte(1) //PNS账户
	ACC_TYPE_OF_ASSET          = byte(2) //资产账户
	ACC_TYPE_OF_CONTRACT       = byte(3) //合约账户
	ACC_TYPE_OF_AUTHORIZE      = byte(4) //授权账户
	ACC_TYPE_OF_LOSE           = byte(5) //挂失账户
	ACC_TYPE_OF_DPOS           = byte(6) //DPoS账户
	ACC_TYPE_OF_DPOS_CANDIDATE = byte(7) //DPoS候选账户
)

const (
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR       uint64 = 2
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS           uint64 = 2
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_DIGITAL_ASSET uint64 = 2
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_CONTRACT      uint64 = 2
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_VOTING        uint64 = 2
	AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_LOSS_REPORT   uint64 = 2
	MIN_MULTIPLE_OF_PLEDGE_FOR_RETRIEVE_LOST_ACCOUNT     uint64 = 10 //最小挂失金额倍数
	CYCLE_HEIGHT_OF_LOSS_TYPE                            uint64 = 1  //挂失周期 挂失有效高度 = 挂失高度 * 挂失周期
	THRESHOLD_HEIGHT_OF_REMOVE_LOSS_REPORT               uint64 = 1  //发起挂失不揭示内容，删除掉,高度阀值
)

const (
	LOSS_STATE_OF_INIT    = byte(0)
	LOSS_STATE_OF_APPLY   = byte(1)
	LOSS_STATE_OF_NOTICE  = byte(2)
	LOSS_STATE_OF_SUCCESS = byte(3)
)

const (
	PNS_TYPE_OF_IP     = byte(0)
	PNS_TYPE_OF_PORT   = byte(1)
	PNS_TYPE_OF_DOMAIN = byte(2)
)

// Check business transaction type

func CheckBizType(bizType uint8) bool {
	var contain bool = false
	switch bizType {
	//case Mint: contain = true
	case Register:
		contain = true
	case Cancellation:
		contain = true
	case RevokeCancellation:
		contain = true
	case Transfer:
		contain = true
	case ContractCall:
		contain = true
	case ExchangeAsset:
		contain = true
	case Vote:
		contain = true
	case ApplyToBeDPoSNode:
		contain = true
	case Redemption:
		contain = true
	case SendLossReport:
		contain = true
	case RevealLossReport:
		contain = true
	case TransferLostAccount:
		contain = true
	case TransferLostAssetAccount:
		contain = true
	case RemoveLossReport:
		contain = true
	case RejectLossReport:
		contain = true
	case ModifyLossType:
		contain = true
	case ModifyPnsOwner:
		contain = true
	case ModifyPnsContent:
		contain = true
	//.... ... todo 还有其它待列
	default:
		contain = false
	}
	return contain
}

// CheckAccType check account type
func CheckAccType(accType byte) bool {
	return ACC_TYPE_OF_GENERAL <= accType && accType <= ACC_TYPE_OF_DPOS_CANDIDATE
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
	case ACC_TYPE_OF_ASSET:
		return true
	case ACC_TYPE_OF_AUTHORIZE:
		return true
	case ACC_TYPE_OF_DPOS:
		return true
	case ACC_TYPE_OF_DPOS_CANDIDATE:
		return true
	default:
		return false
	}
}

// CheckTransferAccType check allow transfer account type
func CheckTransferAccType(accType byte) bool {
	var isAllow bool
	switch accType {
	case ACC_TYPE_OF_GENERAL:
		isAllow = true
	//case ACC_TYPE_OF_ASSET:
	//	isAllow = true
	case ACC_TYPE_OF_CONTRACT:
		isAllow = true
	default:
		isAllow = false
	}
	return isAllow
}

// AmountOfPledgeForCreateAccount amount of pledge for create a account
func AmountOfPledgeForCreateAccount(accType byte) uint64 {
	switch accType {
	case ACC_TYPE_OF_GENERAL:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_REGULAR
	case ACC_TYPE_OF_PNS:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_PNS
	case ACC_TYPE_OF_ASSET:
		return AMOUNT_OF_PLEDGE_FOR_CREATE_ACCOUNT_OF_DIGITAL_ASSET
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

//CheckPnsType check pns type
func CheckPnsType(pnsType byte) bool {
	/*	switch pnsType {
		case PNS_TYPE_OF_IP:
			return true
		case PNS_TYPE_OF_PORT:
			return true
		case PNS_TYPE_OF_DOMAIN:
			return true
		default:
			return false
		}*/
	return PNS_TYPE_OF_IP <= pnsType && pnsType <= PNS_TYPE_OF_DOMAIN
}
