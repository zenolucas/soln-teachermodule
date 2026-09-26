// Command showcase writes testdata/showcase_seed.sql: semi-realistic demo data for showing the portal to
// potential users. It's one teacher (teacher / teacher) with three Grade 6 sections, a full question set per
// classroom, and a few weeks of simulated play. Run it from the repo root:
//
//	go run ./testdata/showcase > testdata/showcase_seed.sql
//
// The output is deterministic (fixed RNG seed). Timestamps are relative to the day the seed is loaded
// (CURDATE() - INTERVAL n DAY), so the Home page's recent activity always looks current.
//
// The simulation follows the game's rules (see BACKEND_AUDIT.md, "Game → server data semantics"):
//   - A fraction/worded scene asks 3 random questions from the classroom's set, and each question attempt is
//     one fraction_responses row: right=1 plus the wrong tries before it, or right=0 when the student ran
//     out of energy on it.
//   - Energy starts at 3 and each wrong answer costs 1. At 0 it's game over: energy resets to 3 and the
//     scene starts again. Clearing a scene gives back 1 energy (pickups along the way), up to 5.
//   - A quiz is 10 multiple-choice questions. The student keeps clicking until the right choice, every
//     click is a multiple_choice_responses row, and the score (first-try right answers) is posted at the
//     end. Scene order is forced, so a student's progress is how many scenes they've cleared.
package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
)

// Password hashes (bcrypt, cost 10).
const (
	teacherHash = "$2a$10$ctgYzWWHK2xyB35nTrpJJOT.E1fxZeFkOJKGBwk6UjBDxTlRQBBlC" // "teacher"
	studentHash = "$2a$10$mykHwAL4VUrklYv79G7Sd.Mo59S1ItAfG7jIPF79p/PBttxDfhJmm" // "student"
)

type frac struct{ n, d int }

func (f frac) String() string { return fmt.Sprintf("%d/%d", f.n, f.d) }

// fracQ is a fraction or worded question. text is empty for plain fraction scenes.
type fracQ struct {
	text string
	a, b frac
}

// mcQ is a multiple-choice question "What is a op b?". sa and ncd are the distractors that model the two
// misconceptions the portal's hints detect (adding straight across, and no common denominator); either may
// be empty when the question has no such distractor.
type mcQ struct {
	a, b    frac
	op      string
	correct string
	wrong   []string
	sa, ncd string
}

func (q mcQ) text() string { return fmt.Sprintf("What is %s %s %s?", q.a, q.op, q.b) }

// similar reports whether both fractions share a denominator, which makes a question easier.
func (q mcQ) similar() bool { return q.a.d == q.b.d }

