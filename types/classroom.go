package types

type Classroom struct {
	// TODO: MAKE CLASSROOM_ID A UUID
	ClassroomID   string // to get around bugs, we set ID to string 
	ClassroomName string
	Section       string
	Description   string
}
