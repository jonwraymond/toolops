package health

import "errors"

var (
	// ErrCheckFailed indicates a health check failed.
	ErrCheckFailed = errors.New("health: check failed")

	// ErrCheckTimeout indicates a health check timed out.
	ErrCheckTimeout = errors.New("health: check timeout")

	// ErrCheckerNotFound indicates a checker was not found.
	ErrCheckerNotFound = errors.New("health: checker not found")

	// ErrNoCheckers indicates no checkers are registered.
	ErrNoCheckers = errors.New("health: no checkers registered")
)
