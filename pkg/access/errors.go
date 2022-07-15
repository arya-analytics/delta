package access

import (
	"github.com/cockroachdb/errors"
)

var Forbidden = errors.New("[access] - forbidden")
