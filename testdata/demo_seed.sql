-- Demo data for the Teacher Portal revamp (T0.2). Additive only: run after soln_db.sql.
-- Purpose: exercise the revamped screens (progress, flags, quiz stats, hints) against a
-- classroom with enough students, scenes played and quiz attempts to be interesting -
-- soln_db.sql alone has only 2 enrolled students in classroom 1 and zero
-- multiple_choice_responses rows.
--
-- Apply: cd ~/.local/share/soln-devtools/setupdb && go run . --reset --demo
-- (loads soln_db.sql, then this file, against the soln_db database).
--
-- Expected numbers (classroom 1, verified live against a fresh --reset --demo load):
-- - Enrolled: 19 (students 3, 4, 6-22; student 5 stays in classroom 2 only)
-- - World placement, per DEC-10's currentScene rule: 0 in World 1, 13 in World 2, 3 in World 3
--   (reached minigame 11 with a score, not yet scored on 12), 3 completed (scored on 12).
--   World 1 is unreachable here: soln_db.sql already gives every student (3-22) a minigame-5
--   score, which DEC-10 always bumps into World 2 - see TASKS.md discrepancy X16.
-- - Flagged students (quiz_below_60, low_accuracy and/or got_stuck): 11 - students 4, 7, 8, 9, 14, 16,
--   17, 18, 19, 21 from this file, plus student 3, whose low_accuracy flags come from soln_db.sql's own
--   base rows for minigames 1-4 (33% right, 9-12 attempts each). The other 9 from this file are all
--   quiz_below_60 (mg5: 4, 8, 9, 14, 17, 19; mg11: 16, 21; mg12: 18); 9 and 19 are also
--   low_accuracy (mg7: 1/7 right = 14%; mg8: 1/6 right = 17%). Student 7 ran out of energy twice on
--   mg6, so they're got_stuck there (and low_accuracy: 6 right / 12 attempts = 50%).
-- - Minigame 5 (latest attempt per student, DEC-9): average 62.1%, median 60% (score 6/10),
--   6 of 19 students below 60%.
-- - Took the quiz (has a score row): minigame 5 - 19; minigame 11 - 6; minigame 12 - 3.
-- Enroll students 6-22 in classroom 1 (3, 4 are already enrolled in soln_db.sql).
-- Classroom 1 then has 19 students: 3, 4, 6-22. Student 5 stays in classroom 2 only,
-- to exercise the "In Classroom 2" label on the add-students search (T2.4a).
INSERT INTO enrollments (classroom_id, student_id) VALUES
(1, 6),
(1, 7),
(1, 8),
(1, 9),
(1, 10),
(1, 11),
(1, 12),
(1, 13),
(1, 14),
(1, 15),
(1, 16),
(1, 17),
(1, 18),
(1, 19),
(1, 20),
(1, 21),
(1, 22);
-- Extra World-2 scene data (scenes 6-10), so students spread across World 2 instead of
-- all sitting at its first scene. Joined by minigame_id/classroom_id, never hard-coded
-- question_ids (matches soln_db.sql's fraction_questions rows for classroom 1).
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 3, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 3, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 4, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 7, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
-- Student 7 also ran out of energy (game over: 0 right, 3 wrong) on two of those questions, so the
-- "Got stuck" flag (2+ game overs on one scene) has a demo case.
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 7, 0, 3
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1 ORDER BY question_id LIMIT 2;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 7, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 7, 2, 0
FROM fraction_questions WHERE minigame_id = 8 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 9, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 10, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 10, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 11, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 11, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 11, 2, 0
FROM fraction_questions WHERE minigame_id = 8 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 11, 2, 0
FROM fraction_questions WHERE minigame_id = 9 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 11, 2, 0
FROM fraction_questions WHERE minigame_id = 10 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 13, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 14, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 14, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 19, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 19, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 20, 2, 0
FROM fraction_questions WHERE minigame_id = 6 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 20, 2, 0
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 20, 2, 0
FROM fraction_questions WHERE minigame_id = 8 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 20, 2, 0
FROM fraction_questions WHERE minigame_id = 9 AND classroom_id = 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 20, 2, 0
FROM fraction_questions WHERE minigame_id = 10 AND classroom_id = 1;

-- Low-accuracy scenes (>=5 attempts, <60% right) for the low_accuracy flag (C2).
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 9, 1, 3
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 0;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 9, 0, 2
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 9, 0, 1
FROM fraction_questions WHERE minigame_id = 7 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 2;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 19, 1, 2
FROM fraction_questions WHERE minigame_id = 8 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 0;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 19, 0, 2
FROM fraction_questions WHERE minigame_id = 8 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 1;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 19, 0, 1
FROM fraction_questions WHERE minigame_id = 8 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 2;

-- Replay: student 3 answers minigame 1's first question twice (two separate playthroughs).
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 3, 1, 1
FROM fraction_questions WHERE minigame_id = 1 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 0;
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts)
SELECT classroom_id, minigame_id, question_id, 3, 0, 1
FROM fraction_questions WHERE minigame_id = 1 AND classroom_id = 1
ORDER BY question_id LIMIT 1 OFFSET 0;

