package models

import (
	_ "modernc.org/sqlite"
)

type Status struct {
	StatusName  string `json:"status_name"`
	StatusValue string `json:"status_value"`
	CreateDate  string `json:"create_dt"`
	UpdateDate  string `json:"update_dt"`
}

// GetSettings returns settings as key value pairs
func GetStatus() (map[string]Status, error) {
	rows, err := DB.Query("SELECT status_name, status_value, create_dt, update_dt from system_status ORDER BY status_name")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	status := make(map[string]Status)

	for rows.Next() {
		singleEntry := Status{}
		err = rows.Scan(&singleEntry.StatusName, &singleEntry.StatusValue, &singleEntry.CreateDate, &singleEntry.UpdateDate)

		if err != nil {
			return nil, err
		}

		status[singleEntry.StatusName] = singleEntry
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return status, err
}

func UpsertStatus(statusName string, statusValue string) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("INSERT INTO system_status (status_name, status_value, create_dt, update_dt) VALUES (?, ?, current_timestamp, current_timestamp) ON CONFLICT(status_name) DO UPDATE SET status_value = ?, update_dt = current_timestamp")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(statusName, statusValue, statusValue)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}
