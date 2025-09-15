package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/nathejk/shared-go/types"
	"nathejk.dk/nathejk/table/patrulje"
	"nathejk.dk/nathejk/table/payment"
	"nathejk.dk/nathejk/table/personnel"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type PaymentInterface interface {
	GetAll(types.TeamID) ([]payment.Payment, payment.Metadata, error)
	GetByReference(string) (*payment.Payment, error)
	AmountPaidByTeamID(types.TeamID) int
}
type PersonnelInterface interface {
	GetAll(context.Context, personnel.Filter) ([]personnel.Staff, error)
	GetByID(context.Context, types.UserID) (*personnel.Staff, error)
}
type PatruljeInterface interface {
	GetAll(context.Context, patrulje.Filter) ([]*patrulje.Patrulje, error)
	GetByID(context.Context, types.TeamID) (*patrulje.Patrulje, error)
}

type Models struct {
	Teams interface {
		GetStartedTeamIDs(Filters) ([]types.TeamID, Metadata, error)
		GetDiscontinuedTeamIDs(Filters) ([]types.TeamID, Metadata, error)
		GetPatruljer(Filters) ([]*Patrulje, Metadata, error)
		GetPatrulje(types.TeamID) (*Patrulje, error)
		GetKlan(types.TeamID) (*Klan, error)
		GetContact(types.TeamID) (*Contact, error)
		RequestedSeniorCount() int
		GetLastPatruljeID() (*types.TeamID, error)
	}
	Members interface {
		GetSpejdere(Filters) ([]*Spejder, Metadata, error)
		GetSeniore(Filters) ([]*Senior, Metadata, error)
		GetInactive(Filters) ([]*SpejderStatus, Metadata, error)
	}
	Permissions interface {
		AddForUser(int64, ...string) error
		GetAllForUser(int64) (Permissions, error)
	}
	Tokens interface {
		New(userID int64, ttl time.Duration, scope string) (*Token, error)
		Insert(token *Token) error
		DeleteAllForUser(scope string, userID int64) error
	}
	Users interface {
		Insert(*User) error
		GetByEmail(string) (*User, error)
		Update(*User) error
		GetForToken(string, string) (*User, error)
	}
	Signup interface {
		GetByID(types.TeamID) (*Signup, error)
		ConfirmBySecret(string) (types.TeamID, error)
	}
	Payment   PaymentInterface
	Personnel PersonnelInterface
	Patrulje  PatruljeInterface
}

func NewModels(db *sql.DB, payment PaymentInterface, personnel PersonnelInterface, patrulje PatruljeInterface) Models {
	return Models{
		Teams:       TeamModel{DB: db},
		Members:     MemberModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Users:       UserModel{DB: db},
		Signup:      SignupModel{DB: db},
		Payment:     payment,
		Personnel:   personnel,
		Patrulje:    patrulje,
	}
}
