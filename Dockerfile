# Postgres stage - seeds the accounts schema on first boot
FROM postgres:16 AS pgsql
ENV POSTGRES_USER=app
ENV POSTGRES_DB=accounts
ADD accounts/sql/schema.sql /docker-entrypoint-initdb.d/

# Build stage
FROM golang:latest AS builder

LABEL maintainer="Geoffrey White <geoffpiercewhite@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Final stage
FROM scratch AS final

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/words.db .
COPY --from=builder /app/templates ./templates

ENV DATABASE_URL=postgresql://app:app@db:5432/accounts?sslmode=disable
ENV REDIS_ADDR=redis:6379

EXPOSE 8080

CMD ["/app/main"]