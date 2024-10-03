package types

type Classroom struct {
	ClassroomID   string // to get around bugs, we set ID to string 
	ClassroomName string
	Section       string
	Description   string
}
