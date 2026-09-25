package handler

import (
	"net/url"
	"sort"
	"strconv"
	"strings"

	"soln-teachermodule/types"
)

// StudentQuery is the students list's search/filter/sort/page state (02 §C5, DEC-19),
// parsed from the request's query string.
type StudentQuery struct {
	Q      string
	Filter string
	Sort   string
	Page   int
}

// parseStudentQuery reads a StudentQuery from url.Values, defaulting Filter to "all",
// Sort to "attention" and Page to 1 - an unrecognized filter/sort value falls back to
// its default rather than being kept as-is, so a stale or hand-edited query string
// never produces an error page.
func parseStudentQuery(v url.Values) StudentQuery {
	q := StudentQuery{
		Q:      v.Get("q"),
		Filter: v.Get("filter"),
		Sort:   v.Get("sort"),
		Page:   1,
	}

	switch q.Filter {
	case "attention", "w1", "w2", "w3":
	default:
		q.Filter = "all"
	}

	switch q.Sort {
	case "attention", "name", "quiz", "accuracy":
	default:
		q.Sort = "attention"
	}

	if page, err := strconv.Atoi(v.Get("page")); err == nil && page > 0 {
		q.Page = page
	}

	return q
}

// studentSortName is the lowercased "lastname firstname" key every name comparison
// (the "name" sort, and every sort's tiebreak) uses - matching
// database.GetEnrolledStudents' own lastname-first ordering.
func studentSortName(s types.StudentInsight) string {
	return strings.ToLower(s.Lastname + " " + s.Firstname)
}

// pctAscNoDataLast compares two -1-or-percentage values (T4.1-T4.4's "-1 = no data"
// convention) for ascending order, except -1 always sorts last regardless of its
// literal (smallest) numeric value.
func pctAscNoDataLast(a, b int) bool {
	if a == -1 {
		return false
	}
	if b == -1 {
		return true
	}
	return a < b
}

// applyStudentQuery filters, sorts and pages a classroom's student insights (02 §C5).
// counts is computed after the q search but before the filter pill, so every pill's
// count reflects the current search term regardless of which pill is selected.
func applyStudentQuery(all []types.StudentInsight, q StudentQuery) (page []types.StudentInsight, total int, counts map[string]int) {
	needle := strings.ToLower(strings.TrimSpace(q.Q))

	var matched []types.StudentInsight
	for _, s := range all {
		if needle != "" && !strings.Contains(strings.ToLower(s.Firstname+" "+s.Lastname), needle) {
			continue
		}
		matched = append(matched, s)
	}

	counts = map[string]int{"all": len(matched)}
	for _, s := range matched {
		if len(s.Flags) > 0 {
			counts["attention"]++
		}
		switch s.World {
		case 1:
			counts["w1"]++
		case 2:
			counts["w2"]++
		case 3:
			counts["w3"]++
		}
	}

	var filtered []types.StudentInsight
	for _, s := range matched {
		switch q.Filter {
		case "attention":
			if len(s.Flags) == 0 {
				continue
			}
		case "w1":
			if s.World != 1 {
				continue
			}
		case "w2":
			if s.World != 2 {
				continue
			}
		case "w3":
			if s.World != 3 {
				continue
			}
		}
		filtered = append(filtered, s)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		switch q.Sort {
		case "name":
			return studentSortName(a) < studentSortName(b)
		case "quiz":
			return pctAscNoDataLast(a.QuizAvgPct, b.QuizAvgPct)
		case "accuracy":
			return pctAscNoDataLast(a.AccuracyPct, b.AccuracyPct)
		default: // "attention"
			if len(a.Flags) != len(b.Flags) {
				return len(a.Flags) > len(b.Flags)
			}
			return studentSortName(a) < studentSortName(b)
		}
	})

	total = len(filtered)

	pageCount := (total + types.StudentsPageSize - 1) / types.StudentsPageSize
	if pageCount < 1 {
		pageCount = 1
	}
	p := q.Page
	if p < 1 {
		p = 1
	}
	if p > pageCount {
		p = pageCount
	}

	start := min((p-1)*types.StudentsPageSize, total)
	end := min(start+types.StudentsPageSize, total)
	page = filtered[start:end]

	return page, total, counts
}
