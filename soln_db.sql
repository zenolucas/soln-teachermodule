-- Create the database
CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

-- Create the users table
CREATE TABLE IF NOT EXISTS users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(255) NOT NULL,
    usertype ENUM('student', 'teacher') NOT NULL
);

-- Create the subjects table
CREATE TABLE IF NOT EXISTS subjects (
    subject_id INT AUTO_INCREMENT PRIMARY KEY,
    subject_name VARCHAR(100) NOT NULL,
    teacher_id INT,
    FOREIGN KEY (teacher_id) REFERENCES users(user_id)
);


-- Create the enrollments table (for which students are assigned to each subject)
CREATE TABLE IF NOT EXISTS enrollments (
    enrollment_id INT AUTO_INCREMENT PRIMARY KEY,
    subject_id INT,
    student_id INT,
    FOREIGN KEY (subject_id) REFERENCES subjects(subject_id),
    FOREIGN KEY (student_id) REFERENCES users(user_id),
    UNIQUE KEY unique_enrollment (subject_id, student_id)
);


-- Create the MiniGames table
CREATE TABLE IF NOT EXISTS minigames (
    minigame_id INT AUTO_INCREMENT PRIMARY KEY,
    minigame_name   VARCHAR(100) NOT NULL,
    minigame_type   ENUM('quest', 'miniboss') NOT NULL
);

-- Create the Levels table
CREATE TABLE IF NOT EXISTS levels (
    level_id INT AUTO_INCREMENT PRIMARY KEY,
    level_name VARCHAR(100)
);

-- Create the LevelMiniGameConfig
CREATE TABLE IF NOT EXISTS levelgames (
    level_id INT, 
    minigame_id INT, 
    FOREIGN KEY (level_id) REFERENCES levels(level_id),
    FOREIGN KEY (minigame_id) REFERENCES minigames(minigame_id)
);

-- Suggestions: 
-- Superadmin and admin


-- Insert initial data into users table
INSERT INTO users (username, password, usertype) VALUES
('user1', 'pw', 'teacher'),
('user2', 'password_hash2', 'teacher'),
('user3', 'password_hash3', 'student'),
('user4', 'password_hash4', 'student'),
('user5', 'password_hash5', 'student');

-- Insert initial data into subjects table
INSERT INTO subjects (subject_name, teacher_id) VALUES
('Math101', 1),
('G6-Mathematics', 2);

-- Insert initial data into enrollments table
INSERT INTO enrollments (subject_id, student_id) VALUES 
(1, 1),
(1, 2),
(2, 3);