CREATE TABLE "series" (
                        "id"                INTEGER,
                        "name"              text,
                        "template"          text,
                        "create_dt"         INTEGER,
                        "update_dt"         INTEGER,
                        PRIMARY KEY("id" AUTOINCREMENT)
)

CREATE TABLE "story" (
                         "id"        INTEGER,
                         "prompt"    INTEGER,
                         "media"       TEXT,
                         "media_type"  TEXT,
                         "wp_media_id"  INTEGER,
                         "wp_story_id"  INTEGER,
                         "wp_url"   TEXT,
                         "title"     TEXT,
                         "content"   TEXT,
                         "tags"      TEXT,
                         PRIMARY KEY("id" AUTOINCREMENT)
)
