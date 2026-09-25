package handler

import (
	"fmt"
	"strings"

	"soln-teachermodule/types"
	"soln-teachermodule/util"
)

// ChoiceStat is one multiple-choice question's per-choice tally, enough to find the
// most-picked wrong choice and classify it (02 §B6).
type ChoiceStat struct {
	Letter  string
	Text    string
	Correct bool
	Count   int
}

// opFromText finds the question's arithmetic operator (DEC-21's "+"/"−", or a plain
// ASCII hyphen if a question was authored with one). util.Combine treats "-" and "−"
// identically, so which minus-like rune is returned doesn't matter beyond opWord.
func opFromText(text string) (op string, ok bool) {
	switch {
	case strings.Contains(text, "+"):
		return "+", true
	case strings.Contains(text, "−"):
		return "−", true
	case strings.Contains(text, "-"):
		return "-", true
	}
	return "", false
}

// opWord is the past-tense verb the class-hint copy uses for the question's operator.
func opWord(op string) string {
	if op == "+" {
		return "added"
	}
	return "subtracted"
}

// applyOp combines two ints straight across (no common denominator), the mistake
// straight_across/no_common_denominator each model.
func applyOp(x, y int, op string) (int, bool) {
	switch op {
	case "+":
		return x + y, true
	case "-", "−":
		return x - y, true
	}
	return 0, false
}

// straightAcross is the straight_across misconception's candidate value:
// (a.Num op c.Num)/(a.Den op c.Den). false if the resulting denominator is zero (e.g.
// subtracting two equal denominators straight across) - that candidate is skipped,
// never divided.
func straightAcross(a, c util.Frac, op string) (util.Frac, bool) {
	num, ok := applyOp(a.Num, c.Num, op)
	if !ok {
		return util.Frac{}, false
	}
	den, ok := applyOp(a.Den, c.Den, op)
	if !ok || den == 0 {
		return util.Frac{}, false
	}
	return util.Frac{Num: num, Den: den}, true
}

// noCommonDenominator is the no_common_denominator misconception's candidate value:
// (a.Num op c.Num)/den, using one of the two original (nonzero, already-checked)
// denominators.
func noCommonDenominator(a, c util.Frac, den int, op string) (util.Frac, bool) {
	num, ok := applyOp(a.Num, c.Num, op)
	if !ok {
		return util.Frac{}, false
	}
	return util.Frac{Num: num, Den: den}, true
}

// matchRule classifies one wrong choice against a question, in the priority order
// 02 §B6 gives: equivalent_choice, straight_across, no_common_denominator. It returns
// "" when the question isn't hintable (not exactly 2 fractions plus an operator), the
// choice doesn't parse as a fraction, or no rule matches.
func matchRule(question, choice string) (rule string) {
	fracs := util.ParseFracs(question)
	if len(fracs) != 2 {
		return ""
	}
	op, ok := opFromText(question)
	if !ok {
		return ""
	}
	a, c := fracs[0], fracs[1]
	if a.Den == 0 || c.Den == 0 {
		return ""
	}
	v, ok := util.ParseChoice(choice)
	if !ok {
		return ""
	}
	x, ok := util.Combine(a, c, op)
	if !ok {
		return ""
	}
	if util.Equal(v, x) {
		return "equivalent_choice"
	}
	if sa, ok := straightAcross(a, c, op); ok && util.Equal(v, sa) {
		return "straight_across"
	}
	if nc, ok := noCommonDenominator(a, c, a.Den, op); ok && util.Equal(v, nc) {
		return "no_common_denominator"
	}
	if nc, ok := noCommonDenominator(a, c, c.Den, op); ok && util.Equal(v, nc) {
		return "no_common_denominator"
	}
	return ""
}

// classHintCopy is 02 §B6's exact per-rule copy for the question-level hint (1g).
func classHintCopy(rule string, op string, top ChoiceStat) string {
	switch rule {
	case "equivalent_choice":
		return fmt.Sprintf("Choice %s (%s) also equals the correct answer — consider replacing it so only one answer is correct.", top.Letter, top.Text)
	case "straight_across":
		return fmt.Sprintf("%d students %s the numerators and the denominators straight across.", top.Count, opWord(op))
	case "no_common_denominator":
		return fmt.Sprintf("%d students combined the numerators without finding a common denominator.", top.Count)
	}
	return ""
}

// ClassHint is the per-question misconception hint (1g). It needs the question text
// (exactly 2 fractions plus an operator) and every choice's tally. false means no
// hint: too few responses, no wrong picks at all, or a most-picked wrong choice that
// matches no rule and isn't concentrated enough to call "spread across".
func ClassHint(question string, choices []ChoiceStat) (string, bool) {
	fracs := util.ParseFracs(question)
	if len(fracs) != 2 {
		return "", false
	}
	op, ok := opFromText(question)
	if !ok {
		return "", false
	}

	total := 0
	for _, ch := range choices {
		total += ch.Count
	}
	if total < types.MinResponsesForHint {
		return "", false
	}

	var top ChoiceStat
	haveWrong := false
	wrongTotal := 0
	for _, ch := range choices {
		if ch.Correct {
			continue
		}
		wrongTotal += ch.Count
		if !haveWrong || ch.Count > top.Count {
			top = ch
			haveWrong = true
		}
	}
	if !haveWrong || wrongTotal == 0 {
		return "", false
	}

	if rule := matchRule(question, top.Text); rule != "" {
		return classHintCopy(rule, op, top), true
	}

	if top.Count*100 < 40*wrongTotal {
		return "Wrong answers were spread across the choices — that suggests guessing rather than one mistake.", true
	}

	return "", false
}

// studentHintCopy mirrors classHintCopy's rules, phrased for a single student (1h).
func studentHintCopy(rule string) string {
	switch rule {
	case "equivalent_choice":
		return "This student repeatedly picked a choice that also equals the correct answer - check these questions for duplicate correct values."
	case "straight_across":
		return "This student tends to add or subtract the numerators and the denominators straight across, without a common denominator."
	case "no_common_denominator":
		return "This student tends to combine the numerators without finding a common denominator."
	}
	return ""
}

// StudentHint is the per-student misconception hint (1h): it fires when at least 2 of
// a student's wrong answers, across the quiz, match the same rule. Checked in the same
// priority order as matchRule, so a student who matches multiple rules gets the
// highest-priority one.
func StudentHint(wrong []struct{ Question, Chosen string }) (string, bool) {
	counts := map[string]int{}
	for _, w := range wrong {
		if rule := matchRule(w.Question, w.Chosen); rule != "" {
			counts[rule]++
		}
	}
	for _, rule := range []string{"equivalent_choice", "straight_across", "no_common_denominator"} {
		if counts[rule] >= 2 {
			return studentHintCopy(rule), true
		}
	}
	return "", false
}
