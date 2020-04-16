package flag

import "fmt"

type (
	errorType uint8
	flagError struct {
		Type  errorType
		Value string
	}
)

const (
	errNonPointer errorType = iota + 1
	errFlagNotFound
	errInvalidNames
	errInvalidType
	errInvalidValue
	errDuplicateFlagRegister
	errFlagValueNotProvided
	errNonFlagValue
	errDuplicateFlagParsed
	errInvalidSelects
	errInvalidDefault
	errInvalidStructure
)

func (t errorType) String() string {
	switch t {
	case 0:
		return "NoError"
	case errNonPointer:
		return "NonPointerStructure "
	case errFlagNotFound:
		return "FlagNotFound"
	case errInvalidNames:
		return "InvalidNames"
	case errInvalidType:
		return "InvalidType"
	case errInvalidValue:
		return "InvalidValue"
	case errDuplicateFlagRegister:
		return "DuplicateFlagRegister"
	case errFlagValueNotProvided:
		return "FlagValueNotProvided"
	case errNonFlagValue:
		return "NonFlagValue"
	case errDuplicateFlagParsed:
		return "DuplicateFlagParsed"
	case errInvalidSelects:
		return "InvalidSelects"
	case errInvalidStructure:
		return "InvalidStructure"
	default:
		return "UnknownError"
	}
}

func (e flagError) Error() string {
	return e.Value
}

func newErrorf(t errorType, format string, v ...interface{}) error {
	return flagError{
		Type:  t,
		Value: fmt.Sprintf(format, v...),
	}
}
