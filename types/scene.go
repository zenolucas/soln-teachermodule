package types

// MinigameKind mirrors handler's private minigameKind - kept as a separate string type
// here (rather than importing handler, which would be a cycle: handler already imports
// view packages that import types) so both the handler layer and the view layer can
// read a scene's kind without either depending on the other.
type MinigameKind string

const (
	KindFractions MinigameKind = "fractions"
	KindWorded    MinigameKind = "worded"
	KindQuiz      MinigameKind = "quiz"
)

// Scene is one of the 12 playable levels, keyed by minigame ID. This is the single
// source of truth for the name/image/blurb previously copy-pasted 12 times across
// classroom.templ's Minigames tab (see FE-13).
type Scene struct {
	MinigameID int
	Name       string
	Kind       MinigameKind
	Image      string
	Alt        string
	Blurb      string
	// Op is the scene's arithmetic operator for display (DEC-21, 02 §B5): "+" for
	// scenes 1-5 (World 1, addition), "−" (U+2212 MINUS SIGN, not a hyphen) for
	// scenes 6-11 (World 2, subtraction), "" for scene 12 (the mixed final quiz).
	Op string
}

// StatsPath is the statistics page route, the same for every scene kind now (T5.5):
// /classroom/statistics already dispatches by minigame ID regardless of kind
// (renderStatisticsPage), the way the old /statistics/fraction index route used to for
// fraction and worded alike. Callers build the query string as
// "?classroom_id=&minigameID=" (DEC-6: this is a new-style route, so classroom_id, not
// the old classroomID).
func (s Scene) StatsPath() string {
	return "/classroom/statistics"
}

// StudentStatsPath is the per-student score fragment route for this scene's kind.
// Unlike StatsPath, all three kinds have their own dedicated route here
// (/statistics/student/fraction|worded|quiz), so this is a direct three-way mapping
// rather than a fallback.
func (s Scene) StudentStatsPath() string {
	switch s.Kind {
	case KindWorded:
		return "/statistics/student/worded"
	case KindQuiz:
		return "/statistics/student/quiz"
	default:
		return "/statistics/student/fraction"
	}
}

// World groups scenes the way the game itself does, in play order. Number and Topic
// (DEC-21) are a structured form of what Title already says in prose ("World 1 -
// Addition of Fractions"), for UI that shows them separately (e.g. 1c's "World N ·
// Topic" heading, which has no other data source - see Discrepancy X12).
type World struct {
	Title  string
	Number int
	Topic  string
	Intro  string
	Scenes []Scene
}

var Worlds = []World{
	{
		Title:  "World 1 - Addition of Fractions",
		Number: 1,
		Topic:  "Addition",
		Intro:  "In the first world, the player will be introduced to a series of dialogues and mini-games centered around adding fractions. These challenges will require them to find common denominators, simplify fractions, and accurately perform addition, as well as helping them visualize fractions through interactive scenarios, building the foundation needed to advance further in the game.",
		Scenes: []Scene{
			{1, "Saisai Moving Rocks Scene", KindFractions, "/public/images/assets/saisai.png", "Saisai Moving Rocks scene", "This mini-game presents the player with a simple addition problem where the fractions are provided, and the player must input the answer.", "+"},
			{2, "Robot Ambush Scene", KindFractions, "/public/images/assets/robot_ambush.png", "Robot Ambush scene", "This mini-game presents the player with a simple addition problem where the fractions are provided, and the player must input the answer.", "+"},
			{3, "Racket Steals Scene", KindWorded, "/public/images/assets/racket.png", "Racket Steals scene", "This level simulates word problems, where players are presented with a problem statement. They must analyze the statement, input the fractions, and solve for the correct answer.", "+"},
			{4, "Racket the Blacksmith Scene", KindWorded, "/public/images/assets/racket_blacksmith.png", "Racket the Blacksmith scene", "This level simulates word problems, where players are presented with a problem statement. They must analyze the statement, input the fractions, and solve for the correct answer.", "+"},
			{5, "Snekkers Quiz Scene", KindQuiz, "/public/images/assets/snekkers.png", "Snekkers Quiz scene", "This level simulates a multiple-choice test format, where players are presented with a question and four answer choices, and they must select one correct option.", "+"},
		},
	},
	{
		Title:  "World 2 - Subtraction of Fractions",
		Number: 2,
		Topic:  "Subtraction",
		Intro:  "For the next world, the player will now be challenged with a series of mini-games that will focus on tasks and puzzles related to subtracting fractions, requiring them to apply their knowledge of finding common denominators, simplifying results, and correctly performing subtraction between fractions in order to progress further in the game.",
		Scenes: []Scene{
			{6, "Waterlogged Room 1", KindFractions, "/public/images/assets/water1.png", "Waterlogged Room 1 scene", "This level presents the player with a simple subtraction problem where the fractions are provided, and the player must input the answer.", "−"},
			{7, "Chip Scene", KindFractions, "/public/images/assets/chip.png", "Chip scene", "This level presents the player with a simple subtraction problem where the fractions are provided, and the player must input the answer.", "−"},
			{8, "Waterlogged Room 2", KindFractions, "/public/images/assets/water2.png", "Waterlogged Room 2 scene", "This level presents the player with a simple subtraction problem where the fractions are provided, and the player must input the answer.", "−"},
			{9, "Waterlogged Room 3", KindFractions, "/public/images/assets/water3.png", "Waterlogged Room 3 scene", "This level presents the player with a simple subtraction problem where the fractions are provided, and the player must input the answer.", "−"},
			{10, "Rat Scene", KindWorded, "/public/images/assets/rat.png", "Rat scene", "This level simulates word problems, where players are presented with a problem statement. They must analyze the statement, input the fractions, and solve for the correct answer.", "−"},
			{11, "Crab Quiz Scene", KindQuiz, "/public/images/assets/crab.png", "Crab Quiz scene", "This level simulates a multiple-choice test format, where players are presented with a question and four answer choices, and they must select one correct option.", "−"},
		},
	},
	{
		Title:  "World 3 - The Final Level",
		Number: 3,
		Topic:  "Final Level",
		Intro:  "In world 3, the player is faced with one final boss where he must face a quiz of everything the player has encountered so far.",
		Scenes: []Scene{
			{12, "Final Boss", KindQuiz, "/public/images/assets/final_boss.png", "Final Boss scene", "The final test before the hero saves the world. This level simulates a multiple-choice test format, where players are presented with a question and four answer choices, and they must select one correct option.", ""},
		},
	},
}

// SceneByID looks up a single scene by its minigame ID, for breadcrumbs on the
// minigame/statistics pages that only know the ID from the URL.
func SceneByID(id int) (Scene, bool) {
	for _, world := range Worlds {
		for _, scene := range world.Scenes {
			if scene.MinigameID == id {
				return scene, true
			}
		}
	}
	return Scene{}, false
}
