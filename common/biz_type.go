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
	//Mint															= byte(0x00)		//铸币交易
	Register                         = byte(0x00) //注册账户
	Cancellation                     = byte(0xff) //注销账户
	RevokeCancellation               = byte(0xfe) //撤销注销账户操作
	Transfer                         = byte(0x01) //转账交易
	ContractCall                     = byte(0x02) //合约调用
	ExchangeTransaction              = byte(0x11) //资产兑换
	VotingForAnAccount               = byte(0x21) //为可投票账号投票
	ApplyToBeDPoSNode                = byte(0x22) //申请成为DPoS节点
	UpdatingVotesOrData              = byte(0x23) //更新投票数据
	SendLossReport                   = byte(0x31) //发送挂失报告（申请挂失）
	RevealLossMessage                = byte(0x32) //显示链上挂失信息
	TransferLostAccountWhenTimeOut   = byte(0x33) //转移挂失账号的资产当挂失报告超时时
	TransferLostAccountWhenConfirmed = byte(0x34) //转移挂失账号的资产当挂失成功时
	RejectLossReportWhenTimeOut      = byte(0x3f) //拒绝挂失报告

	RegisterPNS                   = byte(0x20) //注册PNS账号
	ModifyPnsOwner                = byte(0x21) //修改PNS账号所有者
	ModifyPnsType                 = byte(0x22) //修改PNS类型
	ModifyPnsContent              = byte(0x23) //修改PNS内容
	CancellationPns               = byte(0x2f) //注销PNS账号
	CreateDigitalSecuritiesAssets = byte(0x30) //创建数字证券资产

	//.... ... todo 还有其它待列
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
	case SendLossReport:
		contain = true
	case ModifyPnsOwner:
		contain = true
	case ModifyPnsType:
		contain = true
	case ModifyPnsContent:
		contain = true
	case CancellationPns:
		contain = true
	case CreateDigitalSecuritiesAssets:
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

// CheckRegisterAccType check allow register account type
func CheckRegisterAccType(accType byte) bool {
	if ACC_TYPE_OF_CONTRACT == accType {
		return false
	}
	return ACC_TYPE_OF_GENERAL <= accType && accType <= ACC_TYPE_OF_DPOS_CANDIDATE
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
