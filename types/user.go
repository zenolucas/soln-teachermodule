package types

const UserContextKey = "user"

type AuthenticatedUser struct {
	Username 	string
	UserID 		int
	LoggedIn 	bool 
	AccessToken	string
}

type UserCredentials struct {
	Username	string
	Password	string
}

type Student struct {
	Username	string
	UserID   	string
}