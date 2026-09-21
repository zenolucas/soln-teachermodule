FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache make nodejs npm

# net's default DNS resolver uses cgo when CGO_ENABLED=1 (the default), which links the
# binary against libc - incompatible with the scratch base below, which has no libc at
# all. CGO_ENABLED=0 forces a fully static binary.
ENV CGO_ENABLED=0

COPY . ./
RUN make install
RUN make build

FROM scratch
COPY --from=builder /app/bin/soln-teachermodule /soln-teachermodule

EXPOSE 3000
ENTRYPOINT [ "/soln-teachermodule" ]

# No .env is baked into the image - InitializeDatabase() treats a missing
# .env as fine and reads config straight from the real environment, which is
# how a container should be configured. Provide at minimum (`docker run -e`,
# an --env-file, or your orchestrator's secret/env mechanism):
#   DBUSER, DBPASS, DBNAME       - required, no defaults
#   DBHOST, DBPORT               - default 127.0.0.1:3306, which is wrong
#                                   inside a container; point this at your
#                                   actual MySQL/MariaDB host, e.g. a sibling
#                                   "db" service on the same Docker network
#   SESSION_SECRET                - required, >= 32 characters
#   GAME_TOKEN_SECRET              - required
#   HTTP_LISTEN_ADDRESS            - e.g. ":3000"
