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
// created_at is scanned as a string and parsed in the app server's local time zone
// (matching DEC-16's server-local greeting), since the connection isn't opened with
// the MySQL driver's parseTime option.
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
		a.At, err = time.ParseInLocation("2006-01-02 15:04:05", createdAt, time.Local)
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
