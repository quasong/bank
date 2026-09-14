// Package transfer will orchestrate customer-to-customer payments (phase 3).
//
// Requests must carry an idempotency key. The operation writes ledger
// entries in one database transaction and refuses frozen or closed accounts.
package transfer
