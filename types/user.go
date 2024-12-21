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
	StudentID                     int     `json:"student_id"`
	PlayerBadges                  Badges  `json:"player_badges"`
	CurrentFloor                  int     `json:"current_floor"`
	CurrentQuest                  string  `json:"current_quest"`
	SavedScene                    string  `json:"saved_scene"`
	VectorX                       float32 `json:"vector_x"`
	VectorY                       float32 `json:"vector_y"`
	RockRemoved                   bool    `json:"rock_removed"`
	DisableRockRemoved            bool    `json:"disable_rock_removed"`
	RaketSneakingQuestComplete    bool    `json:"raket_sneaking_quest_complete"`
	UnlockCaveCollision           bool    `json:"unlock_cave_collision"`
	RaketSwordComplete            bool    `json:"raket_sword_complete"`
	RaketQuestProgress            int     `json:"raket_quest_progress"`
	DoRaketBlacksmithAnimation    bool    `json:"do_raket_blacksmith_animation"`
	SwordBottom                   bool    `json:"sword_bottom"`
	SwordGuard                    bool    `json:"sword_guard"`
	SwordLowerBlade               bool    `json:"sword_lower_blade"`
	SwordMiddleBlade              bool    `json:"sword_middle_blade"`
	SwordTopBlade                 bool    `json:"sword_top_blade"`
	DisableDeadRobotQuest         bool    `json:"disable_dead_robot_quest"`
	DisableRaketStealingQuest     bool    `json:"disable_raket_stealing_quest"`
	DisableFreshDialogueQuest     bool    `json:"disable_fresh_dialogue_quest"`
	DisableWaterLogged1Quest      bool    `json:"disable_water_logged_1_quest"`
	DisableWaterLogged2Quest      bool    `json:"disable_water_logged_2_quest"`
	DisableWaterLogged3Quest      bool    `json:"disable_water_logged_3_quest"`
	DisableChipQuest              bool    `json:"disable_chip_quest"`
	DisableRatWizardTrainingQuest bool    `json:"disable_rat_wizard_training_quest"`
	FirstTimeInitFloor1           bool    `json:"first_time_init_floor1"`
	FirstTimeInitFloor2           bool    `json:"first_time_init_floor2"`
	FirstTimeInitFloor3           bool    `json:"first_time_init_floor3"`
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
