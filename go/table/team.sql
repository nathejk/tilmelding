CREATE TABLE IF NOT EXISTS patrulje (
    teamId VARCHAR(99) NOT NULL,
    name VARCHAR(99) NOT NULL,
    groupName VARCHAR(99) NOT NULL,
    korps VARCHAR(9) NOT NULL,
    contactName VARCHAR(99) NOT NULL,
    contactPhone VARCHAR(99) NOT NULL,
    contactEmail VARCHAR(99) NOT NULL,
    contactRole VARCHAR(99) NOT NULL,
    PRIMARY KEY (teamId)
);
