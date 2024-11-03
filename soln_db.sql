CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

CREATE TABLE IF NOT EXISTS teachers (
    teacher_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS students (
    student_id INT AUTO_INCREMENT PRIMARY KEY,
    firstname VARCHAR(50) NOT NULL,
    lastname VARCHAR(50) NOT NULL,
    username VARCHAR(50) NOT NULL,
    section  VARCHAR(50) NOT NULL,
    class_number VARCHAR(50) NOT NULL, 
    password VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS classrooms (
    classroom_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_name VARCHAR(100) NOT NULL,
    section VARCHAR(100),
    description VARCHAR(200),
    teacher_id INT NOT NULL,
    FOREIGN KEY (teacher_id) REFERENCES teachers(teacher_id)
);

CREATE TABLE IF NOT EXISTS enrollments (
    enrollment_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT,
    student_id INT,
    FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
    FOREIGN KEY (student_id) REFERENCES students(student_id),
    UNIQUE KEY unique_enrollment (classroom_id, student_id)
);

-- CREATE TABLE IF NOT EXISTS minigames (
--     minigame_id INT AUTO_INCREMENT PRIMARY KEY,
--     minigame_name   VARCHAR(100) NOT NULL
-- );

CREATE TABLE IF NOT EXISTS multiple_choice_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY, 
    minigame_id INT,
    question_text VARCHAR(500) NOT NULL
);

-- Table to store choices
CREATE TABLE IF NOT EXISTS multiple_choice_choices (
    choice_id INT AUTO_INCREMENT PRIMARY KEY,
    question_id INT, 
    choice_text VARCHAR(20) NOT NULL,
    is_correct BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (question_id) REFERENCES multiple_choice_questions(question_id)
);

CREATE TABLE IF NOT EXISTS multiple_choice_responses (
    response_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT,
    minigame_id INT,
    question_id INT,
    student_id INT,
    choice_id INT,
    FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
    -- FOREIGN KEY (minigame_id) REFERENCES minigames(minigame_id),
    FOREIGN KEY (question_id) REFERENCES multiple_choice_questions(question_id),
    FOREIGN KEY (student_id) REFERENCES students(student_id),
    FOREIGN KEY (choice_id) REFERENCES multiple_choice_choices(choice_id)
);

CREATE TABLE IF NOT EXISTS quiz_scores (
  statistic_id INT AUTO_INCREMENT PRIMARY KEY,
  classroom_id INT NOT NULL,
  minigame_id INT NOT NULL,
  student_id INT NOT NULL,
  score INT NOT NULL,
  FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
  FOREIGN KEY (student_id) REFERENCES students(student_id)
);

CREATE TABLE IF NOT EXISTS fraction_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY,
    minigame_id INT,
    fraction1_numerator INT NOT NULL,
    fraction1_denominator INT NOT NULL,
    fraction2_numerator INT NOT NULL,
    fraction2_denominator INT NOT NULL
);

CREATE TABLE IF NOT EXISTS fraction_statistics (
    statistic_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT,
    minigame_id INT,
    question_id INT,
    student_id INT,
    num_right_attempts INT,
    num_wrong_attempts INT
);

CREATE TABLE IF NOT EXISTS worded_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY,
    minigame_id INT,
    question_text VARCHAR(500),
    fraction1_numerator INT NOT NULL,
    fraction1_denominator INT NOT NULL,
    fraction2_numerator INT NOT NULL,
    fraction2_denominator INT NOT NULL
);

CREATE TABLE IF NOT EXISTS worded_statistics (
    statistic_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT,
    minigame_id INT,
    question_id INT,
    student_id INT,
    num_right_attempts INT,
    num_wrong_attempts INT
);

