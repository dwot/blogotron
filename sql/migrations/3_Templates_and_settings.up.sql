CREATE TABLE "templates" (
                          "template_name"         text,
                          "template_text"       text,
                          "create_dt"         INTEGER,
                          "update_dt"         INTEGER,
                          PRIMARY KEY("template_name")
);

CREATE TABLE "settings" (
                        "setting_name"         text,
                        "setting_value"       text,
                        "create_dt"         INTEGER,
                        "update_dt"         INTEGER,
                        PRIMARY KEY ("setting_name")
);

INSERT INTO "settings" VALUES ('BLOGOTRON_PORT','8666',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('BLOGOTRON_API_PORT','8667',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('ENABLE_GPT4','false',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('OPENAI_API_KEY','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('WP_URL','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('WP_USERNAME','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('WP_PASSWORD','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('SD_URL','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('IMG_MODE','none',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('IMG_WIDTH','800',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('IMG_HEIGHT','450',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('IMG_STEPS','30',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('IMG_SAMPLER','DPM++ 2M',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('IMG_NEGATIVE_PROMPTS','watermark,border,blurry,duplicate',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('UNSPLASH_ACCESS_KEY','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('UNSPLASH_SECRET_KEY','',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('LOW_IDEA_THRESHOLD','0',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('AUTO_POST_ENABLE','false',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('AUTO_POST_INTERVAL','10m',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('AUTO_POST_IMG_ENGINE','generate',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('AUTO_POST_LEN','1250',current_timestamp, current_timestamp);
INSERT INTO "settings" VALUES ('AUTO_POST_STATE','publish',current_timestamp, current_timestamp);

