CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

CREATE TABLE IF NOT EXISTS users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    section  VARCHAR(50),
    usertype ENUM('student', 'teacher') NOT NULL
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
    question_text   VARCHAR(200) NOT NULL,
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
(1, 'What is 1/2 + 1/2 ?', '1'),
(1, 'What is 1/3 + 1/3 ?', '2/3');

INSERT INTO multiple_choice_choices (question_id, option_1, option_2, option_3, option_4) VALUES 
(1, '1/2', '1/3', '1', '2'),
(2, '1/2', '1/3', '1/4', '2/3');