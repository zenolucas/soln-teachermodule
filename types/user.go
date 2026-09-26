package types

type UserCredentials struct {
	Username string
	Password string
}

// Teacher is the signed-in teacher's identity for the shell (greeting, sidebar footer).
// Firstname is often empty - the register form only asks for a username (see 02 "Facts
// about the current data model").
type Teacher struct {
	UserID    int
	Username  string
	Firstname string
}

type Student struct {
	Username  string
	Firstname string
	Lastname  string
	UserID    string
	// OtherClass is the display name of another classroom this student is already
	// enrolled in, or "" if none. Only meaningful on GetUnenrolledStudents' results
	// (see 02 §C6, DEC-25's one-classroom-per-student rule).
	OtherClass string
}
