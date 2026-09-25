package types

import "time"

// Shared thresholds for the insight logic (DEC-20). Views and handlers import these,
// so no literal 60 appears in markup logic. There's deliberately no WrongStreakMin:
// DEC-24 drops the wrong_streak flag until the pending game-client audit resolves V1.
const (
	PassPct                = 60
	MinAttemptsForAccuracy = 5
	MinResponsesForHint    = 5
	StudentsPageSize       = 10
)

// FlagKind identifies why a student was flagged for teacher attention. wrong_streak
// is deliberately absent (DEC-24).
type FlagKind string

const (
	FlagQuizBelow60 FlagKind = "quiz_below_60"
	FlagLowAccuracy FlagKind = "low_accuracy"
)

// Flag is one reason a student needs attention, scoped to a single scene.
type Flag struct {
	Kind       FlagKind
	MinigameID int
	Detail     string
}

// QuizScoreRow is one student's latest attempt at one quiz minigame (DEC-9: latest
// attempt per student per quiz, not best).
type QuizScoreRow struct {
	StudentID  int
	MinigameID int
	Score      int
	Total      int
}

// FractionAggRow is one student's summed right/wrong attempts on one fraction or
// worded minigame.
type FractionAggRow struct {
	StudentID  int
	MinigameID int
	Right      int
	Wrong      int
}

// Activity is one Recent activity event (02 §D1, 01 §1a). Kind is "played" (a
// fraction or worded scene - DEC-24 rules out "finished"/"started" copy) or "scored"
// (a quiz attempt). Score and Total are only meaningful when Kind == "scored".
type Activity struct {
	UserID      int
	First, Last string
	ClassroomID int
	ClassName   string
	MinigameID  int
	Kind        string
	Score       int
	Total       int
	At          time.Time
}

// StudentInsight is one enrolled student's progress, quiz/accuracy averages, and
// flags within a classroom (02 §C2/§C3). QuizAvgPct and AccuracyPct are -1 when the
// student has no rows to average - not 0, which would read as a real 0%.
type StudentInsight struct {
	Student
	Current     int
	Completed   bool
	World       int
	QuizAvgPct  int
	AccuracyPct int
	Flags       []Flag
}

// SceneSummary is one minigame's rollup within a classroom (02 §C4), for the
// Minigames page cards and the Overview's per-world progress. AccuracyPct applies to
// fraction/worded scenes and QuizAvgPct to quiz scenes - the other is always -1 on a
// given scene, same as any field with no data to compute from.
type SceneSummary struct {
	MinigameID    int
	QuestionCount int
	AccuracyPct   int
	QuizAvgPct    int
	CompletionPct int
	TookCount     int
}

// ClassroomSummary is a classroom's rollup for the Home cards and Overview stats
// (02 §C3). QuizAvgPct is -1 when no student has taken a quiz yet. PerWorld is
// indexed 1..3 (World.Number); index 0 is unused, kept so World.Number can index it
// directly without an off-by-one.
type ClassroomSummary struct {
	Classroom
	StudentCount int
	QuizAvgPct   int
	FlaggedCount int
	ReachedW3    int
	PerWorld     [4]int
}

// ChoiceCount is one multiple-choice question's answer choice, with how many
// enrolled students picked it across every attempt (02 §C8: individual responses
// aren't tied to a specific attempt without timestamps, so this counts all of them,
// not just each student's latest).
type ChoiceCount struct {
	ChoiceID  int
	Text      string
	IsCorrect bool
	Count     int
}

// StudentScore is one enrolled student's latest attempt at a quiz (DEC-9).
type StudentScore struct {
	UserID      int
	First, Last string
	Score       int
	Total       int
}

// QuizSummary is a quiz scene's classroom-wide rollup (02 §C8, 01 §1g's four stat
// cards). Histogram is indexed by raw score, 0..Total inclusive (len(Histogram) ==
// Total+1), each entry the count of students with that exact score.
type QuizSummary struct {
	AvgPct    int
	AvgScore  float64
	Median    float64
	Took      int
	Enrolled  int
	Below     int
	Histogram []int
}

// StudentAccuracy is one enrolled student's summed right/wrong attempts on a single
// fraction/worded minigame (02 §C9), for the Students card on that scene's
// statistics page.
type StudentAccuracy struct {
	UserID      int
	First, Last string
	Right       int
	Wrong       int
}
