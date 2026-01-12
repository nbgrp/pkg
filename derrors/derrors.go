// SPDX-License-Identifier: BSD-3-Clause

package derrors

import (
	"errors"
)

func Join(err *error, errs ...error) { //nolint:gocritic // ptrToRefParam here is OK
	if err == nil {
		return
	}
	*err = errors.Join(append([]error{*err}, errs...)...)
}