var fractionScenes = map[int][]fracQ{
	// World 1: addition.
	1: {{"", frac{1, 5}, frac{2, 5}}, {"", frac{2, 7}, frac{3, 7}}, {"", frac{3, 8}, frac{1, 8}}, {"", frac{1, 6}, frac{4, 6}}, {"", frac{2, 9}, frac{5, 9}}, {"", frac{3, 10}, frac{4, 10}}},
	2: {{"", frac{1, 2}, frac{1, 4}}, {"", frac{1, 3}, frac{1, 6}}, {"", frac{2, 5}, frac{1, 10}}, {"", frac{1, 4}, frac{3, 8}}, {"", frac{2, 3}, frac{1, 4}}, {"", frac{1, 2}, frac{2, 5}}},
	3: {
		{"Racket took 1/4 of the pan de sal in the morning and 2/4 in the afternoon. How much of the bread did he take in all?", frac{1, 4}, frac{2, 4}},
		{"Saisai walked 3/8 km to the river, then 2/8 km more to the forest. How far did she walk?", frac{3, 8}, frac{2, 8}},
		{"Racket stole 2/6 of a sack of rice, then came back for 3/6 more. What part of the sack did he steal?", frac{2, 6}, frac{3, 6}},
		{"Liza drank 1/5 liter of calamansi juice at lunch and 3/5 liter at dinner. How much juice did she drink?", frac{1, 5}, frac{3, 5}},
		{"A farmer planted corn on 2/9 of his land and rice on 4/9. What part of the land did he plant?", frac{2, 9}, frac{4, 9}},
	},
	4: {
		{"Racket used 1/2 kg of iron for a sword and 1/4 kg for a shield. How much iron did he use?", frac{1, 2}, frac{1, 4}},
		{"The blacksmith heats the metal for 2/3 hour and shapes it for 1/6 hour. How long does the job take?", frac{2, 3}, frac{1, 6}},
		{"Racket poured 3/10 liter of water into the bucket, then 2/5 liter more. How much water is in the bucket?", frac{3, 10}, frac{2, 5}},
		{"A blade is 5/8 m long and its handle is 1/4 m long. How long is the whole sword?", frac{5, 8}, frac{1, 4}},
		{"Racket bought 1/3 kg of coal in the morning and 1/2 kg in the afternoon. How much coal did he buy?", frac{1, 3}, frac{1, 2}},
	},
	// World 2: subtraction.
	6: {{"", frac{5, 6}, frac{1, 6}}, {"", frac{7, 8}, frac{3, 8}}, {"", frac{4, 5}, frac{2, 5}}, {"", frac{9, 10}, frac{3, 10}}, {"", frac{6, 7}, frac{2, 7}}, {"", frac{5, 9}, frac{2, 9}}},
	7: {{"", frac{3, 4}, frac{1, 2}}, {"", frac{5, 6}, frac{1, 3}}, {"", frac{7, 10}, frac{1, 5}}, {"", frac{2, 3}, frac{1, 6}}, {"", frac{7, 8}, frac{1, 4}}, {"", frac{4, 5}, frac{1, 2}}},
	8: {{"", frac{11, 12}, frac{5, 12}}, {"", frac{7, 9}, frac{4, 9}}, {"", frac{5, 8}, frac{1, 8}}, {"", frac{9, 10}, frac{7, 10}}, {"", frac{6, 11}, frac{2, 11}}},
	9: {{"", frac{5, 6}, frac{3, 8}}, {"", frac{7, 12}, frac{1, 4}}, {"", frac{3, 4}, frac{2, 3}}, {"", frac{9, 10}, frac{1, 2}}, {"", frac{5, 8}, frac{1, 3}}, {"", frac{4, 5}, frac{3, 10}}},
	10: {
		{"The rat had 7/8 of a cheese wheel and ate 3/8 of it. How much of the cheese is left?", frac{7, 8}, frac{3, 8}},
		{"A water tank was 9/10 full. The rat drank 2/5 of the tank. How full is the tank now?", frac{9, 10}, frac{2, 5}},
		{"The rat's tunnel will be 5/6 m long. It has dug 1/3 m so far. How much more must it dig?", frac{5, 6}, frac{1, 3}},
		{"Chip had 3/4 of a bag of seeds and gave 1/2 of the bag to the rat. How much of the bag does Chip have left?", frac{3, 4}, frac{1, 2}},
		{"The room was 11/12 flooded. After the pump ran, the water went down by 5/12. How much of the room is still flooded?", frac{11, 12}, frac{5, 12}},
	},
}

