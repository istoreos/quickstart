package scope_error

import (
	"os"

	"golang.org/x/xerrors"
)

type causer interface {
	Cause() error
}

func IsNotExist(err error) bool {
	for err != nil {
		if c, ok := err.(causer); ok {
			err = c.Cause()
		} else if c2, ok := err.(xerrors.Wrapper); ok {
			err = c2.Unwrap()
		} else {
			break
		}
	}

	if err != nil {
		return os.IsNotExist(err)
	}
	return false
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return xerrors.Errorf("%s: %w", message, err)
}
