// Package transfer orchestrates customer-to-customer payments.
//
// Requests must carry an idempotency key. The operation writes ledger
// entries in one database transaction and refuses frozen or closed accounts.
package transfer
