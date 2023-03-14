CREATE TABLE IF NOT EXISTS mailTemplate (
    slug VARCHAR(99) NOT NULL,
    subject VARCHAR(99) NOT NULL,
    template TEXT NOT NULL,
    PRIMARY KEY (slug)
);
