package ledger

import (
	"testing"

	"github.com/google/uuid"
)

func TestReceiptCode(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeffff1234")
	got := ReceiptCode(id)
	if got != "FFFF1234" {
		t.Fatalf("got %s", got)
	}
}
