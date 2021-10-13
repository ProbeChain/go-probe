package common

import "errors"

func ValidateNil(data interface{}, msg string) error {
	if data == nil {
		return errors.New(msg + ` must be specified`)
	}
	return nil
}

func ValidateAccType(address *Address, targetAccType byte, msg string) error {
	accType, err := ValidAddress(*address)
	if err != nil {
		return errors.New("unsupported account type")
	}
	if accType != targetAccType {
		var suffix string
		switch targetAccType {
		case ACC_TYPE_OF_GENERAL:
			suffix = " account must be general type"
		case ACC_TYPE_OF_PNS:
			suffix = " account must be pns type"
		case ACC_TYPE_OF_ASSET:
			suffix = " account must be asset type"
		case ACC_TYPE_OF_CONTRACT:
			suffix = " account must be contract type"
		case ACC_TYPE_OF_AUTHORIZE:
			suffix = " account must be authorize type"
		case ACC_TYPE_OF_LOSE:
			suffix = " account must be lose type"
		}
		return errors.New(msg + suffix)
	}
	return nil
}

func ByteSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