// Quizzes: 5 = Snekkers (addition), 11 = Crab (subtraction), 12 = Final Boss (mixed).
var quizzes = map[int][]mcQ{
	5: {
		{frac{1, 5}, frac{2, 5}, "+", "3/5", []string{"3/10", "2/5", "4/5"}, "3/10", ""},
		{frac{1, 2}, frac{1, 4}, "+", "3/4", []string{"2/6", "2/4", "1/8"}, "2/6", "2/4"},
		{frac{2, 7}, frac{3, 7}, "+", "5/7", []string{"5/14", "6/7", "1/7"}, "5/14", ""},
		{frac{1, 3}, frac{1, 6}, "+", "1/2", []string{"2/9", "2/6", "1/9"}, "2/9", "2/6"},
		{frac{3, 8}, frac{1, 8}, "+", "1/2", []string{"4/16", "3/8", "5/8"}, "4/16", ""},
		{frac{2, 5}, frac{1, 10}, "+", "1/2", []string{"3/15", "3/10", "2/10"}, "3/15", "3/10"},
		{frac{1, 4}, frac{3, 8}, "+", "5/8", []string{"4/12", "4/8", "3/8"}, "4/12", "4/8"},
		{frac{2, 3}, frac{1, 4}, "+", "11/12", []string{"3/7", "3/12", "1/12"}, "3/7", "3/12"},
		{frac{3, 10}, frac{4, 10}, "+", "7/10", []string{"7/20", "1/10", "12/10"}, "7/20", ""},
		{frac{1, 2}, frac{2, 5}, "+", "9/10", []string{"3/7", "3/10", "1/10"}, "3/7", "3/10"},
	},
	11: {
		{frac{5, 6}, frac{1, 6}, "-", "2/3", []string{"1/6", "5/6", "1/2"}, "", ""},
		{frac{3, 4}, frac{1, 2}, "-", "1/4", []string{"2/2", "2/4", "1/2"}, "2/2", "2/4"},
		{frac{7, 8}, frac{3, 8}, "-", "1/2", []string{"3/8", "5/8", "1/4"}, "", ""},
		{frac{5, 6}, frac{1, 3}, "-", "1/2", []string{"4/3", "4/6", "1/6"}, "4/3", "4/6"},
		{frac{7, 10}, frac{1, 5}, "-", "1/2", []string{"6/5", "6/10", "3/10"}, "6/5", "6/10"},
		{frac{9, 10}, frac{3, 10}, "-", "3/5", []string{"3/10", "7/10", "1/5"}, "", ""},
		{frac{2, 3}, frac{1, 6}, "-", "1/2", []string{"1/6", "1/3", "2/3"}, "", "1/6"},
		{frac{4, 5}, frac{1, 2}, "-", "3/10", []string{"3/3", "3/5", "1/2"}, "3/3", "3/5"},
		{frac{7, 8}, frac{1, 4}, "-", "5/8", []string{"6/4", "6/8", "3/8"}, "6/4", "6/8"},
		{frac{11, 12}, frac{5, 12}, "-", "1/2", []string{"1/12", "7/12", "5/12"}, "", ""},
	},
	12: {
		{frac{2, 5}, frac{1, 5}, "+", "3/5", []string{"3/10", "1/5", "4/5"}, "3/10", ""},
		{frac{1, 2}, frac{1, 3}, "+", "5/6", []string{"2/5", "2/3", "1/6"}, "2/5", "2/3"},
		{frac{7, 9}, frac{4, 9}, "-", "1/3", []string{"2/9", "4/9", "11/9"}, "", ""},
		{frac{3, 4}, frac{1, 8}, "+", "7/8", []string{"4/12", "4/8", "5/8"}, "4/12", "4/8"},
		{frac{5, 6}, frac{1, 2}, "-", "1/3", []string{"4/4", "4/6", "1/2"}, "4/4", "4/6"},
		{frac{3, 10}, frac{1, 2}, "+", "4/5", []string{"4/12", "4/10", "3/5"}, "4/12", "4/10"},
		{frac{9, 10}, frac{2, 5}, "-", "1/2", []string{"7/5", "7/10", "1/10"}, "7/5", "7/10"},
		{frac{1, 4}, frac{2, 3}, "+", "11/12", []string{"3/7", "3/12", "1/2"}, "3/7", "3/12"},
		{frac{7, 12}, frac{1, 4}, "-", "1/3", []string{"6/8", "6/12", "1/4"}, "6/8", "6/12"},
		{frac{5, 8}, frac{1, 4}, "+", "7/8", []string{"6/12", "6/8", "3/8"}, "6/12", "6/8"},
	},
}

// sceneOrder is the game's forced play order.
var sceneOrder = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

type room struct {
	id                      int
	name, section, desc     string
	startDaysAgo            int
	classHour, classMinute  int
	progress                []int // scenes cleared, one entry per student (12 = finished the game)
	equivalentChoiceVariant bool  // this teacher's Snekkers quiz has a second correct-looking choice
}

