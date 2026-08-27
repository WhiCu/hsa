package user

import "errors"

var ErrUnknownRole = errors.New("user: unknown role")

type Role struct {
	slug string
}

func (r Role) String() string {
	return r.slug
}

var (
	Unknown = Role{""}
	Member  = Role{"member"}
	Admin   = Role{"admin"}
)

func RoleFromString(s string) (Role, error) {
	switch s {
	case Member.slug:
		return Member, nil
	case Admin.slug:
		return Admin, nil
	}

	return Unknown, ErrUnknownRole
}
