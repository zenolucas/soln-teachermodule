package database

import (
	"context"
	"fmt"
	"time"

	"soln-teachermodule/types"
)

// getRecentActivitySQL is a UNION over the two tables that each produce a distinct
// recent-activity event (02 §D1): fraction_responses covers both fraction and worded
// scenes - they share the same table - and multiple_choice_scores is the one row per
// quiz attempt. multiple_choice_responses (the per-question choice picks within an
// attempt) isn't queried here: it would only duplicate the same attempt the
// accompanying scores row already reports, and types.Activity.Kind only has room for
// "played"/"scored" - DEC-24 drops "started"/"finished" entirely, not just for
// fraction/worded rows. Both halves join enrollments (DEC-8, enrolled only) and scope
// to the teacher's own classrooms.
const getRecentActivitySQL = `
SELECT u.user_id, u.firstname, u.lastname, c.classroom_id, c.classroom_name,
       fr.minigame_id, 'played' AS kind, 0 AS score, 0 AS total, fr.created_at
FROM fraction_responses fr
JOIN classrooms c   ON c.classroom_id = fr.classroom_id AND c.teacher_id = ?
JOIN enrollments e  ON e.classroom_id = fr.classroom_id AND e.student_id = fr.student_id
JOIN users u        ON u.user_id = fr.student_id

UNION ALL

SELECT u.user_id, u.firstname, u.lastname, c.classroom_id, c.classroom_name,
       mcs.minigame_id, 'scored' AS kind, mcs.score AS score,
       (SELECT COUNT(*) FROM multiple_choice_questions q
        WHERE q.minigame_id = mcs.minigame_id AND q.classroom_id = mcs.classroom_id) AS total,
       mcs.created_at
FROM multiple_choice_scores mcs
JOIN classrooms c   ON c.classroom_id = mcs.classroom_id AND c.teacher_id = ?
JOIN enrollments e  ON e.classroom_id = mcs.classroom_id AND e.student_id = mcs.student_id
JOIN users u        ON u.user_id = mcs.student_id

ORDER BY created_at DESC
LIMIT ?
`

