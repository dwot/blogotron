package models

import (
	_ "modernc.org/sqlite"
)

type Template struct {
	TemplateName string `json:"template_name"`
	TemplateText string `json:"template_text"`
	CreateDate   string `json:"create_dt"`
	UpdateDate   string `json:"update_dt"`
}

func GetTemplatesSimple() (map[string]string, error) {
	rows, err := DB.Query("SELECT template_name, template_text, create_dt, update_dt from templates ORDER BY template_name")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	templates := make(map[string]string)

	for rows.Next() {
		singleEntry := Template{}
		err = rows.Scan(&singleEntry.TemplateName, &singleEntry.TemplateText, &singleEntry.CreateDate, &singleEntry.UpdateDate)

		if err != nil {
			return nil, err
		}

		templates[singleEntry.TemplateName] = singleEntry.TemplateText
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return templates, err
}

func GetTemplates() (map[string]Template, error) {
	rows, err := DB.Query("SELECT template_name, template_text, create_dt, update_dt from templates ORDER BY template_name")

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	templates := make(map[string]Template)

	for rows.Next() {
		singleEntry := Template{}
		err = rows.Scan(&singleEntry.TemplateName, &singleEntry.TemplateText, &singleEntry.CreateDate, &singleEntry.UpdateDate)

		if err != nil {
			return nil, err
		}

		templates[singleEntry.TemplateName] = singleEntry
	}

	err = rows.Err()

	if err != nil {
		return nil, err
	}

	return templates, err
}

func UpsertTemplate(templateName string, templateText string) (bool, error) {

	tx, err := DB.Begin()
	if err != nil {
		return false, err
	}

	stmt, err := tx.Prepare("INSERT INTO templates (template_name, template_text, create_dt, update_dt) VALUES (?, ?, current_timestamp, current_timestamp) ON CONFLICT(template_name) DO UPDATE SET template_text = ?, update_dt = current_timestamp")

	if err != nil {
		return false, err
	}

	defer stmt.Close()

	_, err = stmt.Exec(templateName, templateText, templateText)

	if err != nil {
		return false, err
	}

	tx.Commit()

	return true, nil
}
