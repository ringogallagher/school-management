// Package memrepo holds in-memory implementations of the repository interfaces.
// They let the HTTP integration tests (internal/app) run the real router, handlers,
// middleware and services without a database. The PostgreSQL variant of the same
// tests lives behind the "integration" build tag.
package memrepo
