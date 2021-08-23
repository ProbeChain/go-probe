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

// TxType is probe transaction type
type TxType byte

const (
	Mint															TxType = 0x00		//铸币交易
	RegisteredAccount      											TxType = 0x10		//注册账户
	CancellationAccount      										TxType = 0x1f		//注销账户
	Transfer 														TxType = 0x12		//转账交易
	RegisterPNS														TxType = 0x20		//注册PNS账号
	ModifyPnsOwner													TxType = 0x21		//修改PNS账号所有者
	ModifyPnsType													TxType = 0x22		//修改PNS类型
	ModifyPnsContent												TxType = 0x23		//修改PNS内容
	CancellationPns      											TxType = 0x2f		//注销PNS账号
	CreateDigitalSecuritiesAssets 									TxType = 0x30		//创建数字证券资产

	//.... ... todo 还有其它待列
)

