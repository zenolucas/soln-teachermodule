-- The Sol'n schema: tables and indexes only, no data. Teachers sign up in the portal and students
-- register in the game. For demo data (one teacher, three classes, a few weeks of play), load
-- testdata/showcase_seed.sql on top; see the README's "Demo data".
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
    password VARCHAR(255) NOT NULL,
    -- Incremented on logout; a session cookie carrying an older value is rejected (server-side logout).
    session_version INT NOT NULL DEFAULT 0
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

-- One JSON document per student (SAVE-01): the game owns the save's shape, and loading merges it over
-- database.DefaultSave, which holds the defaults. No row until the student first saves.
CREATE TABLE IF NOT EXISTS save_states (
    save_id INT AUTO_INCREMENT PRIMARY KEY,
    student_id INT NOT NULL UNIQUE,
    save_data JSON NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (student_id) REFERENCES users(user_id)
);


CREATE TABLE IF NOT EXISTS fraction_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT NOT NULL,
    minigame_id INT NOT NULL,
    question_text VARCHAR(500),
    fraction1_numerator INT NOT NULL,
    fraction1_denominator INT NOT NULL,
    fraction2_numerator INT NOT NULL,
    fraction2_denominator INT NOT NULL,
    FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id)
);

CREATE TABLE IF NOT EXISTS fraction_responses (
    statistic_id INT AUTO_INCREMENT PRIMARY KEY,
    classroom_id INT NOT NULL,
    minigame_id INT NOT NULL,
    question_id INT NOT NULL,
    student_id INT NOT NULL,
    num_right_attempts INT DEFAULT 0,
    num_wrong_attempts INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
    FOREIGN KEY (question_id) REFERENCES fraction_questions(question_id),
    FOREIGN KEY (student_id) REFERENCES users(user_id)
);

CREATE TABLE IF NOT EXISTS multiple_choice_questions (
    question_id INT AUTO_INCREMENT PRIMARY KEY, 
    classroom_id INT,
    minigame_id INT,
    question_text VARCHAR(500) NOT NULL
);

-- Table to store choices
CREATE TABLE IF NOT EXISTS multiple_choice_choices (
    choice_id INT AUTO_INCREMENT PRIMARY KEY,
    question_id INT, 
    choice_text VARCHAR(255) NOT NULL,
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
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (classroom_id) REFERENCES classrooms(classroom_id),
  FOREIGN KEY (student_id) REFERENCES users(user_id)
);

-- Every read path filters on (minigame_id, classroom_id) or (student_id, minigame_id),
-- and none of those columns were indexed - only the auto-increment PKs and FK columns
-- were. Small tables today, but these are the queries that run on every page load and
-- every game event.
CREATE INDEX IF NOT EXISTS idx_fq_minigame_classroom  ON fraction_questions (minigame_id, classroom_id);
CREATE INDEX IF NOT EXISTS idx_fr_lookup              ON fraction_responses (classroom_id, minigame_id, question_id);
CREATE INDEX IF NOT EXISTS idx_fr_student             ON fraction_responses (student_id, minigame_id);
CREATE INDEX IF NOT EXISTS idx_fr_created             ON fraction_responses (classroom_id, created_at);
CREATE INDEX IF NOT EXISTS idx_mcq_minigame_classroom ON multiple_choice_questions (minigame_id, classroom_id);
CREATE INDEX IF NOT EXISTS idx_mcr_lookup             ON multiple_choice_responses (classroom_id, minigame_id, question_id);
CREATE INDEX IF NOT EXISTS idx_mcr_student            ON multiple_choice_responses (student_id, minigame_id);
CREATE INDEX IF NOT EXISTS idx_mcs_lookup             ON multiple_choice_scores (classroom_id, minigame_id);
CREATE INDEX IF NOT EXISTS idx_mcs_created            ON multiple_choice_scores (classroom_id, created_at);
CREATE INDEX IF NOT EXISTS idx_users_type             ON users (usertype);
CREATE INDEX IF NOT EXISTS idx_save_states_student    ON save_states (student_id);
