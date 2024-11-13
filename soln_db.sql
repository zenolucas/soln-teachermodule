CREATE DATABASE IF NOT EXISTS soln_db;
USE soln_db;

CREATE TABLE IF NOT EXISTS users (
    user_id INT AUTO_INCREMENT PRIMARY KEY,
    firstname VARCHAR(50),
    lastname VARCHAR(50),
    username VARCHAR(50) NOT NULL UNIQUE,
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

CREATE TABLE IF NOT EXISTS save_states (
    save_id INT AUTO_INCREMENT PRIMARY KEY,
    student_id INT, 
    current_floor INT,
    current_quest VARCHAR(100),
    saved_scene VARCHAR(100),
    vector_x FLOAT,
    vector_y FLOAT,
    badge_rock BOOLEAN DEFAULT FALSE,
    badge_bowl BOOLEAN DEFAULT FALSE,
    badge_carrot BOOLEAN DEFAULT FALSE,
    badge_cake BOOLEAN DEFAULT FALSE,
    badge_sword BOOLEAN DEFAULT FALSE,
    badge_mushroom BOOLEAN DEFAULT FALSE,
    badge_bucket1 BOOLEAN DEFAULT FALSE,
    badge_flask BOOLEAN DEFAULT FALSE,
    badge_bucket2 BOOLEAN DEFAULT FALSE,
    badge_bucket3 BOOLEAN DEFAULT FALSE,
    badge_crystal_ball BOOLEAN DEFAULT FALSE,
    badge_original_robot BOOLEAN DEFAULT FALSE,
    first_time_init_floor1 BOOLEAN DEFAULT FALSE,
    first_time_init_floor2 BOOLEAN DEFAULT FALSE,
    first_time_init_floor3 BOOLEAN DEFAULT FALSE,
    disable_dead_robot_quest BOOLEAN DEFAULT FALSE,
    disable_raket_stealing_quest BOOLEAN DEFAULT FALSE,
    disable_fresh_dialogue_quest BOOLEAN DEFAULT FALSE,
    disable_water_logged_1_quest BOOLEAN DEFAULT FALSE,
    disable_water_logged_2_quest BOOLEAN DEFAULT FALSE,
    disable_water_logged_3_quest BOOLEAN DEFAULT FALSE,
    disable_chip_quest BOOLEAN DEFAULT FALSE,
    disable_rat_wizard_training_quest BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (student_id) REFERENCES users(user_id)
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
    num_right_attempts INT DEFAULT 0,
    num_wrong_attempts INT DEFAULT 0
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
('teacher1', 'teacher', 'password'),
('user2', 'teacher', 'pw');

-- insert student data
-- insert student data
INSERT INTO users (username, firstname, lastname, usertype, class_number, section, password) VALUES
('user3', 'John', 'Johnson', 'student', '1', '1', 'pw'),
('user4', 'Mike', 'Tyson', 'student', '2', '1', 'pw'),
('user5', 'Joe', 'Seph', 'student', '3', '1', 'pw'),
('user6', 'Emily', 'Davis', 'student', '4', '1', 'pw'),
('user7', 'Sarah', 'Smith', 'student', '5', '1', 'pw'),
('user8', 'David', 'Lee', 'student', '6', '1', 'pw'),
('user9', 'Anna', 'Brown', 'student', '7', '1', 'pw'),
('user10', 'Chris', 'Wilson', 'student', '8', '1', 'pw'),
('user11', 'Sophia', 'Martinez', 'student', '9', '1', 'pw'),
('user12', 'James', 'Garcia', 'student', '10', '1', 'pw'),
('user13', 'Liam', 'Rodriguez', 'student', '11', '1', 'pw'),
('user14', 'Olivia', 'Hernandez', 'student', '12', '1', 'pw'),
('user15', 'Jackson', 'Lopez', 'student', '13', '1', 'pw'),
('user16', 'Mia', 'Gonzalez', 'student', '14', '1', 'pw'),
('user17', 'Ethan', 'Perez', 'student', '15', '1', 'pw'),
('user18', 'Isabella', 'Martinez', 'student', '16', '1', 'pw'),
('user19', 'Benjamin', 'Taylor', 'student', '17', '1', 'pw'),
('user20', 'Charlotte', 'Anderson', 'student', '18', '1', 'pw'),
('user21', 'Lucas', 'Thomas', 'student', '19', '1', 'pw'),
('user22', 'Amelia', 'Jackson', 'student', '20', '1', 'pw');



-- Insert initial data into classrooms table
INSERT INTO classrooms (classroom_name, section, description, teacher_id) VALUES
('Classroom 1', '1', 'Lorem Ipsum', 1),
('Classroom 2', '2', 'Lorem Ipsum', 1),
('Classroom 3', '3', 'Lorem Ipsum', 2);

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

-- insert sample data for minigames 6, 7, 8, 9 
INSERT INTO fraction_questions (minigame_id, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES
(6, 2, 4, 1, 4),
(6, 3, 4, 2, 4),
(6, 4, 5, 3, 5),
(7, 2, 3, 1, 3),
(7, 3, 3, 2, 3), 
(7, 2, 7, 1, 7),
(8, 2, 4, 1, 4),
(8, 2, 3, 1, 3), 
(8, 2, 4, 1, 4),
(9, 4, 9, 3, 9),
(9, 4, 5, 3, 5), 
(9, 5, 7, 4, 7);

-- insert data for worded fraction questions
INSERT INTO fraction_questions (minigame_id, question_text, fraction1_numerator, fraction1_denominator, fraction2_numerator, fraction2_denominator) VALUES
(3, "what is 2/4 + 2/4?", 2, 4, 2, 4),
(3, "what is 7/10 + 1/5?", 7, 10, 1, 5),
(3, "what is 9/5 + 4/5", 9, 5, 4, 5),
(4, "what is 3/4 + 3/4?", 3, 4, 3, 4),
(4, "what is 8/10 + 2/5?", 8, 10, 2, 5),
(4, "what is 8/5 + 3/5", 8, 5, 3, 5),
(10, "what is 6/10 - 3/10?", 6, 10, 3, 10),
(10, "what is 8/10 - 2/10?", 8, 10, 2, 10),
(10, "what is 3/5 - 1/5", 3, 5, 1, 5);

-- sample data for quiz, minigame 5
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

-- sample data for quiz, minigame 11 
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


-- sample data for quiz, minigame 12
INSERT INTO multiple_choice_questions (minigame_id, question_text) VALUES
(12, 'What is 1/2 + 1/4?'),
(12, 'What is 2/3 - 1/3?'),
(12, 'What is 1/4 + 1/4?'),
(12, 'What is 2/3 + 1/6?'),
(12, 'What is 5/8 - 1/8?'),
(12, 'What is 2/2 - 1/2?'),
(12, 'What is 1/3 + 1/3?'),
(12, 'What is 3/4 - 1/4?'),
(12, 'What is 1/2 + 1/4?'),
(12, 'What is 3/4 - 1/2?');

INSERT INTO multiple_choice_choices (question_id, choice_text, is_correct) VALUES
(21, '1/2', FALSE),
(21, '3/4', TRUE),
(21, '1/4', FALSE),
(21, '1', FALSE),

(22, '1/3', TRUE),
(22, '2/3', FALSE),
(22, '1/2', FALSE),
(22, '1', FALSE),

(23, '1/2', TRUE),
(23, '1/4', FALSE),
(23, '1', FALSE),
(23, '3/4', FALSE),

(24, '1/6', FALSE),
(24, '5/6', TRUE),
(24, '1/2', FALSE),
(24, '2/3', FALSE),

(25, '1/2', TRUE),
(25, '5/8', FALSE),
(25, '3/8', FALSE),
(25, '1/4', FALSE),

(26, '1/2', TRUE),
(26, '1', FALSE),
(26, '1/4', FALSE),
(26, '3/4', FALSE),

(27, '2/3', TRUE),
(27, '1/3', FALSE),
(27, '1/2', FALSE),
(27, '1', FALSE),

(28, '1/2', TRUE),
(28, '3/4', FALSE),
(28, '1/4', FALSE),
(28, '1', FALSE),

(29, '3/4', TRUE),
(29, '1/2', FALSE),
(29, '1/3', FALSE),
(29, '1/4', FALSE),

(30, '1/4', TRUE),
(30, '1/2', FALSE),
(30, '1/3', FALSE),
(30, '2/4', FALSE);

-- test values for statistics quiz, minigame id 5
INSERT INTO multiple_choice_scores (student_id, classroom_id, minigame_id, score) VALUES
(3, 1, 5, 7),
(4, 1, 5, 5),
(5, 1, 5, 8),
(6, 1, 5, 6),
(7, 1, 5, 9),
(8, 1, 5, 4),
(9, 1, 5, 3),
(10, 1, 5, 7),
(11, 1, 5, 10),
(12, 1, 5, 6),
(13, 1, 5, 8),
(14, 1, 5, 5),
(15, 1, 5, 7),
(16, 1, 5, 6),
(17, 1, 5, 4),
(18, 1, 5, 9),
(19, 1, 5, 2),
(20, 1, 5, 8),
(21, 1, 5, 6),
(22, 1, 5, 10);


-- sample response values for simple fraction questions with minigame ID 1
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 1, 1, 3, 1, 3),
(1, 1, 2, 3, 1, 1),
(1, 1, 3, 3, 1, 2);

-- sample response values for simple fraction questions with minigame ID 2
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 2, 4, 3, 1, 1),
(1, 2, 5, 3, 1, 1),
(1, 2, 6, 3, 1, 3);

-- sample response values for worded fraction questions with minigame ID 3
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 3, 7, 3, 1, 2),
(1, 3, 8, 3, 1, 1),
(1, 3, 9, 3, 1, 2);

-- sample response values for worded fraction questions with minigame ID 4
INSERT INTO fraction_responses (classroom_id, minigame_id, question_id, student_id, num_right_attempts, num_wrong_attempts) VALUES 
(1, 4, 10, 3, 1, 1),
(1, 4, 11, 3, 1, 3),
(1, 4, 12, 3, 1, 2);


-- SAMPLE VALUE FOR SAVED STATES
INSERT INTO save_states (
    student_id, current_floor, current_quest, saved_scene, vector_x, vector_y,
    badge_rock, badge_bowl, badge_carrot, badge_cake
) VALUES (
    3, 1, 'share_pie_with_racket', 'res://scenes/levels/Floor1.tscn', 1232.74, 1043.073,
    TRUE, TRUE, TRUE, TRUE
);

