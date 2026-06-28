# Build stage
FROM golang:latest AS builder

LABEL maintainer="Geoffrey White <geoffpiercewhite@gmail.com>"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main .

# Final stage
FROM scratch

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/words.db .
COPY --from=builder /app/templates ./templates

EXPOSE 8080

CMD ["/app/main"]