var rooms = []room{
	{1, "St. Joseph", "Grade 6", "Math 6 · Mon/Wed/Fri 8:00 AM · Room 204", 36, 8, 0,
		[]int{12, 12, 12, 12, 11, 11, 11, 10, 10, 9, 9, 9, 8, 8, 8, 7, 7, 7, 6, 6, 6, 6, 5, 5, 4, 4, 3, 2, 1, 0}, false},
	{2, "St. Therese", "Grade 6", "Math 6 · Mon/Wed/Fri 10:30 AM · Room 206", 29, 10, 30,
		[]int{12, 11, 11, 10, 9, 9, 8, 8, 7, 7, 7, 6, 6, 6, 5, 5, 5, 5, 4, 4, 4, 3, 3, 3, 2, 2, 1, 0}, true},
	{3, "St. Lorenzo Ruiz", "Grade 6", "Math 6 · Tue/Thu 1:00 PM · Room 105", 15, 13, 0,
		[]int{6, 5, 5, 5, 4, 4, 4, 4, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 0, 0, 0}, false},
}

// unenrolledStudents registered in the game but aren't in a class yet (transferees), so the
// Add students search has someone to find.
const unenrolledStudents = 4

var firstNames = []string{
	"Andrea", "Angelo", "Bea", "Carlo", "Kyla", "Daniel", "Denise", "Enzo", "Erica", "Francis",
	"Gabriel", "Hannah", "Ian", "Isabel", "Jasmine", "John Paul", "Joshua", "Julia", "Kevin", "Kristine",
	"Lance", "Leah", "Mark", "Marian", "Miguel", "Nicole", "Paolo", "Patricia", "Rafael", "Rhea",
	"Sofia", "Samantha", "Tristan", "Trisha", "Vince", "Yssa", "Zach", "Althea", "Bianca", "Christian",
	"Dominic", "Elijah", "Faith", "Gian", "Janine", "Jerome", "Kate", "Luis", "Mikaela", "Nathan",
	"Pia", "Renz", "Sean", "Therese", "Ysabel", "Jericho", "Clarisse", "Aaron", "Chloe", "Marco",
}

var lastNames = []string{
	"Dela Cruz", "Santos", "Reyes", "Bautista", "Garcia", "Mendoza", "Villanueva", "Ramos", "Aquino", "Castillo",
	"Navarro", "Fernandez", "Gonzales", "Torres", "Flores", "Rivera", "Domingo", "Mercado", "Pascual", "Salazar",
	"Soriano", "Tolentino", "Valdez", "Aguilar", "Cabrera", "De Guzman", "Enriquez", "Francisco", "Ignacio", "Lim",
	"Manalo", "Ocampo", "Peñaflor", "Quiambao", "Sarmiento", "Tan", "Umali", "Velasco", "Yap", "Zamora",
	"Bulan", "Dizon", "Galang", "Macaraeg", "Pangilinan", "Bagtas", "Carino", "Dulay", "Kimmayong", "Tabora",
}

type student struct {
	id                int
	first, last, user string
	section, classNo  string
	room              *room
	cleared           int
	partial           bool    // has started the scene after its last cleared one
	ability           float64 // chance of a first-try right answer on an easy question
	misconception     string  // "sa", "ncd" or ""
	stopsEarlyDaysAgo int     // 0, or the last day (days ago) this student played
	homework          bool    // also plays some evenings
}

var rng = rand.New(rand.NewSource(20260926))

