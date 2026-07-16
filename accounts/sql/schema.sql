CREATE TABLE Users (
    UserID SERIAL PRIMARY KEY,
    Username TEXT NOT NULL,
    Password TEXT NOT NULL,
    CreatedAt DATE DEFAULT CURRENT_DATE,
    unique (Username)
);

CREATE INDEX idx_users_username on Users (Username);
