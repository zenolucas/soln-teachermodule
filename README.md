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

    Configure Environment Variables: create .env file and set up the environment variables (HTTP_LISTEN_ADDRESS, DB_USER, DB_PASSWORD, DB_NAME, SESSION_SECRET).

Usage

    Run the Server:

    go run main.go

    The portal will be available at http://localhost: [ insert port number / HTTP_LISTEN_ADDRESS ]

Demo data

    `soln_db.sql` is the schema only. `testdata/showcase_seed.sql` fills it with semi-realistic
    demo data for showing the portal off:
    - one teacher, **teacher / teacher** (Clarissa Reyes), with three Grade 6 sections (84 students);
    - a question set for every scene, including word problems and quiz distractors that model common
      fraction mistakes;
    - a few weeks of simulated play that follows the game's rules (energy, game overs, quiz retakes),
      so progress, flags, "Got stuck", quiz statistics and misconception hints all have something to show.
    Every student's password is `student` (e.g. log in to the game as one of them). Timestamps are
    relative to the day the file is loaded, so recent activity always looks current.

    Load it into the dev database with:

    cd ~/.local/share/soln-devtools/setupdb && go run . --reset --demo

    The file is generated. To change the data, edit `testdata/showcase/main.go` and run
    `go run ./testdata/showcase > testdata/showcase_seed.sql`.

Migrations

    `soln_db.sql` is the current schema; a fresh load already includes every migration.
    `migrations/` holds incremental ALTERs for databases that were already set up from
    an older `soln_db.sql`. Apply one against an existing database with:

    mariadb -u$DB_USER -p$DB_PASSWORD $DB_NAME < migrations/001_created_at.sql
    mariadb -u$DB_USER -p$DB_PASSWORD $DB_NAME < migrations/002_save_states_json.sql
    mariadb -u$DB_USER -p$DB_PASSWORD $DB_NAME < migrations/003_session_version.sql

Deploying with HTTPS

    `deploy/` runs the portal on the internet behind Caddy, which gets and renews a
    Let's Encrypt certificate automatically. You need a server with Docker, a domain
    whose DNS points at it, and ports 80 and 443 open.

    cp deploy/.env.example deploy/.env      # set DOMAIN, the DB passwords and both secrets
    docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d --build

    Only Caddy is exposed; the app and MariaDB stay on Docker's internal network, and
    the session cookie is marked Secure (COOKIE_SECURE=true). The schema is loaded on the
    first start, together with the showcase data (see "Demo data"), so visitors can log in
    as teacher / teacher. Those passwords are public: that's the point of a demo, but for a
    real school deployment remove the `02-showcase.sql` line from deploy/docker-compose.yml
    before the first start.

    In the game, students type the full address (https://your-domain) into the server
    box instead of an IP. A bare IP still means plain HTTP on port 3000, for a school LAN.

    To try the stack locally first, set DOMAIN=localhost: Caddy then uses its own
    certificate authority, so the browser will show a warning.

Contributing

Contributions are welcome! Please fork the repository and make a pull request with a clear description of changes.
