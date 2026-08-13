-- name: CreateUser :one
INSERT INTO
    Users (Username, Password)
VALUES
    ($1, $2)
RETURNING
    UserID;

-- name: GetUser :one
SELECT
    UserID,
    Username,
    Password
FROM
    Users
WHERE
    UserID = $1;

-- name: GetUserByUsername :one
SELECT
    UserID,
    Username,
    Password
FROM
    Users
WHERE
    Username = $1;

-- name: UpdateUserPassword :exec
UPDATE Users
SET
    Password = $2
WHERE
    UserID = $1;

-- name: DeleteUser :exec
DELETE FROM Users
WHERE
    UserID = $1;

-- name: GetUsernames :many
SELECT
    Username
FROM
    Users;

-- name: StartGameSession :one
INSERT INTO
    GameSessions (UserID, GameType)
VALUES
    ($1, $2)
RETURNING
    GameSessionID;

-- name: EndGameSession :exec
UPDATE GameSessions
SET
    EndTime = now()
WHERE
    GameSessionID = $1;

-- name: GetUserStats :one
SELECT
    COUNT(*) AS GamesPlayed,
    COALESCE(SUM(EXTRACT(EPOCH FROM (EndTime - StartTime))), 0)::float8 AS TotalSeconds
FROM
    GameSessions
WHERE
    UserID = $1;