package models

import (
	_ "modernc.org/sqlite"
)

type Article struct {
	Id             int    `json:"id"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Description    string `json:"description"`
	PrimaryKeyword string `json:"primary_keyword"`
	MediaId        int    `json:"media_id"`
	Prompt         string `json:"prompt"`
	YtUrl          string `json:"yt_url"`
	ImgPrompt      string `json:"img_prompt"`
	ImgSearch      string `json:"img_search"`
	ImgSrcUrl      string `json:"img_src_url"`
	Concept        string `json:"concept"`
	IdeaId         string `json:"idea_id"`
	Status         string `json:"status"`
	Version        int    `json:"version"`
	CreateDate     string `json:"create_dt"`
	UpdateDate     string `json:"update_dt"`
	WordPressId    int    `json:"wordpress_id"`
}

func GetArticles() ([]Article, error) {

	rows, err := DB.Query("SELECT id, wordpress_id, title, content, description, primary_keyword, media_id, prompt, yt_url, img_prompt, " +
		"img_search, img_src_url, concept, idea_id, status, version, create_dt, update_dt from articles ")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	articles := make([]Article, 0)

	for rows.Next() {
		singleEntry := Article{}
		err := rows.Scan(&singleEntry.Id, &singleEntry.WordPressId, &singleEntry.Title, &singleEntry.Content, &singleEntry.Description,
			&singleEntry.PrimaryKeyword, &singleEntry.MediaId, &singleEntry.Prompt, &singleEntry.YtUrl,
			&singleEntry.ImgPrompt, &singleEntry.ImgSearch, &singleEntry.ImgSrcUrl, &singleEntry.Concept,
			&singleEntry.IdeaId, &singleEntry.Status, &singleEntry.Version,
			&singleEntry.CreateDate, &singleEntry.UpdateDate)

		if err != nil {
			return nil, err
		}
		articles = append(articles, singleEntry)
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return articles, err
}

func GetArticleById(id int) (Article, error) {

	row := DB.QueryRow("SELECT id, wordpress_id, title, content, description, primary_keyword, media_id, prompt, yt_url, img_prompt, "+
		"img_search, img_src_url, concept, idea_id, status, version, create_dt, update_dt from articles where id = ?", id)

	singleEntry := Article{}
	err := row.Scan(&singleEntry.Id, &singleEntry.WordPressId, &singleEntry.Title, &singleEntry.Content, &singleEntry.Description,
		&singleEntry.PrimaryKeyword, &singleEntry.MediaId, &singleEntry.Prompt, &singleEntry.YtUrl,
		&singleEntry.ImgPrompt, &singleEntry.ImgSearch, &singleEntry.ImgSrcUrl, &singleEntry.Concept,
		&singleEntry.IdeaId, &singleEntry.Status, &singleEntry.Version,
		&singleEntry.CreateDate, &singleEntry.UpdateDate)

	return singleEntry, err
}

func UpsertArticle(article Article) (int64, error) {

	stmt, err := DB.Prepare("INSERT INTO articles (wordpress_id, title, content, description, primary_keyword, media_id, prompt, yt_url, img_prompt, " +
		"img_search, img_src_url, concept, idea_id, status, version, create_dt, update_dt) " +
		"VALUES (?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, current_timestamp, current_timestamp) " +
		"ON CONFLICT(id) DO UPDATE SET wordpress_id = ?, title = ?, content = ?, description = ?, primary_keyword = ?, media_id = ?, prompt = ?, yt_url = ?, img_prompt = ?, " +
		"img_search = ?, img_src_url = ?, concept = ?, idea_id = ?, status = ?, version = ?, update_dt = current_timestamp")

	if err != nil {
		return -1, err
	}

	res, err := stmt.Exec(article.WordPressId, article.Title, article.Content, article.Description, article.PrimaryKeyword, article.MediaId, article.Prompt, article.YtUrl,
		article.ImgPrompt, article.ImgSearch, article.ImgSrcUrl, article.Concept, article.IdeaId, article.Status, article.Version,
		article.WordPressId, article.Title, article.Content, article.Description, article.PrimaryKeyword, article.MediaId, article.Prompt, article.YtUrl,
		article.ImgPrompt, article.ImgSearch, article.ImgSrcUrl, article.Concept, article.IdeaId, article.Status, article.Version)

	if err != nil {
		return -1, err
	}

	id, err := res.LastInsertId()

	if err != nil {
		return -1, err
	}

	return id, err
}
