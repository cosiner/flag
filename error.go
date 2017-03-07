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
	errInvalidType
	errInvalidValue
	errDuplicateFlagRegister
	errStandaloneFLag
	errStandaloneValue
	errDuplicateFlagParsed
	errInvalidSelects
	errInvalidDefault
)

func (t errorType) String() string {
	switch t {
	case errNonPointer:
		return "NonPointerStructure "
	case errFlagNotFound:
		return "FlagNotFound"
	case errInvalidType:
		return "InvalidType"
	case errInvalidValue:
		return "InvalidValue"
	case errDuplicateFlagRegister:
		return "DuplicateFlagRegister"
	case errStandaloneFLag:
		return "StandaloneFLag"
	case errStandaloneValue:
		return "StandaloneValue"
	case errDuplicateFlagParsed:
		return "DuplicateFlagParsed"
	case errInvalidSelects:
		return "InvalidSelects"
	default:
		return "UnknownError"
	}
}

func (e flagError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Value)
}

func newErrorf(t errorType, format string, v ...interface{}) error {
	return flagError{
		Type:  t,
		Value: fmt.Sprintf(format, v...),
	}
}