func main() {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format, args...) }

	w(`-- Showcase data for the Sol'n Teacher Portal. GENERATED by testdata/showcase/main.go; don't edit by hand:
--   go run ./testdata/showcase > testdata/showcase_seed.sql
-- Load it after soln_db.sql (schema only), into an empty database:
--   cd ~/.local/share/soln-devtools/setupdb && go run . --reset --demo
-- Accounts: teacher / teacher (Clarissa Reyes, owns every classroom). Every student's password is "student".
-- Timestamps are relative to the day this file is loaded, so the data always looks recent.
-- No USE statement: the loader picks the database (setupdb, the integration tests, Docker's MARIADB_DATABASE).

`)

	// Accounts.
	w("INSERT INTO users (user_id, username, firstname, lastname, usertype, section, class_number, password) VALUES\n")
	w("(1, 'teacher', 'Clarissa', 'Reyes', 'teacher', NULL, NULL, '%s');\n\n", teacherHash)

	students := makeStudents()
	w("INSERT INTO users (user_id, username, firstname, lastname, usertype, section, class_number, password) VALUES\n")
	for i, s := range students {
		w("(%d, '%s', '%s', '%s', 'student', '%s', '%s', '%s')%s\n", s.id, s.user, sqlStr(s.first), sqlStr(s.last), sqlStr(s.section), s.classNo, studentHash, sep(i, len(students)))
	}
	w("\n")

	w("INSERT INTO classrooms (classroom_id, classroom_name, section, description, teacher_id) VALUES\n")
	for i, r := range rooms {
		w("(%d, '%s', '%s', '%s', 1)%s\n", r.id, sqlStr(r.name), sqlStr(r.section), sqlStr(r.desc), sep(i, len(rooms)))
	}
	w("\n")

	var enrolled []student
	for _, s := range students {
		if s.room != nil {
			enrolled = append(enrolled, s)
		}
	}
	w("INSERT INTO enrollments (classroom_id, student_id) VALUES\n")
	for i, s := range enrolled {
		w("(%d, %d)%s\n", s.room.id, s.id, sep(i, len(enrolled)))
	}
	w("\n")

	// Question sets: every classroom gets its own copy, as if each teacher had set it up.
	fracIDs := map[[2]int][]int{} // (classroom, minigame) -> question IDs
	mcIDs := map[[2]int][]int{}
	choiceIDs := map[int]map[string]int{} // MC question ID -> choice text -> choice ID
	nextFQ, nextMC, nextChoice := 1, 1, 1
	w("INSERT INTO fraction_questions (question_id, classroom_id, minigame_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES\n")
	var fqRows []string
	for _, r := range rooms {
		for _, mg := range sceneOrder {
			for _, q := range fractionScenes[mg] {
				text := "NULL"
				if q.text != "" {
					text = "'" + sqlStr(q.text) + "'"
				}
				fqRows = append(fqRows, fmt.Sprintf("(%d, %d, %d, %s, %d, %d, %d, %d)", nextFQ, r.id, mg, text, q.a.n, q.a.d, q.b.n, q.b.d))
				fracIDs[[2]int{r.id, mg}] = append(fracIDs[[2]int{r.id, mg}], nextFQ)
				nextFQ++
			}
		}
	}
	w("%s;\n\n", strings.Join(fqRows, ",\n"))

	var mcRows, choiceRows []string
	for _, r := range rooms {
		for _, mg := range []int{5, 11, 12} {
			for _, q := range quizFor(r, mg) {
				mcRows = append(mcRows, fmt.Sprintf("(%d, %d, %d, '%s')", nextMC, r.id, mg, q.text()))
				mcIDs[[2]int{r.id, mg}] = append(mcIDs[[2]int{r.id, mg}], nextMC)
				choiceIDs[nextMC] = map[string]int{}
				for _, c := range choiceOrder(q) {
					correct := "FALSE"
					if c == q.correct {
						correct = "TRUE"
					}
					choiceRows = append(choiceRows, fmt.Sprintf("(%d, %d, '%s', %s)", nextChoice, nextMC, c, correct))
					choiceIDs[nextMC][c] = nextChoice
					nextChoice++
				}
				nextMC++
			}
		}
	}
	w("INSERT INTO multiple_choice_questions (question_id, classroom_id, minigame_id, question_text) VALUES\n%s;\n\n", strings.Join(mcRows, ",\n"))
	w("INSERT INTO multiple_choice_choices (choice_id, question_id, choice_text, is_correct) VALUES\n%s;\n\n", strings.Join(choiceRows, ",\n"))

	// Play history.
	sim := &simulation{fracIDs: fracIDs, mcIDs: mcIDs, choiceIDs: choiceIDs}
	for i := range enrolled {
		sim.play(&enrolled[i])
	}
	// Rows go in chronological order, so auto-increment IDs follow time (the portal's "latest attempt"
	// rule uses the highest statistic_id).
	sort.SliceStable(sim.frac, func(i, j int) bool { return sim.frac[i].at < sim.frac[j].at })
	sort.SliceStable(sim.resp, func(i, j int) bool { return sim.resp[i].at < sim.resp[j].at })
	sort.SliceStable(sim.scores, func(i, j int) bool { return sim.scores[i].at < sim.scores[j].at })
	writeRows(w, "fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts, created_at)", sim.frac)
	writeRows(w, "multiple_choice_responses (classroom_id, minigame_id, question_id, student_id, choice_id, created_at)", sim.resp)
	writeRows(w, "multiple_choice_scores (classroom_id, minigame_id, student_id, score, created_at)", sim.scores)

	os.Stdout.WriteString(b.String())
}

