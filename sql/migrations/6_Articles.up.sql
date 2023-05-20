CREATE TABLE "articles" (
                        "id"                INTEGER,
                        "wordpress_id"      INTEGER,
                        "title"             text,
                        "content"           text,
                        "description"       text,
                        "primary_keyword"   text,
                        "media_id"        INTEGER,
                        "prompt"            text,
                        "yt_url"            text,
                        "img_prompt"    text,
                        "img_search"    text,
                        "img_src_url"    text,
                        "concept"           text,
                        "idea_id"           INTEGER,
                        "status"            text,
                        "version"           INTEGER,
                        "create_dt"         INTEGER,
                        "update_dt"         INTEGER,
                        PRIMARY KEY("id" AUTOINCREMENT)
);

