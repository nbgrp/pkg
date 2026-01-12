// SPDX-License-Identifier: BSD-3-Clause

package derrors

import (
	"errors"
)

func Join(err *error, errs ...error) {
	*err = errors.Join(append([]error{*err}, errs...)...)
}