// quizFor is the classroom's quiz. In the equivalent-choice variant, one Snekkers question has a distractor
// ("4/8") equal to its right answer ("1/2"), the kind of authoring slip the portal's question hints catch.
func quizFor(r room, mg int) []mcQ {
	qs := append([]mcQ(nil), quizzes[mg]...)
	if r.equivalentChoiceVariant && mg == 5 {
		q := qs[4]
		q.wrong = []string{"4/16", "3/8", "4/8"}
		qs[4] = q
	}
	return qs
}

// choiceOrder puts the right answer at a fixed, varied position (A–D) per question.
func choiceOrder(q mcQ) []string {
	pos := (q.a.n*7 + q.a.d*3 + q.b.n*5 + q.b.d) % 4
	out := append([]string(nil), q.wrong...)
	out = append(out[:pos], append([]string{q.correct}, out[pos:]...)...)
	return out
}

func makeStudents() []student {
	used := map[string]bool{}
	users := map[string]int{}
	name := func() (string, string, string) {
		for {
			f, l := firstNames[rng.Intn(len(firstNames))], lastNames[rng.Intn(len(lastNames))]
			if used[f+l] {
				continue
			}
			used[f+l] = true
			u := strings.ToLower(strings.NewReplacer(" ", "", "ñ", "n", "Ñ", "n").Replace(f)) + "." +
				strings.ToLower(strings.NewReplacer(" ", "", "ñ", "n", "Ñ", "n").Replace(l))
			users[u]++
			if users[u] > 1 {
				u = fmt.Sprintf("%s%d", u, users[u])
			}
			return f, l, u
		}
	}

	var out []student
	id := 2
	for ri := range rooms {
		r := &rooms[ri]
		var group []student
		for _, cleared := range shuffled(r.progress) {
			f, l, u := name()
			s := student{first: f, last: l, user: u, section: r.name, room: r, cleared: cleared}
			// Most of a class is solid and a few students struggle; strugglers tend to be further behind.
			weak := 0.14
			if cleared <= 4 {
				weak = 0.3
			}
			if rng.Float64() < weak {
				s.ability = clamp(0.64+rng.NormFloat64()*0.06, 0.5, 0.75)
			} else {
				s.ability = clamp(0.92+rng.NormFloat64()*0.04, 0.82, 0.98)
			}
			s.partial = cleared < 12 && rng.Float64() < 0.65
			switch x := rng.Float64(); {
			case s.ability < 0.72 && x < 0.55:
				s.misconception = "sa"
			case s.ability < 0.8 && x < 0.8:
				s.misconception = "ncd"
			}
			if cleared < 12 && rng.Float64() < 0.12 {
				s.stopsEarlyDaysAgo = 7 + rng.Intn(6)
			}
			s.homework = rng.Float64() < 0.3
			group = append(group, s)
		}
		// Class numbers follow the class list: alphabetical by last name.
		sort.Slice(group, func(i, j int) bool {
			if group[i].last != group[j].last {
				return group[i].last < group[j].last
			}
			return group[i].first < group[j].first
		})
		for i := range group {
			group[i].id = id
			group[i].classNo = fmt.Sprint(i + 1)
			id++
		}
		out = append(out, group...)
	}
	for i := 0; i < unenrolledStudents; i++ {
		f, l, u := name()
		sec := rooms[i%len(rooms)].name
		out = append(out, student{id: id, first: f, last: l, user: u, section: sec, classNo: fmt.Sprint(40 + i)})
		id++
	}
	return out
}

