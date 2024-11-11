package types

const UserContextKey = "user"

type AuthenticatedUser struct {
	Username    string
	UserID      int
	LoggedIn    bool
	AccessToken string
}

type UserCredentials struct {
	Username string
	Password string
}

type Student struct {
	Username  string
	Firstname string
	Lastname  string
	UserID    string
}

type SaveData struct {
	StudentID           int     `json:"student_id"`
	PlayerBadges        Badges  `json:"player_badges"`
	CurrentFloor        string  `json:"current_floor"`
	CurrentQuest        string  `json:"current_quest"`
	SavedScene          string  `json:"saved_scene"`
	VectorX             float32 `json:"vector_x"`
	VectorY             float32 `json:"vector_y"`
	FirstTimeInitFloor1 bool    `json:"first_time_init_floor1"`
	FirstTimeInitFloor2 bool    `json:"first_time_init_floor2"`
	FirstTimeInitFloor3 bool    `json:"first_time_init_floor3"`
}

type Badges struct {
	ShinyRock     bool `json:"shiny_rock"`
	Bowl          bool `json:"bowl"`
	Carrot        bool `json:"carrot"`
	Cake          bool `json:"cake"`
	Sword         bool `json:"sword"`
	Mushroom      bool `json:"mushroom"`
	Bucket1       bool `json:"bucket1"`
	Flask         bool `json:"flask"`
	Bucket2       bool `json:"bucket2"`
	Bucket3       bool `json:"bucket3"`
	CrystalBall   bool `json:"crystal_ball"`
	Shell         bool `json:"shell"`
	OriginalRobot bool `json:"original_robot"`
}
