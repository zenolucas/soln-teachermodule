package handler

import (
	"fmt"
	"strings"

	"soln-teachermodule/types"
)

// sceneOrder is the play order the game forces (DEC-10's V2 answer), flattened once
// from types.Worlds so currentScene/finishedCount don't rebuild it per student.
var sceneOrder = func() []int {
	var ids []int
	for _, world := range types.Worlds {
		for _, scene := range world.Scenes {
			ids = append(ids, scene.MinigameID)
		}
	}
	return ids
}()

// sceneLabel is the scene's display name with the redundant " Scene" suffix trimmed,
// e.g. "Crab Quiz Scene" -> "Crab Quiz" (used in flag detail copy, 02 §C2).
func sceneLabel(scene types.Scene) string {
	return strings.TrimSuffix(scene.Name, " Scene")
}

// EvaluateFlags applies the two DEC-24 flag rules (quiz_below_60, low_accuracy) to a
// classroom's raw rows and returns each flagged student's flags, keyed by student ID.
// There's no wrong_streak rule: DEC-24 defers it to a pending game-client audit.
func EvaluateFlags(quiz []types.QuizScoreRow, frac []types.FractionAggRow) map[int][]types.Flag {
	flags := map[int][]types.Flag{}
	for _, q := range quiz {
		if q.Total <= 0 || q.Score*100 >= types.PassPct*q.Total {
			continue
		}
		scene, _ := types.SceneByID(q.MinigameID)
		detail := fmt.Sprintf("%s · %d/%d", sceneLabel(scene), q.Score, q.Total)
		flags[q.StudentID] = append(flags[q.StudentID], types.Flag{
			Kind: types.FlagQuizBelow60, MinigameID: q.MinigameID, Detail: detail,
		})
	}
	for _, f := range frac {
		total := f.Right + f.Wrong
		if total < types.MinAttemptsForAccuracy || f.Right*100 >= types.PassPct*total {
			continue
		}
		scene, _ := types.SceneByID(f.MinigameID)
		detail := fmt.Sprintf("%s · %d%% correct", sceneLabel(scene), f.Right*100/total)
		flags[f.StudentID] = append(flags[f.StudentID], types.Flag{
			Kind: types.FlagLowAccuracy, MinigameID: f.MinigameID, Detail: detail,
		})
	}
	return flags
}

// FlagLabel is the short label shown next to a flag badge.
func FlagLabel(kind types.FlagKind) string {
	switch kind {
	case types.FlagQuizBelow60:
		return "Quiz below 60%"
	case types.FlagLowAccuracy:
		return "Low accuracy"
	default:
		return ""
	}
}

// currentScene implements DEC-10's progress rule, which replaces 02 §C1's "≥1 row =
// finished" rule now that V2 has confirmed the game forces scene order:
//   - current = the last scene in types.Worlds order that has any row (played)
//   - if that scene is a quiz with a score row, current advances to the next scene;
//     if that quiz is scene 12, the student is completed instead
//   - no rows at all means current = the first scene
//
// Every scene before current is finished, since play order is forced - see
// finishedCount.
func currentScene(played, quizScored map[int]bool) (minigameID int, completed bool) {
	lastIdx := -1
	for i, id := range sceneOrder {
		if played[id] {
			lastIdx = i
		}
	}
	if lastIdx == -1 {
		return sceneOrder[0], false
	}
	last := sceneOrder[lastIdx]
	scene, _ := types.SceneByID(last)
	if scene.Kind == types.KindQuiz && quizScored[last] {
		if lastIdx+1 >= len(sceneOrder) {
			return last, true
		}
		return sceneOrder[lastIdx+1], false
	}
	return last, false
}

// finishedCount is how many scenes precede current in play order - all of them,
// since finishedCount holds because play order is forced (DEC-10). A completed
// student has finished every scene.
func finishedCount(current int, completed bool) int {
	if completed {
		return len(sceneOrder)
	}
	for i, id := range sceneOrder {
		if id == current {
			return i
		}
	}
	return 0
}

// worldOf is the world number (types.World.Number) containing a minigame ID, or 0
// if the ID isn't in types.Worlds.
func worldOf(minigameID int) int {
	for _, world := range types.Worlds {
		for _, scene := range world.Scenes {
			if scene.MinigameID == minigameID {
				return world.Number
			}
		}
	}
	return 0
}