// A timestamp is days ago plus seconds into that day; it renders as a CURDATE()-relative SQL expression.
type ts struct{ day, sec int }

func (t ts) sql() string {
	return fmt.Sprintf("CURDATE() - INTERVAL %d DAY + INTERVAL %d SECOND", t.day, t.sec)
}

type row struct {
	at     float64 // sort key: larger is later
	values string
}

func key(t ts) float64 { return float64(-t.day)*86400 + float64(t.sec) }

type simulation struct {
	fracIDs, mcIDs     map[[2]int][]int
	choiceIDs          map[int]map[string]int
	frac, resp, scores []row
}

// sessions lists the days (days ago) a student plays: their class's meeting days since the start of the
// term, minus a few absences, plus evenings at home for some. Always at least 1 day ago, so nothing
// is in the future.
func sessions(s *student) []ts {
	r := s.room
	var out []ts
	gaps := []int{2, 2, 3} // Mon→Wed→Fri→Mon
	if r.classHour >= 13 {
		gaps = []int{2, 5} // Tue→Thu→Tue
	}
	for d, i := r.startDaysAgo, 0; d >= 1; d, i = d-gaps[i%len(gaps)], i+1 {
		if s.stopsEarlyDaysAgo > 0 && d < s.stopsEarlyDaysAgo {
			break
		}
		if rng.Float64() < 0.08 { // absent
			continue
		}
		out = append(out, ts{d, (r.classHour*60+r.classMinute+5+rng.Intn(15))*60 + rng.Intn(60)})
		if s.homework && d > 1 && rng.Float64() < 0.5 {
			out = append(out, ts{d - 1, (19*60+rng.Intn(120))*60 + rng.Intn(60)})
		}
	}
	return out
}

func (sim *simulation) play(s *student) {
	scenes := s.cleared
	if s.partial {
		scenes++
	}
	if scenes == 0 {
		return
	}
	days := sessions(s)
	if len(days) == 0 {
		return
	}
	// Spread the scenes evenly over the sessions, in order. A session can hold several scenes, played one
	// after another.
	energy := 3
	var last ts
	for i := 0; i < scenes; i++ {
		t := days[i*len(days)/scenes]
		if i > 0 && t.day == last.day && t.sec <= last.sec {
			t.sec = last.sec
		} else if i > 0 {
			energy = 3 // energy isn't saved: a new session starts at 3
		}
		t.sec += 30 + rng.Intn(90)
		mg := sceneOrder[i]
		finish := i < s.cleared
		if mg == 5 || mg == 11 || mg == 12 {
			if finish {
				last = sim.quiz(s, mg, t)
			}
			continue
		}
		energy, last = sim.fractionScene(s, mg, t, energy, finish)
		if mg == 2 || mg == 7 {
			energy = min(energy+1, 5) // the cooking and safe side minigames between scenes give energy too
		}
	}
}

// fractionScene plays one fraction/worded scene and returns the energy left. When finish is false the
// student stops partway: after a question or two, or at a game over.
func (sim *simulation) fractionScene(s *student, mg int, t ts, energy int, finish bool) (int, ts) {
	ids := sim.fracIDs[[2]int{s.room.id, mg}]
	qs := fractionScenes[mg]
	gameOvers := 0
	for {
		order := rng.Perm(len(ids))[:3]
		stopAfter := 3
		if !finish {
			stopAfter = 1 + rng.Intn(2)
		}
		over := false
		for n, qi := range order {
			if n == stopAfter {
				return energy, t
			}
			p := s.ability - difficulty(qs[qi], mg) + 0.08*float64(gameOvers)
			p = clamp(p, 0.2, 0.98)
			wrong := 0
			for rng.Float64() >= p {
				wrong++
				energy--
				t.sec += 20 + rng.Intn(40)
				if energy == 0 {
					break
				}
			}
			t.sec += 30 + rng.Intn(60)
			right := 1
			if energy == 0 {
				right = 0
			}
			sim.frac = append(sim.frac, row{key(t), fmt.Sprintf("(%d, %d, %d, %d, %d, %d, %s)", s.room.id, mg, ids[qi], s.id, right, wrong, t.sql())})
			if energy == 0 {
				over = true
				break
			}
		}
		if !over {
			return min(energy+1, 5), t
		}
		energy = 3
		gameOvers++
		t.sec += 60 + rng.Intn(120)
		if !finish && rng.Float64() < 0.5 {
			return energy, t // gave up for the day at the game over screen
		}
	}
}

