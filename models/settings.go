package models

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type Setting struct {
	SettingName  string `json:"setting_name"`
	SettingValue string `json:"setting_value"`
	CreateDate   string `json:"create_dt"`
	UpdateDate   string `json:"update_dt"`
}

func GetSettingsSimple() (map[string]string, error) {
	rows, err := DB.Query("SELECT setting_name, setting_value, create_dt, update_dt from settings ORDER BY setting_name")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	settings := make(map[string]string)

	for rows.Next() {
		singleEntry := Setting{}
		err = rows.Scan(&singleEntry.SettingName, &singleEntry.SettingValue, &singleEntry.CreateDate, &singleEntry.UpdateDate)

		if err != nil {
			return nil, err
		}

		settings[singleEntry.SettingName] = singleEntry.SettingValue
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return settings, err
}

// GetSettings returns settings as key value pairs
func GetSettings() (map[string]Setting, error) {
	rows, err := DB.Query("SELECT setting_name, setting_value, create_dt, update_dt from settings ORDER BY setting_name")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	settings := make(map[string]Setting)

	for rows.Next() {
		singleEntry := Setting{}
		err = rows.Scan(&singleEntry.SettingName, &singleEntry.SettingValue, &singleEntry.CreateDate, &singleEntry.UpdateDate)

		if err != nil {
			return nil, err
		}

		settings[singleEntry.SettingName] = singleEntry
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return settings, err
}

func GetSettingByName(settingName string) (Setting, error) {

	stmt, err := DB.Prepare("SELECT setting_name, setting_value, create_dt, update_dt from settings WHERE setting_name = ?")

	if err != nil {
		return Setting{}, err
	}

	setting := Setting{}

	sqlErr := stmt.QueryRow(settingName).Scan(&setting.SettingName, &setting.SettingValue, &setting.CreateDate, &setting.UpdateDate)

	if sqlErr != nil {
		if sqlErr == sql.ErrNoRows {
			return Setting{}, nil
		}
		return Setting{}, sqlErr
	}
	return setting, nil
}

func UpsertSetting(settingName string, settingValue string) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("INSERT INTO settings (setting_name, setting_value, create_dt, update_dt) VALUES (?, ?, current_timestamp, current_timestamp) ON CONFLICT(setting_name) DO UPDATE SET setting_value = ?, update_dt = current_timestamp")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(settingName, settingValue, settingValue)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}

func DeleteSetting(settingName string) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("DELETE FROM settings WHERE setting_name = ?")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(settingName)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}
