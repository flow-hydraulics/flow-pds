package flow_helpers

import (
	"strings"

	fvm_errors "github.com/onflow/flow-go/fvm/errors"
)

var InvalidProposalSeqNumberErrorString = fvm_errors.ErrCodeInvalidProposalSeqNumberError.String()

func IsInvalidProposalSeqNumberError(err error) bool {
	return strings.Contains(err.Error(), InvalidProposalSeqNumberErrorString)
}
