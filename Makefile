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
	@go build -tags dev -o bin/soln-teachermodule main.go 
