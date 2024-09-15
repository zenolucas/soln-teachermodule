package types

const UserContextKey = "user"

type AuthenticatedUser struct {
	Username 	string
	LoggedIn 	bool 
}

type UserCredentials struct {
	Username	string
	Password	string
}

type Student struct {
	Username	string
}