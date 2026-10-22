run: build
	@./bin/soln-teachermodule

install:
	@go install github.com/a-h/templ/cmd/templ@v0.3.819
	@go get ./...
	@go mod vendor
	@go mod tidy
	@go mod download
	@# Install exactly what package-lock.json pins: daisyui@latest is v5, which needs
	@# Tailwind 4 and breaks this Tailwind 3 setup (no theme colours, so the CSS build fails).
	@npm ci

css:
	@npx tailwindcss -i view/css/app.css -o public/styles.css --watch

css-build:
	@npx tailwindcss -i view/css/app.css -o public/styles.css --minify

templ:
	@templ generate --watch --proxy=http://localhost:3000

build: css-build
	@templ generate view
	@go build -tags dev -o bin/soln-teachermodule .

# The static showcase (export.go): every read-only page, pre-rendered into dist/ from the
# showcase data. Needs a MariaDB root DSN, e.g.
#   make demo SOLN_EXPORT_DB_ROOT_DSN='root@tcp(127.0.0.1:3306)/'
# DEMO_BOOKING_URL and DEMO_DOWNLOAD_URL are read from the environment too.
demo: css-build
	@templ generate view
	@go run . export-demo -out dist
