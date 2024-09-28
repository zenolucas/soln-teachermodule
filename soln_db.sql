CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

CREATE TABLE IF NOT EXISTS users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    usertype ENUM('student', 'teacher') NOT NULL
);

CREATE TABLE IF NOT EXISTS classrooms (
    classroom_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_name VARCHAR(100) NOT NULL,
    section VARCHAR(100),
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

CREATE TABLE IF NOT EXISTS questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY, 
    minigame_id INT,
    question_text   VARCHAR(200) NOT NULL,
    correct_answer  VARCHAR(200) NOT NULL
);

CREATE TABLE IF NOT EXISTS choices (
    choice_id INT AUTO_INCREMENT PRIMARY KEY,
    question_id INT, 
    C1 VARCHAR(20),
    C2 VARCHAR(20),
    C3 VARCHAR(20),
    C4 VARCHAR(20),
    FOREIGN KEY (question_id) REFERENCES questions(question_id)
);

-- Suggestions: 
-- Superadmin and admin

-- Insert initial data into users table
INSERT INTO users (username, password, usertype) VALUES
('user1', 'pw', 'teacher'),
('user2', 'password_hash2', 'teacher'),
('user3', 'pw', 'student'),
('user4', 'password_hash4', 'student'),
('user5', 'password_hash5', 'student');

-- Insert initial data into subjects table
INSERT INTO classrooms (classroom_name, section, teacher_id) VALUES
('Math101', '1', 1),
('Remedial Class 1', '2', 2);

-- Insert initial data into enrollments table
INSERT INTO enrollments (classroom_id, student_id) VALUES 
(1, 1),
(1, 2),
(2, 3);

INSERT INTO questions (minigame_id, question_text, correct_answer) VALUES 
(1, 'What is 1/2 + 1/2 ?', '1'),
(1, 'What is 1/3 + 1/3 ?', '2/3');

INSERT INTO choices (question_id, C1, C2, C3, C4) VALUES 
(1, '1/2', '1/3', '1', '2'),
(2, '1/2', '1/3', '1/4', '2/3');