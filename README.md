## Sol'n Teacher Portal

Welcome to the Sol'n Teacher Portal! This web application is a part of the Sol'n educational game project designed to help Grade 6 students at Saint Louis University Basic Education School learn fractions. Teachers use this portal to create and manage interactive fraction-based minigames and quizzes for students. The portal seamlessly connects to the Sol'n student game application, providing a unified experience for both teachers and students.
j
# Features

    Teacher Management: Allows teachers to create, edit, and delete fraction minigames and quizzes.
    Classroom Management: Edit questions from minigames connected to the student module

![Screenshot from 2025-01-10 09-54-56](https://github.com/user-attachments/assets/da853fed-595b-49db-9a2b-c5047256c4b7)
![Screenshot from 2025-01-10 09-55-09](https://github.com/user-attachments/assets/929b3e21-4ccf-4dff-8bb8-171fa202bea8)

    Real-time Student Updates: View live results and performance data from students’ interactions in the Sol'n game.
    
![Screenshot from 2025-01-10 09-55-38](https://github.com/user-attachments/assets/7f9b6442-8055-4a15-9e01-03c3b492b76c)

# What 'game' is the portal connected to, I hear you ask?
Sol'n is a 2D videogame made in the Godot game engine, designed to help elementary students practice operations with fractions.
![fraction-addition](https://github.com/user-attachments/assets/7aeb414d-a19a-4f81-8ec9-437202ee7167)
Teachers may also use the game to implement quizzes and assess student's current knowledge of fractions.
![quiz](https://github.com/user-attachments/assets/94053e66-61c2-40b2-9ad7-57ff1ae65569)


# Technologies Used

    Backend: Go (Golang)
    Frontend: HTMX, daisyUI, Tailwind CSS
    Database: MySQL
    Session Management: Gorilla sessions
    Server Framework: Chi router
    Integration: Connects with the Sol'n student game built in Godot using GDScript.

# Installation

    Clone the Repository:

git clone https://github.com/zenolucas/soln-teachermodule.git
cd soln-teachermodule

Install Dependencies: Ensure Go is installed, then install dependencies using:

    go mod tidy

    Set Up the Database:
        Create a MySQL database for Sol'n.
        Run the SQL script provided in the database directory to set up db user and db tables.

    Configure Environment Variables: create .env file and set up the environment variables ( DB_USER, DB_PASSWORD, DB_NAME, SESSION_SECRET).

Usage

    Run the Server:

    go run main.go

    The portal will be available at http://localhost:3000.

    Teacher Portal Interface:
        Login with your teacher credentials.
        Manage Minigames and Quizzes: Access the dashboard to create fraction-based minigames.
        Track Student Progress: View detailed reports on student performance in quizzes and minigames.

Contributing

Contributions are welcome! Please fork the repository and make a pull request with a clear description of changes.
