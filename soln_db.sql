CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

CREATE TABLE IF NOT EXISTS users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    firstname VARCHAR(50),
    lastname VARCHAR(50),
    username VARCHAR(50) NOT NULL,
    usertype ENUM('teacher', 'student') NOT NULL,
    section  VARCHAR(50),
    class_number VARCHAR(50), 
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

CREATE TABLE IF NOT EXISTS fraction_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY,
    minigame_id INT,
    question_text VARCHAR(500),
    fraction1_numerator INT NOT NULL,
    fraction1_denominator INT NOT NULL,
    fraction2_numerator INT NOT NULL,
    fraction2_denominator INT NOT NULL
);

CREATE TABLE IF NOT EXISTS fraction_responses (
    statistic_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT,
    minigame_id INT,
    question_id INT,
    student_id INT,
    num_right_attempts INT,
    num_wrong_attempts INT
);
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
    FOREIGN KEY (question_id) REFERENCES multiple_choice_questions(question_id),
    FOREIGN KEY (student_id) REFERENCES users(user_id),
    FOREIGN KEY (choice_id) REFERENCES multiple_choice_choices(choice_id)
);

CREATE TABLE IF NOT EXISTS multiple_choice_scores (
  statistic_id INT AUTO_INCREMENT PRIMARY KEY,
  classroom_id INT NOT NULL,
  minigame_id INT NOT NULL,
  student_id INT NOT NULL,
  score INT NOT NULL,
  FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
  FOREIGN KEY (student_id) REFERENCES users(user_id)
);

-- Insert sample teacher data into users table
INSERT INTO users (username, usertype, password) VALUES
('user1', 'teacher', 'pw'),
('user2', 'teacher', 'pw');

-- insert student data
INSERT INTO users (username, firstname, lastname, usertype, class_number, section, password) VALUES
('user3', 'John', 'Johnson', 'student', '1', '1', 'pw'),
('user4', 'Mike', 'Tyson', 'student', '1', '1', 'pw'),
('user5', 'Joe', 'Seph', 'student', '1', '2', 'pw');

-- Insert initial data into classrooms table
INSERT INTO classrooms (classroom_name, section, description, teacher_id) VALUES
('Math101', '1', 'Lorem Ipsum', 1),
('Remedial Class 1', '2', 'Lorem Ipsum', 2);

-- Insert initial data into enrollments table
INSERT INTO enrollments (classroom_id, student_id) VALUES 
(1, 3),
(1, 4),
(2, 5);

-- insert data for simple fraction questions
INSERT INTO fraction_questions (minigame_id, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES
(1, 5, 4, 3, 4),
(1, 7, 10, 1, 5),
(1, 9, 5, 4, 5),
(2, 1, 2, 1, 2),
(2, 1, 3, 3, 1), 
(2, 2, 2, 2, 2);

-- insert data for worded fraction questions
INSERT INTO fraction_questions (minigame_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES
(3, "what is 2/4 + 2/4?", 2, 4, 2, 4),
(3, "what is 7/10 + 1/5?", 7, 10, 1, 5),
(3, "what is 9/5 + 4/5", 9, 5, 4, 5),
(4, "what is 3/4 + 3/4?", 3, 4, 3, 4),
(4, "what is 8/10 + 2/5?", 8, 10, 2, 5),
(4, "what is 8/5 + 3/5", 8, 5, 3, 5);

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

INSERT INTO multiple_choice_questions (minigame_id, question_text) VALUES
(11, 'What is 1/2 - 1/4?'),
(11, 'What is 2/3 - 1/3?'),
(11, 'What is 4/5 - 1/5?'),
(11, 'What is 2/3 - 1/6?'),
(11, 'What is 5/8 - 1/8?'),
(11, 'What is 2/2 - 1/2?'),
(11, 'What is 5/6 - 1/3?'),
(11, 'What is 3/4 - 1/4?'),
(11, 'What is 7/10 - 3/10?'),
(11, 'What is 3/4 - 1/2?');

INSERT INTO multiple_choice_choices (question_id, choice_text, is_correct) VALUES
(11, '1/2', FALSE),
(11, '1/4', TRUE),
(11, '1/8', FALSE),
(11, '3/4', FALSE),

(12, '1/3', TRUE),
(12, '2/3', FALSE),
(12, '1/2', FALSE),
(12, '1', FALSE),

(13, '2/5', FALSE),
(13, '3/5', TRUE),
(13, '1/5', FALSE),
(13, '4/5', FALSE),

(14, '1/6', FALSE),
(14, '1/2', TRUE),
(14, '1/3', FALSE),
(14, '5/6', FALSE),

(15, '4/8', FALSE),
(15, '3/8', FALSE),
(15, '5/8', FALSE),
(15, '1/2', TRUE),

(16, '1/2', TRUE),
(16, '1/4', FALSE),
(16, '3/4', FALSE),
(16, '2/4', FALSE),

(17, '1/2', TRUE),
(17, '1/3', FALSE),
(17, '2/3', FALSE),
(17, '5/6', FALSE),

(18, '2/4', FALSE),
(18, '1/4', FALSE),
(18, '3/4', FALSE),
(18, '1/2', TRUE),

(19, '1/5', FALSE),
(19, '4/10', TRUE),
(19, '7/10', FALSE),
(19, '1/2', FALSE),

(20, '1/4', TRUE),
(20, '1/2', FALSE),
(20, '1/3', FALSE),
(20, '2/4', FALSE);


-- test values for statistics quiz
INSERT INTO multiple_choice_scores (student_id, classroom_id, minigame_id, score) VALUES
(3, 1, 5, 1),
(4, 1, 5, 2);

-- sample response values for simple fraction questions with minigame ID 1
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 1, 1, 3, 2, 1),
(1, 1, 2, 3, 3, 1),
(1, 1, 3, 3, 4, 1);

-- sample response values for simple fraction questions with minigame ID 2
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 2, 4, 3, 2, 1),
(1, 2, 5, 3, 3, 1),
(1, 2, 6, 3, 4, 1);

-- sample response values for worded fraction questions with minigame ID 3
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 3, 7, 3, 2, 1),
(1, 3, 8, 3, 3, 1),
(1, 3, 9, 3, 4, 1);

-- sample response values for worded fraction questions with minigame ID 4
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 4, 10, 3, 2, 1),
(1, 4, 11, 3, 3, 1),
(1, 4, 12, 3, 4, 1);