-- insert data for worded questions
INSERT INTO worded_questions (minigame_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES
(3, "what is 2/4 + 2/4?", 2, 4, 2, 4),
(3, "what is 7/10 + 1/5?", 7, 10, 1, 5),
(3, "what is 9/5 + 4/5", 9, 5, 4, 5),
(4, "what is 3/4 + 3/4?", 3, 4, 3, 4),
(4, "what is 8/10 + 2/5?", 8, 10, 2, 5),
(4, "what is 8/5 + 3/5", 8, 5, 3, 5);

-- insert data for fraction_questions
INSERT INTO fraction_questions (minigame_id, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES
(1, 5, 4, 3, 4),
(1, 7, 10, 1, 5),
(1, 9, 5, 4, 5),
(2, 1, 2, 1, 2),
(2, 1, 3, 3, 1), 
(2, 2, 2, 2, 2);

-- Suggestions: 
-- Superadmin and admin


-- Insert teacher data into users table
INSERT INTO teachers (username, password) VALUES
('user1', 'pw'),
('user2', 'pw');

-- insert student data
INSERT INTO students (username, firstname, lastname, class_number, section, password) VALUES
('user3', 'John', 'Johnson', '1', '1', 'pw'),
('user4', 'Mike', 'Tyson', '1', '1', 'pw'),
('user5', 'Joe', 'Seph', '1', '1', 'pw');

-- Insert initial data into subjects table
INSERT INTO classrooms (classroom_name, section, description, teacher_id) VALUES
('Math101', '1', 'Lorem Ipsum', 1),
('Remedial Class 1', '2', 'Lorem Ipsum', 2);

-- Insert initial data into enrollments table
INSERT INTO enrollments (classroom_id, student_id) VALUES 
(1, 1),
(1, 2),
(2, 3);

INSERT INTO multiple_choice_questions (minigame_id, question_text) VALUES 
(5, 'What is 1/2 + 1/2 ?'),
(5, 'What is 1/3 + 1/3 ?'),
(5, 'What is 1/4 + 1/4 ?'),
(5, 'What is 1/5 + 1/5 ?'),
(5, 'What is 1/6 + 1/6 ?'),
(5, 'What is 1/7 + 1/7 ?'),
(5, 'What is 1/8 + 1/8 ?'),
(5, 'What is 1/9 + 1/9 ?'),
(5, 'What is 1/10 + 1/10 ?'),
(5, 'What is 1/2 + 1/4 ?');

INSERT INTO multiple_choice_choices (question_id, choice_text, is_correct) VALUES 
(1, '1/2', FALSE),
(1, '1/3', FALSE),
(1, '1', TRUE),
(1, '2', FALSE),

(2, '1/2', FALSE),
(2, '1/3', FALSE),
(2, '1/4', FALSE),
(2, '2/3', TRUE),

(3, '1/2', TRUE),
(3, '1/4', FALSE),
(3, '3/4', FALSE),
(3, '1', FALSE),

(4, '1/5', FALSE),
(4, '1/2', FALSE),
(4, '2/5', TRUE),
(4, '3/5', FALSE),

(5, '1/3', TRUE),
(5, '1/6', FALSE),
(5, '1/2', FALSE),
(5, '2/3', FALSE),

(6, '2/7', TRUE),
(6, '1/7', FALSE),
(6, '1/5', FALSE),
(6, '1/6', FALSE),

(7, '1/4', TRUE),
(7, '1/8', FALSE),
(7, '1/2', FALSE),
(7, '1/3', FALSE),

(8, '2/9', TRUE),
(8, '1/9', FALSE),
(8, '1/2', FALSE),
(8, '1/3', FALSE),

(9, '1/5', TRUE),
(9, '1/10', FALSE),
(9, '2/5', FALSE),
(9, '3/5', FALSE),

(10, '1/2', FALSE),
(10, '1/4', FALSE),
(10, '3/4', TRUE),
(10, '2/3', FALSE);

-- test values for statistics quiz
INSERT INTO quiz_scores (student_id, classroom_id, minigame_id, score) VALUES
(3, 1, 5, 1),
(2, 1, 5, 2);

-- test values for statistics fractoins FOR MINIGAME ID 1
INSERT INTO fraction_statistics (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 1, 1, 1, 2, 1),
(1, 1, 2, 1, 3, 1),
(1, 1, 3, 1, 4, 1);

-- test values for statistics fractoins FOR MINIGAME ID 2
INSERT INTO fraction_statistics (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 2, 4, 1, 2, 1),
(1, 2, 5, 1, 3, 1),
(1, 2, 6, 1, 4, 1);

-- test values for worded statistics FOR MINIGAME ID 3
INSERT INTO worded_statistics (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 3, 1, 1, 2, 1),
(1, 3, 2, 1, 3, 1),
(1, 3, 3, 1, 4, 1);

-- test values for worded statistics FOR MINIGAME ID 4
INSERT INTO worded_statistics(classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 4, 4, 1, 2, 1),
(1, 4, 5, 1, 3, 1),
(1, 4, 6, 1, 4, 1);

-- you know what? Let's just merge worded statistics into fraction_statistics