// difficulty lowers the chance of a first-try right answer: unlike denominators are harder, and so
// are word problems and the later subtraction scenes.
func difficulty(q fracQ, mg int) float64 {
	d := 0.0
	if q.a.d != q.b.d {
		d += 0.08
	}
	if q.text != "" {
		d += 0.06
	}
	if mg >= 6 {
		d += 0.04
	}
	return d
}

func (sim *simulation) quiz(s *student, mg int, t ts) ts {
	qs := quizFor(*s.room, mg)
	ids := sim.mcIDs[[2]int{s.room.id, mg}]
	boost := 0.0
	for attempt := 0; attempt < 2; attempt++ {
		score := 0
		for qi, q := range qs {
			choices := sim.choiceIDs[ids[qi]]
			p := s.ability + boost - 0.05
			if !q.similar() {
				p -= 0.1
			}
			click := func(c string) {
				t.sec += 15 + rng.Intn(45)
				sim.resp = append(sim.resp, row{key(t), fmt.Sprintf("(%d, %d, %d, %d, %d, %s)", s.room.id, mg, ids[qi], s.id, choices[c], t.sql())})
			}
			if rng.Float64() < clamp(p, 0.15, 0.97) {
				score++
				click(q.correct)
				continue
			}
			first := pickWrong(s, q)
			click(first)
			tried := map[string]bool{first: true}
			for rng.Float64() >= 0.6 {
				var left []string
				for _, c := range q.wrong {
					if !tried[c] {
						left = append(left, c)
					}
				}
				if len(left) == 0 {
					break
				}
				c := left[rng.Intn(len(left))]
				tried[c] = true
				click(c)
			}
			click(q.correct)
		}
		t.sec += 20
		sim.scores = append(sim.scores, row{key(t), fmt.Sprintf("(%d, %d, %d, %d, %s)", s.room.id, mg, s.id, score, t.sql())})
		// Some students who scored low retake the quiz a few minutes later (usually, not always, better).
		if score >= 6 || rng.Float64() < 0.5 {
			return t
		}
		boost = 0.1
		t.sec += 180 + rng.Intn(300)
	}
	return t
}

// pickWrong is a student's first wrong choice: the distractor matching their misconception when the
// question has one, otherwise mostly the straight-across answer (the most common mistake), otherwise any.
func pickWrong(s *student, q mcQ) string {
	// In the equivalent-choice variant, "4/8" looks right to anyone who reduces fractions.
	for _, c := range q.wrong {
		if c == "4/8" && q.correct == "1/2" && rng.Float64() < 0.7 {
			return c
		}
	}
	switch {
	case s.misconception == "sa" && q.sa != "" && rng.Float64() < 0.75:
		return q.sa
	case s.misconception == "ncd" && q.ncd != "" && rng.Float64() < 0.75:
		return q.ncd
	case q.sa != "" && rng.Float64() < 0.35:
		return q.sa
	}
	return q.wrong[rng.Intn(len(q.wrong))]
}

func writeRows(w func(string, ...any), table string, rows []row) {
	const batch = 500
	for i := 0; i < len(rows); i += batch {
		end := min(i+batch, len(rows))
		w("INSERT INTO %s VALUES\n", table)
		for j := i; j < end; j++ {
			w("%s%s\n", rows[j].values, sep(j-i, end-i))
		}
		w("\n")
	}
}

func shuffled(xs []int) []int {
	out := append([]int(nil), xs...)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func sep(i, n int) string {
	if i == n-1 {
		return ";"
	}
	return ","
}

func sqlStr(s string) string { return strings.ReplaceAll(s, "'", "''") }

func clamp(x, lo, hi float64) float64 { return max(lo, min(hi, x)) }
