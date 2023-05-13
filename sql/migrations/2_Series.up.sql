CREATE TABLE "series" (
                          "id"                INTEGER,
                          "series_name"         text,
                          "series_prompt"       text,
                          "create_dt"         INTEGER,
                          "update_dt"         INTEGER,
                          PRIMARY KEY("id" AUTOINCREMENT)
);

ALTER TABLE "idea"
    ADD COLUMN idea_concept TEXT;

ALTER TABLE "idea"
    ADD COLUMN series_id INTEGER;

