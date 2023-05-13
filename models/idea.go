package models

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"strconv"
)

type Idea struct {
	Id          int    `json:"id"`
	IdeaText    string `json:"idea_text"`
	Status      string `json:"status"`
	IdeaConcept string `json:"idea_concept"`
	SeriesId    int    `json:"series_id"`
	CreateDate  string `json:"create_dt"`
	UpdateDate  string `json:"update_dt"`
}

func GetIdeas(count int) ([]Idea, error) {

	rows, err := DB.Query("SELECT id, idea_text, status, idea_concept, series_id, create_dt, update_dt from idea LIMIT " + strconv.Itoa(count))

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	idea := make([]Idea, 0)

	for rows.Next() {
		singleIdea := Idea{}
		err = rows.Scan(&singleIdea.Id, &singleIdea.IdeaText, &singleIdea.Status, &singleIdea.IdeaConcept, &singleIdea.SeriesId, &singleIdea.CreateDate, &singleIdea.UpdateDate)

		if err != nil {
			return nil, err
		}

		idea = append(idea, singleIdea)
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return idea, err
}

func GetOpenIdeas() ([]Idea, error) {

	rows, err := DB.Query("SELECT id, idea_text, status, idea_concept, series_id, create_dt, update_dt from idea WHERE status = 'NEW' and series_id = 0")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	idea := make([]Idea, 0)

	for rows.Next() {
		singleIdea := Idea{}
		err = rows.Scan(&singleIdea.Id, &singleIdea.IdeaText, &singleIdea.Status, &singleIdea.IdeaConcept, &singleIdea.SeriesId, &singleIdea.CreateDate, &singleIdea.UpdateDate)

		if err != nil {
			return nil, err
		}

		idea = append(idea, singleIdea)
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return idea, err
}

func GetOpenSeriesIdeas(id string) ([]Idea, error) {
	stmt, err := DB.Prepare("SELECT id, idea_text, status, idea_concept, series_id, create_dt, update_dt from idea WHERE status = 'NEW' and series_id = ?")
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	idea := make([]Idea, 0)

	for rows.Next() {
		singleIdea := Idea{}
		err = rows.Scan(&singleIdea.Id, &singleIdea.IdeaText, &singleIdea.Status, &singleIdea.IdeaConcept, &singleIdea.SeriesId, &singleIdea.CreateDate, &singleIdea.UpdateDate)

		if err != nil {
			return nil, err
		}

		idea = append(idea, singleIdea)
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return idea, err
}

func GetIdeaById(id string) (Idea, error) {

	stmt, err := DB.Prepare("SELECT id, idea_text, status, idea_concept, series_id, create_dt, update_dt from idea WHERE id = ?")

	if err != nil {
		return Idea{}, err
	}

	idea := Idea{}

	sqlErr := stmt.QueryRow(id).Scan(&idea.Id, &idea.IdeaText, &idea.Status, &idea.IdeaConcept, &idea.SeriesId, &idea.CreateDate, &idea.UpdateDate)

	if sqlErr != nil {
		if sqlErr == sql.ErrNoRows {
			return Idea{}, nil
		}
		return Idea{}, sqlErr
	}
	return idea, nil
}

func AddIdea(newIdea Idea) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("INSERT INTO idea (idea_text, status, idea_concept, series_id, create_dt, update_dt) VALUES (?,?,?,?, current_timestamp, current_timestamp)")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(newIdea.IdeaText, newIdea.Status, newIdea.IdeaConcept, newIdea.SeriesId)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}

func SetIdeaWritten(ideaId int) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("UPDATE idea SET status = 'WRITTEN', update_dt = current_timestamp WHERE Id = ?")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(ideaId)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}

func UpdateIdea(ourIdea Idea, id int) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("UPDATE idea SET idea_text = ?, status = ?, idea_concept = ?, series_id = ?, update_dt = current_timestamp WHERE Id = ?")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(ourIdea.IdeaText, ourIdea.Status, ourIdea.IdeaConcept, ourIdea.SeriesId, ourIdea.Id)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}

func DeleteIdea(ideaId int) (bool, error) {

	tx, err := DB.Begin()

	if err != nil {
		return false, err
	}

	stmt, err := DB.Prepare("DELETE from idea where id = ?")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(ideaId)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}
