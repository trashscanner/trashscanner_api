package models

type Role string

const (
	RoleAdmin     Role = "admin"
	RoleUser      Role = "user"
	RoleAnonymous Role = "anonymous"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleUser, RoleAnonymous:
		return true
	default:
		return false
	}
}
