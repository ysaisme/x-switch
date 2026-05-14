package routing

import "errors"

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrSiteNotFound     = errors.New("site not found")
	ErrNoActiveProfile  = errors.New("no active profile")
)
