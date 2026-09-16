package editiontest_test

import (
	"testing"

	"github.com/gokuai/yunku-cli/pkg/edition"
	"github.com/gokuai/yunku-cli/pkg/editiontest"
)

func TestOpenSourceDefaultHooksContract(t *testing.T) {
	editiontest.RunContractTests(t, edition.Get())
}
