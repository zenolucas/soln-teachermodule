package handler

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"soln-teachermodule/database"
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

// buildStudentInsights composes one StudentInsight per enrolled student from the raw
// per-classroom query results (02 §C2/§C3). finished, quiz and frac are expected to
// cover the same classroom as students; finished is passed to currentScene as both
// the played and quizScored argument (see database.GetFinishedScenes).
func buildStudentInsights(students []types.Student, finished map[int]map[int]bool, quiz []types.QuizScoreRow, frac []types.FractionAggRow) []types.StudentInsight {
	flagsByStudent := EvaluateFlags(quiz, frac)

	quizByStudent := map[int][]types.QuizScoreRow{}
	for _, q := range quiz {
		quizByStudent[q.StudentID] = append(quizByStudent[q.StudentID], q)
	}
	fracByStudent := map[int][]types.FractionAggRow{}
	for _, f := range frac {
		fracByStudent[f.StudentID] = append(fracByStudent[f.StudentID], f)
	}

	insights := make([]types.StudentInsight, 0, len(students))
	for _, s := range students {
		studentID, _ := strconv.Atoi(s.UserID)
		current, completed := currentScene(finished[studentID], finished[studentID])

		quizAvg := -1
		sum, count := 0, 0
		for _, q := range quizByStudent[studentID] {
			if q.Total <= 0 {
				continue
			}
			sum += q.Score * 100 / q.Total
			count++
		}
		if count > 0 {
			quizAvg = sum / count
		}

		accuracy := -1
		right, total := 0, 0
		for _, f := range fracByStudent[studentID] {
			right += f.Right
			total += f.Right + f.Wrong
		}
		if total > 0 {
			accuracy = right * 100 / total
		}

		insights = append(insights, types.StudentInsight{
			Student:     s,
			Current:     current,
			Completed:   completed,
			World:       worldOf(current),
			QuizAvgPct:  quizAvg,
			AccuracyPct: accuracy,
			Flags:       flagsByStudent[studentID],
		})
	}
	return insights
}

// summarize rolls a classroom's StudentInsights up into its ClassroomSummary
// (02 §C3). Completed students are already World 3 by construction - currentScene
// (DEC-10) returns minigame 12 on completion, and 12 is World 3's only scene - so no
// special-casing is needed here beyond checking World == 3.
func summarize(room types.Classroom, insights []types.StudentInsight) types.ClassroomSummary {
	sum := types.ClassroomSummary{
		Classroom:    room,
		StudentCount: len(insights),
		QuizAvgPct:   -1,
	}

	quizSum, quizCount := 0, 0
	for _, in := range insights {
		if len(in.Flags) > 0 {
			sum.FlaggedCount++
		}
		if in.QuizAvgPct >= 0 {
			quizSum += in.QuizAvgPct
			quizCount++
		}
		if in.World >= 1 && in.World <= 3 {
			sum.PerWorld[in.World]++
		}
		if in.World == 3 {
			sum.ReachedW3++
		}
	}
	if quizCount > 0 {
		sum.QuizAvgPct = quizSum / quizCount
	}
	return sum
}

