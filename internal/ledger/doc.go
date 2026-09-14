// Package ledger will be the source of truth for money (phase 2+).
//
// Customer deposits are liabilities of the bank. A transfer is a pair of
// journal lines, not an UPDATE of two balance columns. Amounts are integer
// minor units (cents); never float64.
package ledger