// GetRecentActivity returns a teacher's most recent activity events, newest first,
// across every classroom they own. It's raw rows: a fraction/worded scene writes one
// row per question, so playing one scene can produce several consecutive "played"
// rows here. collapseActivity (handler/insights.go) merges those into one event per
// scene.
//
// created_at is scanned as a string and parsed as UTC, not the app server's local
// time zone: a MySQL/MariaDB TIMESTAMP column is stored in UTC and converted to the
// *session's* time_zone on read (default "SYSTEM", i.e. the DB server's own OS time
// zone) - not the connecting client's - so the string this driver receives is on the
// DB server's clock, which has no reason to match the app server's (confirmed live:
// this dev DB's SYSTEM zone is UTC while the app server itself runs in Pacific time -
// parsing as time.Local previously misread a same-second insert as "8 hr ago"). The
// connection isn't opened with the MySQL driver's parseTime option, which would
// otherwise handle this conversion automatically.
func GetRecentActivity(ctx context.Context, teacherID int, limit int) ([]types.Activity, error) {
	rows, err := db.QueryContext(ctx, getRecentActivitySQL, teacherID, teacherID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activity []types.Activity
	for rows.Next() {
		var a types.Activity
		var createdAt string
		if err := rows.Scan(&a.UserID, &a.First, &a.Last, &a.ClassroomID, &a.ClassName,
			&a.MinigameID, &a.Kind, &a.Score, &a.Total, &createdAt); err != nil {
			return nil, fmt.Errorf("GetRecentActivity: %v", err)
		}
		a.At, err = time.ParseInLocation("2006-01-02 15:04:05", createdAt, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("GetRecentActivity: parsing created_at %q: %v", createdAt, err)
		}
		activity = append(activity, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetRecentActivity: %v", err)
	}
	return activity, nil
}

// GetFinishedScenes returns, per enrolled student, the set of minigame IDs they have
// at least one row for in this classroom - a fraction_responses row for a
// fraction/worded scene, or a multiple_choice_scores row for a quiz. The same map is
// meant to be passed as both the played and quizScored arguments to
// handler.currentScene (DEC-10): for a quiz scene, the only way a row exists at all
// is if it was scored, so "has a row" and "was scored" are the same fact there.
func GetFinishedScenes(ctx context.Context, classroomID int) (map[int]map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT fr.student_id, fr.minigame_id
		FROM fraction_responses fr
		JOIN enrollments e ON e.classroom_id = fr.classroom_id AND e.student_id = fr.student_id
		WHERE fr.classroom_id = ?
		UNION
		SELECT mcs.student_id, mcs.minigame_id
		FROM multiple_choice_scores mcs
		JOIN enrollments e ON e.classroom_id = mcs.classroom_id AND e.student_id = mcs.student_id
		WHERE mcs.classroom_id = ?
	`, classroomID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	finished := map[int]map[int]bool{}
	for rows.Next() {
		var studentID, minigameID int
		if err := rows.Scan(&studentID, &minigameID); err != nil {
			return nil, fmt.Errorf("GetFinishedScenes: %v", err)
		}
		if finished[studentID] == nil {
			finished[studentID] = map[int]bool{}
		}
		finished[studentID][minigameID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetFinishedScenes: %v", err)
	}
	return finished, nil
}

// latestQuizScoresSQL keeps DEC-9's "latest attempt" rule (MAX(statistic_id) per
// student per quiz) in one place, so switching to best-attempt later is a one-line
// change here instead of a hunt through every quiz query.
const latestQuizScoresSQL = `
SELECT mcs.student_id, mcs.minigame_id, mcs.score,
       (SELECT COUNT(*) FROM multiple_choice_questions q
        WHERE q.minigame_id = mcs.minigame_id AND q.classroom_id = mcs.classroom_id) AS total
FROM multiple_choice_scores mcs
JOIN enrollments e ON e.classroom_id = mcs.classroom_id AND e.student_id = mcs.student_id
JOIN (
	SELECT student_id, minigame_id, MAX(statistic_id) AS latest_id
	FROM multiple_choice_scores
	WHERE classroom_id = ?
	GROUP BY student_id, minigame_id
) latest ON latest.latest_id = mcs.statistic_id
WHERE mcs.classroom_id = ?
`

// GetLatestQuizScores returns each enrolled student's latest attempt at every quiz
// they've taken in this classroom, with Total filled in from a count of that quiz's
// questions.
func GetLatestQuizScores(ctx context.Context, classroomID int) ([]types.QuizScoreRow, error) {
	rows, err := db.QueryContext(ctx, latestQuizScoresSQL, classroomID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []types.QuizScoreRow
	for rows.Next() {
		var s types.QuizScoreRow
		if err := rows.Scan(&s.StudentID, &s.MinigameID, &s.Score, &s.Total); err != nil {
			return nil, fmt.Errorf("GetLatestQuizScores: %v", err)
		}
		scores = append(scores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetLatestQuizScores: %v", err)
	}
	return scores, nil
}

// GetFractionAggregates sums each enrolled student's right/wrong attempts per
// fraction/worded minigame in this classroom. It doesn't compute MAX(num_wrong_attempts)
// (02 §C2's wrong_streak input): DEC-24 drops the wrong_streak flag pending the
// game-client audit, and types.FractionAggRow has no field for it - adding one back is
// a matter for whoever revisits DEC-24.
func GetFractionAggregates(ctx context.Context, classroomID int) ([]types.FractionAggRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT fr.student_id, fr.minigame_id, SUM(fr.num_right_attempts), SUM(fr.num_wrong_attempts)
		FROM fraction_responses fr
		JOIN enrollments e ON e.classroom_id = fr.classroom_id AND e.student_id = fr.student_id
		WHERE fr.classroom_id = ?
		GROUP BY fr.student_id, fr.minigame_id
	`, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aggs []types.FractionAggRow
	for rows.Next() {
		var a types.FractionAggRow
		if err := rows.Scan(&a.StudentID, &a.MinigameID, &a.Right, &a.Wrong); err != nil {
			return nil, fmt.Errorf("GetFractionAggregates: %v", err)
		}
		aggs = append(aggs, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetFractionAggregates: %v", err)
	}
	return aggs, nil
}

// GetEnrolledStudents is GetStudents' enrollments+users join, but ordered by
// lastname, firstname - GetStudents itself has no ORDER BY today, so its result order
// isn't guaranteed, and this task's file list doesn't include changing it.
func GetEnrolledStudents(ctx context.Context, classroomID int) ([]types.Student, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT u.firstname, u.lastname, u.user_id
		FROM enrollments e
		JOIN users u ON u.user_id = e.student_id
		WHERE e.classroom_id = ?
		ORDER BY u.lastname, u.firstname
	`, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []types.Student
	for rows.Next() {
		var s types.Student
		if err := rows.Scan(&s.Firstname, &s.Lastname, &s.UserID); err != nil {
			return nil, fmt.Errorf("GetEnrolledStudents: %v", err)
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetEnrolledStudents: %v", err)
	}
	return students, nil
}

// GetTeacherClassroomIDs lists the classroom IDs a teacher owns, for callers (e.g.
// loadTeacherSummaries) that need to loop over every classroom a teacher has.
func GetTeacherClassroomIDs(ctx context.Context, teacherID int) ([]int, error) {
	rows, err := db.QueryContext(ctx, "SELECT classroom_id FROM classrooms WHERE teacher_id = ?", teacherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("GetTeacherClassroomIDs: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetTeacherClassroomIDs: %v", err)
	}
	return ids, nil
}

// GetQuestionCounts returns each minigame's question count in this classroom (02
// §C4), from a UNION of the two question tables. A minigame_id only ever appears in
// one of them, so there's nothing to deduplicate.
func GetQuestionCounts(ctx context.Context, classroomID int) (map[int]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT minigame_id, COUNT(*) FROM fraction_questions WHERE classroom_id = ? GROUP BY minigame_id
		UNION ALL
		SELECT minigame_id, COUNT(*) FROM multiple_choice_questions WHERE classroom_id = ? GROUP BY minigame_id
	`, classroomID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[int]int{}
	for rows.Next() {
		var minigameID, count int
		if err := rows.Scan(&minigameID, &count); err != nil {
			return nil, fmt.Errorf("GetQuestionCounts: %v", err)
		}
		counts[minigameID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetQuestionCounts: %v", err)
	}
	return counts, nil
}

// GetSceneAccuracy sums enrolled students' right/wrong attempts per fraction/worded
// minigame in this classroom, as [2]int{right, wrong}.
func GetSceneAccuracy(ctx context.Context, classroomID int) (map[int][2]int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT fr.minigame_id, SUM(fr.num_right_attempts), SUM(fr.num_wrong_attempts)
		FROM fraction_responses fr
		JOIN enrollments e ON e.classroom_id = fr.classroom_id AND e.student_id = fr.student_id
		WHERE fr.classroom_id = ?
		GROUP BY fr.minigame_id
	`, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	acc := map[int][2]int{}
	for rows.Next() {
		var minigameID, right, wrong int
		if err := rows.Scan(&minigameID, &right, &wrong); err != nil {
			return nil, fmt.Errorf("GetSceneAccuracy: %v", err)
		}
		acc[minigameID] = [2]int{right, wrong}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetSceneAccuracy: %v", err)
	}
	return acc, nil
}

// GetQuizChoiceDistribution returns, per question, how many enrolled students picked
// each choice - across every attempt, not just each student's latest (02 §C8):
// multiple_choice_responses rows aren't individually tied to a specific attempt
// without a timestamp to group them by, so there's no way to isolate "the latest
// attempt's responses" the way GetLatestQuizScores does for scores. The enrollment
// check lives inside the LEFT JOIN's own ON clause, not a WHERE, so a choice with zero
// (enrolled) responses still gets a row with Count 0 instead of being dropped.
func GetQuizChoiceDistribution(ctx context.Context, classroomID, minigameID int) (map[int][]types.ChoiceCount, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT c.question_id, c.choice_id, c.choice_text, c.is_correct, COUNT(r.response_id)
		FROM multiple_choice_choices c
		JOIN multiple_choice_questions q ON q.question_id = c.question_id
		LEFT JOIN multiple_choice_responses r
			ON r.choice_id = c.choice_id
			AND r.minigame_id = q.minigame_id
			AND r.classroom_id = q.classroom_id
			AND EXISTS (SELECT 1 FROM enrollments e WHERE e.classroom_id = r.classroom_id AND e.student_id = r.student_id)
		WHERE q.minigame_id = ? AND q.classroom_id = ?
		GROUP BY c.question_id, c.choice_id, c.choice_text, c.is_correct
		ORDER BY c.question_id, c.choice_id
	`, minigameID, classroomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := map[int][]types.ChoiceCount{}
	for rows.Next() {
		var questionID int
		var cc types.ChoiceCount
		if err := rows.Scan(&questionID, &cc.ChoiceID, &cc.Text, &cc.IsCorrect, &cc.Count); err != nil {
			return nil, fmt.Errorf("GetQuizChoiceDistribution: %v", err)
		}
		dist[questionID] = append(dist[questionID], cc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetQuizChoiceDistribution: %v", err)
	}
	return dist, nil
}

// GetQuizStudentScores returns every enrolled student's latest attempt at one quiz
// (DEC-9), with their name for display. Total is the same per-classroom question
// count for every row (a quiz has one question count), included per row so callers
// don't need a second query just to compute a percentage.
func GetQuizStudentScores(ctx context.Context, classroomID, minigameID int) ([]types.StudentScore, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT u.user_id, u.firstname, u.lastname, mcs.score,
		       (SELECT COUNT(*) FROM multiple_choice_questions q
		        WHERE q.minigame_id = mcs.minigame_id AND q.classroom_id = mcs.classroom_id) AS total
		FROM multiple_choice_scores mcs
		JOIN enrollments e ON e.classroom_id = mcs.classroom_id AND e.student_id = mcs.student_id
		JOIN users u ON u.user_id = mcs.student_id
		JOIN (
			SELECT student_id, MAX(statistic_id) AS latest_id
			FROM multiple_choice_scores
			WHERE classroom_id = ? AND minigame_id = ?
			GROUP BY student_id
		) latest ON latest.latest_id = mcs.statistic_id
		WHERE mcs.classroom_id = ? AND mcs.minigame_id = ?
	`, classroomID, minigameID, classroomID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []types.StudentScore
	for rows.Next() {
		var s types.StudentScore
		if err := rows.Scan(&s.UserID, &s.First, &s.Last, &s.Score, &s.Total); err != nil {
			return nil, fmt.Errorf("GetQuizStudentScores: %v", err)
		}
		scores = append(scores, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetQuizStudentScores: %v", err)
	}
	return scores, nil
}

// GetStudentAccuracy sums each enrolled student's right/wrong attempts on one
// fraction/worded minigame (02 §C9), for the fraction/worded statistics page's
// Students card. A student with zero rows for this minigame doesn't appear at all -
// there's nothing to average.
func GetStudentAccuracy(ctx context.Context, classroomID, minigameID int) ([]types.StudentAccuracy, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT u.user_id, u.firstname, u.lastname, SUM(fr.num_right_attempts), SUM(fr.num_wrong_attempts)
		FROM fraction_responses fr
		JOIN enrollments e ON e.classroom_id = fr.classroom_id AND e.student_id = fr.student_id
		JOIN users u ON u.user_id = fr.student_id
		WHERE fr.classroom_id = ? AND fr.minigame_id = ?
		GROUP BY u.user_id, u.firstname, u.lastname
	`, classroomID, minigameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.StudentAccuracy
	for rows.Next() {
		var s types.StudentAccuracy
		if err := rows.Scan(&s.UserID, &s.First, &s.Last, &s.Right, &s.Wrong); err != nil {
			return nil, fmt.Errorf("GetStudentAccuracy: %v", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetStudentAccuracy: %v", err)
	}
	return out, nil
}
