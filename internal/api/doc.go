// Package api is Qnetward's HTTP and WebSocket API: authentication and
// sessions, workspaces, probes, history, alert rules, and admin endpoints.
//
// Handlers are methods on *Server, which New builds from injected
// dependencies (store, cache, object storage, mailer) and options. Nothing in
// the package reads global state, so tests build their own instance.
package api
