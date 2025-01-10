## Sol'n Teacher Portal

Welcome to the Sol'n Teacher Portal! This web application is a part of the Sol'n educational game project designed to help Grade 6 students at Saint Louis University Basic Education School learn fractions. Teachers use this portal to create and manage interactive fraction-based minigames and quizzes for students. The portal seamlessly connects to the Sol'n student game application, providing a unified experience for both teachers and students.
j
# Features

    Teacher Management: Allows teachers to create, edit, and delete fraction minigames and quizzes.
    Classroom Management: Assign specific quizzes to students and track student progress.

![Screenshot from 2025-01-10 09-54-56](https://github.com/user-attachments/assets/da853fed-595b-49db-9a2b-c5047256c4b7)
![Screenshot from 2025-01-10 09-55-09](https://github.com/user-attachments/assets/929b3e21-4ccf-4dff-8bb8-171fa202bea8)

    
    User Authentication: Secure login system with session-based authentication.
    Real-time Student Updates: View live results and performance data from students’ interactions in the Sol'n game.
    
![Screenshot from 2025-01-10 09-55-38](https://github.com/user-attachments/assets/7f9b6442-8055-4a15-9e01-03c3b492b76c)

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
        Run the SQL script provided in the database directory to set up tables, including users for managing teachers and students.

    Configure Environment Variables: Copy .env.example to .env and set up the environment variables (e.g., DB_USER, DB_PASSWORD, DB_NAME, SESSION_SECRET).

Session and Token Management: Configure session settings in .env for session storage and expiration:

    SESSION_SECRET=your_secret_key

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
