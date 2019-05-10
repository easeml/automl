package types

const (
	// UserRoot is the name of the root user. This user has no password, cannot log in or
	// log out and can only be authenticated with an API key.
	UserRoot = "root"

	// UserAnon is the user id assigned to unauthenticated users.
	UserAnon = "anonymous"

	// UserThis is the user id of the currently logged in user.
	UserThis = "this"
)

// User contains information about users.
type User struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	PasswordHash string        `json:"password,omitempty"`
}

// IsRoot returns true if the given user is the root user.
func (user User) IsRoot() bool {
	return user.ID == UserRoot
}

// IsAnon returns true if the given user is the anonymous user.
func (user User) IsAnon() bool {
	return user.ID == UserAnon
}

// GetAnonUser returns an anonymous user instance.
func GetAnonUser() User {
	return User{ID: UserAnon}
}
