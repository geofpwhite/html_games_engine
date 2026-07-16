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