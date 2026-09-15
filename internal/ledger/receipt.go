package ledger

import (
	"strings"

	"github.com/google/uuid"
)

// ReceiptCode is the last 8 hex digits of a journal id, for short display.
// Copy/export still uses the full UUID.
func ReceiptCode(id uuid.UUID) string {
	s := strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))
	if len(s) <= 8 {
		return s
	}
	return s[len(s)-8:]
}
