CREATE TABLE IF NOT EXISTS patrulje (
    teamId VARCHAR(99) NOT NULL,
    name VARCHAR(99) NOT NULL DEFAULT "",
    groupName VARCHAR(99) NOT NULL DEFAULT "",
    korps VARCHAR(9) NOT NULL DEFAULT "",
    contactName VARCHAR(99) NOT NULL DEFAULT "",
    contactPhone VARCHAR(99) NOT NULL DEFAULT "",
    contactEmail VARCHAR(99) NOT NULL DEFAULT "",
    contactRole VARCHAR(99) NOT NULL DEFAULT "",
    signupStatus VARCHAR(9) NOT NULL DEFAULT "",
    PRIMARY KEY (teamId)
);
