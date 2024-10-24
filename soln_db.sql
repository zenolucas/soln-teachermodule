CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

CREATE TABLE IF NOT EXISTS users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    section  VARCHAR(50),
    usertype ENUM('student', 'teacher') NOT NULL
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
    FOREIGN KEY (teacher_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS enrollments (
    enrollment_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT,
    student_id INT,
    FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
    FOREIGN KEY (student_id) REFERENCES users(user_id),
    UNIQUE KEY unique_enrollment (classroom_id, student_id)
);

CREATE TABLE IF NOT EXISTS levels (
    level_id INT AUTO_INCREMENT PRIMARY KEY,
    level_name VARCHAR(100)
);

CREATE TABLE IF NOT EXISTS minigames (
    minigame_id INT AUTO_INCREMENT PRIMARY KEY,
    minigame_name   VARCHAR(100) NOT NULL,
    minigame_type   ENUM('quest', 'miniboss') NOT NULL
);

CREATE TABLE IF NOT EXISTS level_minigames (
    level_id INT, 
    minigame_id INT, 
    FOREIGN KEY (level_id) REFERENCES levels(level_id),
    FOREIGN KEY (minigame_id) REFERENCES minigames(minigame_id)
);

CREATE TABLE IF NOT EXISTS multiple_choice_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY, 
    minigame_id INT,
    question_text   VARCHAR(500) NOT NULL,
    correct_answer  VARCHAR(200) NOT NULL
);

CREATE TABLE IF NOT EXISTS multiple_choice_choices (
    choice_id INT AUTO_INCREMENT PRIMARY KEY,
    question_id INT, 
    option_1 VARCHAR(20),
    option_2 VARCHAR(20),
    option_3 VARCHAR(20),
    option_4 VARCHAR(20),
    FOREIGN KEY (question_id) REFERENCES multiple_choice_questions(question_id)
);

CREATE TABLE IF NOT EXISTS fraction_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY,
    minigame_id INT,
    fraction1_numerator INT NOT NULL,
    fraction1_denominator INT NOT NULL,
    fraction2_numerator INT NOT NULL,
    fraction2_denominator INT NOT NULL
);

CREATE TABLE IF NOT EXISTS statistics (
  statistic_id INT AUTO_INCREMENT PRIMARY KEY,
  classroom_id INT NOT NULL,
  username varchar(50) NOT NULL,
  minigameID INT NOT NULL,
  score INT NOT NULL
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
INSERT INTO users (username, password, usertype) VALUES
('user1', 'pw', 'teacher'),
('user2', 'password_hash2', 'teacher');

-- insert student data
INSERT INTO users (username, section, password, usertype) VALUES
('user3', '1', 'pw', 'student'),
('user4', '1', 'pw', 'student'),
('user5', '2', 'pw', 'student');

-- Insert initial data into subjects table
INSERT INTO classrooms (classroom_name, section, description, teacher_id) VALUES
('Math101', '1', 'Lorem Ipsum', 1),
('Remedial Class 1', '2', 'Lorem Ipsum', 2);

-- Insert initial data into enrollments table
INSERT INTO enrollments (classroom_id, student_id) VALUES 
(1, 3),
(1, 4),
(2, 5);

INSERT INTO multiple_choice_questions (minigame_id, question_text, correct_answer) VALUES 
(5, 'What is 1/2 + 1/2 ?', '1'),
(5, 'What is 1/3 + 1/3 ?', '2/3'),
(5, 'What is 1/4 + 1/4 ?', '1/2'),
(5, 'What is 1/5 + 1/5 ?', '2/5'),
(5, 'What is 1/6 + 1/6 ?', '1/3'),
(5, 'What is 1/7 + 1/7 ?', '2/7'),
(5, 'What is 1/8 + 1/8 ?', '1/4'),
(5, 'What is 1/9 + 1/9 ?', '2/9'),
(5, 'What is 1/10 + 1/10 ?', '1/5'),
(5, 'What is 1/2 + 1/4 ?', '3/4');

INSERT INTO multiple_choice_choices (question_id, option_1, option_2, option_3, option_4) VALUES 
(1, '1/2', '1/3', '1', '2'),
(2, '1/2', '1/3', '1/4', '2/3'),
(3, '1/2', '1/4', '3/4', '1'),
(4, '1/5', '1/2', '2/5', '3/5'),
(5, '1/3', '1/6', '1/2', '2/3'),
(6, '2/7', '1/7', '1/5', '1/6'),
(7, '1/4', '1/8', '1/2', '1/3'),
(8, '2/9', '1/9', '1/2', '1/3'),
(9, '1/5', '1/10', '2/5', '3/5'),
(10, '1/2', '1/4', '3/4', '2/3');

-- test values for statistics
INSERT INTO statistics (username, classroom_id, minigameID, score) VALUES
("USER1", 1, 5, 1),
("USER2", 1, 5, 2),
("USER3", 1, 5, 3),
("USER4", 1, 5, 4),
("USER5", 1, 5, 5),
("USER6", 1, 5, 6),
("USER7", 1, 5, 4),
("USER8", 1, 5, 4),
("USER9", 1, 5, 4),
("USER10", 1, 5, 4);