// Package account holds demand-deposit accounts and same-currency jars.
//
// A customer identity is not an account. Opening a spend balance, a jar,
// freezing it, and projecting balances from the ledger belong here.
// Cached balance_cents is updated in the same transaction as posting.
package account
