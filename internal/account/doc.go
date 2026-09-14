// Package account holds demand-deposit accounts.
//
// A customer identity is not an account. Opening a demand-deposit
// account, freezing it, and projecting balances from the ledger belong here.
// Cached balance_cents is updated in the same transaction as posting.
package account
