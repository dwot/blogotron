package models

import (
	"database/sql"
	_ "modernc.org/sqlite"
	"strconv"
)

type Series struct {
	Id           int    `json:"id"`
	SeriesName   string `json:"series_name"`
	SeriesPrompt string `json:"series_prompt"`
	CreateDate   string `json:"create_dt"`
	UpdateDate   string `json:"update_dt"`
}

func GetSeries(count int) ([]Series, error) {

	rows, err := DB.Query("SELECT id, series_name, series_prompt, create_dt, update_dt from series LIMIT " + strconv.Itoa(count))

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	series := make([]Series, 0)

	for rows.Next() {
		singleSeries := Series{}
		err = rows.Scan(&singleSeries.Id, &singleSeries.SeriesName, &singleSeries.SeriesPrompt, &singleSeries.CreateDate, &singleSeries.UpdateDate)

		if err != nil {
			return nil, err
		}

		series = append(series, singleSeries)
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return series, err
}

func GetSeriesById(id string) (Series, error) {

	stmt, err := DB.Prepare("SELECT id, series_name, series_prompt, create_dt, update_dt from series WHERE id = ?")

	if err != nil {
		return Series{}, err
	}

	series := Series{}

	sqlErr := stmt.QueryRow(id).Scan(&series.Id, &series.SeriesName, &series.SeriesPrompt, &series.CreateDate, &series.UpdateDate)

	if sqlErr != nil {
		if sqlErr == sql.ErrNoRows {
			return Series{}, nil
		}
		return Series{}, sqlErr
	}
	return series, nil
}

func AddSeriesReturningId(newSeries Series) (int, error) {
	id := 0
	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}

	stmt, err := tx.Prepare("INSERT INTO series (series_name, series_prompt, create_dt, update_dt) VALUES (?, ?, current_timestamp, current_timestamp) RETURNING id")

	if err != nil {
		return 0, err
	}

	defer stmt.Close()

	err = stmt.QueryRow(newSeries.SeriesName, newSeries.SeriesPrompt).Scan(&id)

	if err != nil {
		return 0, err
	}

	tx.Commit()

	return id, nil
}
func AddSeries(newSeries Series) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("INSERT INTO series (series_name, series_prompt, create_dt, update_dt) VALUES (?, ?, current_timestamp, current_timestamp)")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(newSeries.SeriesName, newSeries.SeriesPrompt)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}

func UpdateSeries(ourSeries Series, id int) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("UPDATE series SET series_name = ?, series_prompt = ?, update_dt = current_timestamp WHERE Id = ?")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(ourSeries.SeriesName, ourSeries.SeriesPrompt, ourSeries.Id)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}

func DeleteSeries(seriesId int) (bool, error) {

	tx, err := DB.Begin()

	if err != nil {
		return false, err
	}

	stmt, err := DB.Prepare("DELETE from series where id = ?")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(seriesId)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}