-- Minigame 5 retakes: 2 students' later attempt scores lower than their first (tests V3,
-- "count the latest attempt" - DEC-9). soln_db.sql's original rows for students 8 and 9
-- stay; these are inserted after them, so MAX(statistic_id) picks these up as latest.
INSERT INTO multiple_choice_scores (classroom_id, minigame_id, student_id, score) VALUES
(1, 5, 8, 2),
(1, 5, 9, 1);
-- Minigame 11 (World 2 quiz) scores, for students who reached it.
INSERT INTO multiple_choice_scores (classroom_id, minigame_id, student_id, score) VALUES
(1, 11, 15, 7),
(1, 11, 16, 5),
(1, 11, 17, 8),
(1, 11, 18, 6),
(1, 11, 21, 4),
(1, 11, 22, 9);
-- Minigame 12 (Final Boss) scores, for students who reached and completed it.
INSERT INTO multiple_choice_scores (classroom_id, minigame_id, student_id, score) VALUES
(1, 12, 17, 7),
(1, 12, 18, 5),
(1, 12, 22, 9);
-- Responses for each quiz's latest scored attempt (mg 5, 11, 12): exactly `score` correct
-- choices out of 10 per student, so GetQuizResponseStatistics/misconception hints have data.
-- On mg5 Q1 ("1/2 + 1/2") and Q10 ("1/2 + 1/4"), students who get the question wrong pick
-- choice "1/2" (each question's first wrong choice), giving >=5 picks on each (step 6).
INSERT INTO multiple_choice_responses (classroom_id, minigame_id, question_id, student_id, choice_id) VALUES
(1, 5, 1, 3, 1),
(1, 5, 2, 3, 5),
(1, 5, 3, 3, 9),
(1, 5, 4, 3, 15),
(1, 5, 5, 3, 17),
(1, 5, 6, 3, 21),
(1, 5, 7, 3, 25),
(1, 5, 8, 3, 29),
(1, 5, 9, 3, 33),
(1, 5, 10, 3, 37),
(1, 5, 1, 4, 1),
(1, 5, 2, 4, 5),
(1, 5, 3, 4, 10),
(1, 5, 4, 4, 13),
(1, 5, 5, 4, 17),
(1, 5, 6, 4, 21),
(1, 5, 7, 4, 25),
(1, 5, 8, 4, 29),
(1, 5, 9, 4, 33),
(1, 5, 10, 4, 37),
(1, 5, 1, 6, 1),
(1, 5, 2, 6, 5),
(1, 5, 3, 6, 10),
(1, 5, 4, 6, 15),
(1, 5, 5, 6, 17),
(1, 5, 6, 6, 21),
(1, 5, 7, 6, 25),
(1, 5, 8, 6, 29),
(1, 5, 9, 6, 33),
(1, 5, 10, 6, 37),
(1, 5, 1, 7, 1),
(1, 5, 2, 7, 8),
(1, 5, 3, 7, 9),
(1, 5, 4, 7, 15),
(1, 5, 5, 7, 17),
(1, 5, 6, 7, 21),
(1, 5, 7, 7, 25),
(1, 5, 8, 7, 29),
(1, 5, 9, 7, 33),
(1, 5, 10, 7, 39),
(1, 5, 1, 8, 1),
(1, 5, 2, 8, 5),
(1, 5, 3, 8, 10),
(1, 5, 4, 8, 13),
(1, 5, 5, 8, 18),
(1, 5, 6, 8, 22),
(1, 5, 7, 8, 26),
(1, 5, 8, 8, 29),
(1, 5, 9, 8, 33),
(1, 5, 10, 8, 37),
(1, 5, 1, 9, 1),
(1, 5, 2, 9, 5),
(1, 5, 3, 9, 10),
(1, 5, 4, 9, 13),
(1, 5, 5, 9, 18),
(1, 5, 6, 9, 22),
(1, 5, 7, 9, 26),
(1, 5, 8, 9, 30),
(1, 5, 9, 9, 33),
(1, 5, 10, 9, 37),
(1, 5, 1, 10, 1),
(1, 5, 2, 10, 5),
(1, 5, 3, 10, 9),
(1, 5, 4, 10, 15),
(1, 5, 5, 10, 17),
(1, 5, 6, 10, 21),
(1, 5, 7, 10, 25),
(1, 5, 8, 10, 29),
(1, 5, 9, 10, 33),
(1, 5, 10, 10, 37),
(1, 5, 1, 11, 3),
(1, 5, 2, 11, 8),
(1, 5, 3, 11, 9),
(1, 5, 4, 11, 15),
(1, 5, 5, 11, 17),
(1, 5, 6, 11, 21),
(1, 5, 7, 11, 25),
(1, 5, 8, 11, 29),
(1, 5, 9, 11, 33),
(1, 5, 10, 11, 39),
(1, 5, 1, 12, 1),
(1, 5, 2, 12, 5),
(1, 5, 3, 12, 10),
(1, 5, 4, 12, 15),
(1, 5, 5, 12, 17),
(1, 5, 6, 12, 21),
(1, 5, 7, 12, 25),
(1, 5, 8, 12, 29),
(1, 5, 9, 12, 33),
(1, 5, 10, 12, 37),
(1, 5, 1, 13, 1),
(1, 5, 2, 13, 8),
(1, 5, 3, 13, 9),
(1, 5, 4, 13, 15),
(1, 5, 5, 13, 17),
(1, 5, 6, 13, 21),
(1, 5, 7, 13, 25),
(1, 5, 8, 13, 29),
(1, 5, 9, 13, 33),
(1, 5, 10, 13, 37),
(1, 5, 1, 14, 1),
(1, 5, 2, 14, 5),
(1, 5, 3, 14, 10),
(1, 5, 4, 14, 13),
(1, 5, 5, 14, 17),
(1, 5, 6, 14, 21),
(1, 5, 7, 14, 25),
(1, 5, 8, 14, 29),
(1, 5, 9, 14, 33),
(1, 5, 10, 14, 37),
(1, 5, 1, 15, 1),
(1, 5, 2, 15, 5),
(1, 5, 3, 15, 9),
(1, 5, 4, 15, 15),
(1, 5, 5, 15, 17),
(1, 5, 6, 15, 21),
(1, 5, 7, 15, 25),
(1, 5, 8, 15, 29),
(1, 5, 9, 15, 33),
(1, 5, 10, 15, 37),
(1, 5, 1, 16, 1),
(1, 5, 2, 16, 5),
(1, 5, 3, 16, 10),
(1, 5, 4, 16, 15),
(1, 5, 5, 16, 17),
(1, 5, 6, 16, 21),
(1, 5, 7, 16, 25),
(1, 5, 8, 16, 29),
(1, 5, 9, 16, 33),
(1, 5, 10, 16, 37),
(1, 5, 1, 17, 1),
(1, 5, 2, 17, 5),
(1, 5, 3, 17, 10),
(1, 5, 4, 17, 13),
(1, 5, 5, 17, 18),
(1, 5, 6, 17, 21),
(1, 5, 7, 17, 25),
(1, 5, 8, 17, 29),
(1, 5, 9, 17, 33),
(1, 5, 10, 17, 37),
(1, 5, 1, 18, 1),
(1, 5, 2, 18, 8),
(1, 5, 3, 18, 9),
(1, 5, 4, 18, 15),
(1, 5, 5, 18, 17),
(1, 5, 6, 18, 21),
(1, 5, 7, 18, 25),
(1, 5, 8, 18, 29),
(1, 5, 9, 18, 33),
(1, 5, 10, 18, 39),
(1, 5, 1, 19, 1),
(1, 5, 2, 19, 5),
(1, 5, 3, 19, 10),
(1, 5, 4, 19, 13),
(1, 5, 5, 19, 18),
(1, 5, 6, 19, 22),
(1, 5, 7, 19, 26),
(1, 5, 8, 19, 29),
(1, 5, 9, 19, 33),
(1, 5, 10, 19, 37),
(1, 5, 1, 20, 1),
(1, 5, 2, 20, 8),
(1, 5, 3, 20, 9),
(1, 5, 4, 20, 15),
(1, 5, 5, 20, 17),
(1, 5, 6, 20, 21),
(1, 5, 7, 20, 25),
(1, 5, 8, 20, 29),
(1, 5, 9, 20, 33),
(1, 5, 10, 20, 37),
(1, 5, 1, 21, 1),
(1, 5, 2, 21, 5),
(1, 5, 3, 21, 10),
(1, 5, 4, 21, 15),
(1, 5, 5, 21, 17),
(1, 5, 6, 21, 21),
(1, 5, 7, 21, 25),
(1, 5, 8, 21, 29),
(1, 5, 9, 21, 33),
(1, 5, 10, 21, 37),
(1, 5, 1, 22, 3),
(1, 5, 2, 22, 8),
(1, 5, 3, 22, 9),
(1, 5, 4, 22, 15),
(1, 5, 5, 22, 17),
(1, 5, 6, 22, 21),
(1, 5, 7, 22, 25),
(1, 5, 8, 22, 29),
(1, 5, 9, 22, 33),
(1, 5, 10, 22, 39),
(1, 11, 11, 15, 41),
(1, 11, 12, 15, 46),
(1, 11, 13, 15, 49),
(1, 11, 14, 15, 54),
(1, 11, 15, 15, 60),
(1, 11, 16, 15, 61),
(1, 11, 17, 15, 65),
(1, 11, 18, 15, 72),
(1, 11, 19, 15, 74),
(1, 11, 20, 15, 77),
(1, 11, 11, 16, 41),
(1, 11, 12, 16, 46),
(1, 11, 13, 16, 49),
(1, 11, 14, 16, 53),
(1, 11, 15, 16, 57),
(1, 11, 16, 16, 61),
(1, 11, 17, 16, 65),
(1, 11, 18, 16, 72),
(1, 11, 19, 16, 74),
(1, 11, 20, 16, 77),
(1, 11, 11, 17, 41),
(1, 11, 12, 17, 46),
(1, 11, 13, 17, 50),
(1, 11, 14, 17, 54),
(1, 11, 15, 17, 60),
(1, 11, 16, 17, 61),
(1, 11, 17, 17, 65),
(1, 11, 18, 17, 72),
(1, 11, 19, 17, 74),
(1, 11, 20, 17, 77),
(1, 11, 11, 18, 41),
(1, 11, 12, 18, 46),
(1, 11, 13, 18, 49),
(1, 11, 14, 18, 53),
(1, 11, 15, 18, 60),
(1, 11, 16, 18, 61),
(1, 11, 17, 18, 65),
(1, 11, 18, 18, 72),
(1, 11, 19, 18, 74),
(1, 11, 20, 18, 77),
(1, 11, 11, 21, 41),
(1, 11, 12, 21, 46),
(1, 11, 13, 21, 49),
(1, 11, 14, 21, 53),
(1, 11, 15, 21, 57),
(1, 11, 16, 21, 62),
(1, 11, 17, 21, 65),
(1, 11, 18, 21, 72),
(1, 11, 19, 21, 74),
(1, 11, 20, 21, 77),
(1, 11, 11, 22, 41),
(1, 11, 12, 22, 45),
(1, 11, 13, 22, 50),
(1, 11, 14, 22, 54),
(1, 11, 15, 22, 60),
(1, 11, 16, 22, 61),
(1, 11, 17, 22, 65),
(1, 11, 18, 22, 72),
(1, 11, 19, 22, 74),
(1, 11, 20, 22, 77),
(1, 12, 21, 17, 81),
(1, 12, 22, 17, 86),
(1, 12, 23, 17, 90),
(1, 12, 24, 17, 94),
(1, 12, 25, 17, 97),
(1, 12, 26, 17, 101),
(1, 12, 27, 17, 105),
(1, 12, 28, 17, 109),
(1, 12, 29, 17, 113),
(1, 12, 30, 17, 117),
(1, 12, 21, 18, 81),
(1, 12, 22, 18, 86),
(1, 12, 23, 18, 90),
(1, 12, 24, 18, 93),
(1, 12, 25, 18, 98),
(1, 12, 26, 18, 101),
(1, 12, 27, 18, 105),
(1, 12, 28, 18, 109),
(1, 12, 29, 18, 113),
(1, 12, 30, 18, 117),
(1, 12, 21, 22, 81),
(1, 12, 22, 22, 85),
(1, 12, 23, 22, 89),
(1, 12, 24, 22, 94),
(1, 12, 25, 22, 97),
(1, 12, 26, 22, 101),
(1, 12, 27, 22, 105),
(1, 12, 28, 22, 109),
(1, 12, 29, 22, 113),
(1, 12, 30, 22, 117);
