CREATE TABLE Users (
    UserID SERIAL PRIMARY KEY,
    Username TEXT NOT NULL,
    Password TEXT NOT NULL,
    CreatedAt DATE DEFAULT CURRENT_DATE,
    unique (Username)
);

CREATE INDEX idx_users_username on Users (Username);

CREATE TYPE GameType AS ENUM (
    'connect4',
    'connectthedots',
    'tictactoe',
    'hangman',
    'whiteboard'
);

CREATE TABLE GameSessions (
    GameSessionID SERIAL PRIMARY KEY,
    UserID INT NOT NULL REFERENCES Users (UserID),
    GameType GameType NOT NULL,
    StartTime TIMESTAMPTZ NOT NULL DEFAULT now(),
    EndTime TIMESTAMPTZ
);

CREATE INDEX idx_game_sessions_userid ON GameSessions (UserID);