// loadClassroomInsights runs the four per-classroom insight queries and composes
// their results (pure buildStudentInsights) into one StudentInsight per enrolled
// student.
func loadClassroomInsights(ctx context.Context, classroomID int) ([]types.StudentInsight, error) {
	students, err := database.GetEnrolledStudents(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	finished, err := database.GetFinishedScenes(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	quiz, err := database.GetLatestQuizScores(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	frac, err := database.GetFractionAggregates(ctx, classroomID)
	if err != nil {
		return nil, err
	}
	return buildStudentInsights(students, finished, quiz, frac), nil
}

// loadTeacherSummaries builds one ClassroomSummary per classroom a teacher owns, for
// the Home cards and the sidebar's flag badges.
func loadTeacherSummaries(ctx context.Context, teacherID int) ([]types.ClassroomSummary, error) {
	classroomIDs, err := database.GetTeacherClassroomIDs(ctx, teacherID)
	if err != nil {
		return nil, err
	}

	summaries := make([]types.ClassroomSummary, 0, len(classroomIDs))
	for _, id := range classroomIDs {
		room, err := database.GetClassroom(ctx, id)
		if err != nil {
			return nil, err
		}
		insights, err := loadClassroomInsights(ctx, id)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summarize(room, insights))
	}
	return summaries, nil
}

// buildSceneSummaries composes every scene's SceneSummary (02 §C4) from the raw
// per-classroom query results. finished is the same "has any row" map
// database.GetFinishedScenes returns (keyed student then minigame); every scene in
// types.Worlds is always present in the result, even one with no data at all.
func buildSceneSummaries(enrolled int, finished map[int]map[int]bool, quizRows []types.QuizScoreRow, counts map[int]int, acc map[int][2]int) map[int]types.SceneSummary {
	took := map[int]int{}
	for _, scenes := range finished {
		for minigameID, done := range scenes {
			if done {
				took[minigameID]++
			}
		}
	}

	quizByMinigame := map[int][]types.QuizScoreRow{}
	for _, q := range quizRows {
		quizByMinigame[q.MinigameID] = append(quizByMinigame[q.MinigameID], q)
	}

	summaries := map[int]types.SceneSummary{}
	for _, world := range types.Worlds {
		for _, scene := range world.Scenes {
			s := types.SceneSummary{
				MinigameID:    scene.MinigameID,
				QuestionCount: counts[scene.MinigameID],
				AccuracyPct:   -1,
				QuizAvgPct:    -1,
				CompletionPct: -1,
				TookCount:     took[scene.MinigameID],
			}
			if enrolled > 0 {
				s.CompletionPct = s.TookCount * 100 / enrolled
			}

			if scene.Kind == types.KindQuiz {
				sum, count := 0, 0
				for _, q := range quizByMinigame[scene.MinigameID] {
					if q.Total <= 0 {
						continue
					}
					sum += q.Score * 100 / q.Total
					count++
				}
				if count > 0 {
					s.QuizAvgPct = sum / count
				}
			} else if rw, ok := acc[scene.MinigameID]; ok {
				right, wrong := rw[0], rw[1]
				if right+wrong > 0 {
					s.AccuracyPct = right * 100 / (right + wrong)
				}
			}

			summaries[scene.MinigameID] = s
		}
	}
	return summaries
}

// summarizeQuiz rolls a quiz's raw scores into its QuizSummary (02 §C8, 01 §1g's four
// stat cards): AvgScore/AvgPct come from the same mean score (never two independently
// rounded numbers that could disagree), Median from the sorted raw scores (even count
// averages the two middle scores), Below from a cross-multiplied threshold check (no
// floats, matching EvaluateFlags' quiz_below_60 rule), and Histogram indexed by raw
// score 0..total (len(Histogram) == total+1).
func summarizeQuiz(scores []types.StudentScore, total, enrolled int) types.QuizSummary {
	sum := types.QuizSummary{
		Took:      len(scores),
		Enrolled:  enrolled,
		Histogram: make([]int, total+1),
	}
	if len(scores) == 0 {
		return sum
	}

	sumScore := 0
	for _, s := range scores {
		sumScore += s.Score
		if s.Score >= 0 && s.Score <= total {
			sum.Histogram[s.Score]++
		}
		if total > 0 && s.Score*100 < types.PassPct*total {
			sum.Below++
		}
	}
	sum.AvgScore = float64(sumScore) / float64(len(scores))
	if total > 0 {
		sum.AvgPct = int(sum.AvgScore / float64(total) * 100)
	}

	sorted := make([]int, len(scores))
	for i, s := range scores {
		sorted[i] = s.Score
	}
	sort.Ints(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		sum.Median = float64(sorted[mid-1]+sorted[mid]) / 2
	} else {
		sum.Median = float64(sorted[mid])
	}

	return sum
}

// collapseActivity merges consecutive "played" rows for the same student and scene
// into a single event: a fraction/worded scene writes one row per question, so
// without this, playing a 3-question scene would show up as 3 near-identical rows.
// "scored" rows (quiz attempts) are never collapsed - database.GetRecentActivity
// already returns one per attempt. rows is expected newest-first; order and every
// non-collapsed row are otherwise preserved. DEC-24: the merged event is always
// "played {scene}", never "finished" or "started".
func collapseActivity(rows []types.Activity) []types.Activity {
	var out []types.Activity
	for _, r := range rows {
		if r.Kind == "played" && len(out) > 0 {
			last := out[len(out)-1]
			if last.Kind == "played" && last.UserID == r.UserID &&
				last.ClassroomID == r.ClassroomID && last.MinigameID == r.MinigameID {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

// buildJourney composes one student's 12-tile journey (01 §1h) from their insight
// (Current/Completed, from currentScene - DEC-10) and the classroom-wide quiz/frac
// rows (T4.2), filtered here to just this student. A tile before Current, or every
// tile when Completed, is "done" - pass if its score/accuracy is ≥60%, fail
// otherwise; a finished tile with no matching row at all (possible in sparse test
// data, though DEC-10 assumes forced play order means this doesn't happen for real)
// defaults to pass, since there's no data suggesting otherwise. The Current tile
// itself, and everything after it, never shows a score (DEC-24: no "finished 60%
// done" style partial-progress claims).
func buildJourney(ins types.StudentInsight, quiz []types.QuizScoreRow, frac []types.FractionAggRow) []types.JourneyTile {
	studentID, _ := strconv.Atoi(ins.UserID)

	quizByScene := map[int]types.QuizScoreRow{}
	for _, q := range quiz {
		if q.StudentID == studentID {
			quizByScene[q.MinigameID] = q
		}
	}
	fracByScene := map[int]types.FractionAggRow{}
	for _, f := range frac {
		if f.StudentID == studentID {
			fracByScene[f.MinigameID] = f
		}
	}

	currentIdx := -1
	for i, id := range sceneOrder {
		if id == ins.Current {
			currentIdx = i
		}
	}

	tiles := make([]types.JourneyTile, 0, len(sceneOrder))
	for i, id := range sceneOrder {
		scene, _ := types.SceneByID(id)
		tile := types.JourneyTile{Scene: scene}

		switch {
		case ins.Completed || (currentIdx >= 0 && i < currentIdx):
			pass := true
			if scene.Kind == types.KindQuiz {
				if q, ok := quizByScene[id]; ok && q.Total > 0 {
					tile.ScoreText = fmt.Sprintf("%d/%d", q.Score, q.Total)
					pass = q.Score*100 >= types.PassPct*q.Total
				}
			} else if f, ok := fracByScene[id]; ok {
				if total := f.Right + f.Wrong; total > 0 {
					pct := f.Right * 100 / total
					tile.ScoreText = fmt.Sprintf("%d%%", pct)
					pass = pct >= types.PassPct
				}
			}
			if pass {
				tile.State = "done-pass"
			} else {
				tile.State = "done-fail"
			}
		case !ins.Completed && i == currentIdx:
			tile.State = "current"
		default:
			tile.State = "locked"
		}

		tiles = append(tiles, tile)
	}
	return tiles
}

// neighbours finds id's previous/next entries in ids (DEC-19: the class list in the
// current sort order) - "" at either end, for a Previous/Next button to disable
// instead of link.
func neighbours(ids []string, id string) (prev, next string) {
	for i, v := range ids {
		if v != id {
			continue
		}
		if i > 0 {
			prev = ids[i-1]
		}
		if i < len(ids)-1 {
			next = ids[i+1]
		}
		return prev, next
	}
	return "", ""
}
