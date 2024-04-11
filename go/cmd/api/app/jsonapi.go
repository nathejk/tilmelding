package app

import (
	"sync"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"nathejk.dk/internal/jsonlog"
	"nathejk.dk/internal/vcs"
)

var (
	version = vcs.Version()
)

type User interface {
	ID() int64
	IsActivated() bool
	IsAnonymous() bool
}

type UserPermissions interface {
	Include(string) bool
}

type UserRepository interface {
	GetForToken(scope string, token string) (User, error)
	GetPermissions(int64) (UserPermissions, error)
}

type JsonApi struct {
	Logger *jsonlog.Logger
	wg     sync.WaitGroup
	User   UserRepository
}
