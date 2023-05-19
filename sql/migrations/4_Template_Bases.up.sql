INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('system-prompt', 'You are blog writer and you want to write a new article.', current_timestamp, current_timestamp);
INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('article-prompt', 'Write an article about {{.Prompt}}.  The article should be {{.Length}} words long and it should use HTML headings and subheadings to organize the article.  Include the primary keyword (which is {{.Keyword}}) in the title, the first paragraph, and a couple of times throughout the text, as naturally as possible.  Include a good title wrapped in an h1 tag.', current_timestamp, current_timestamp);
INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('title-prompt', 'Find a title for your article. Use the language of the article.', current_timestamp, current_timestamp);
INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('description-prompt', 'Write a concise and captivating meta description for the article that includes the primary keyword {{.Keyword}}. Return the description alone, no other text or markup.', current_timestamp, current_timestamp);
INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('imggen-prompt', 'Come up with a good prompt to give to Dall-E to generate an image to accompany your article.', current_timestamp, current_timestamp);
INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('img-prompt', '{{.ImagePrompt}}', current_timestamp, current_timestamp);
INSERT into templates (template_name, template_text, create_dt, update_dt) VALUES ('imgsearch-prompt', 'Come up with a simple search term to find an image on Unsplash to use in your article.', current_timestamp, current_timestamp);