package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

func createTableIfNotExists(db *sql.DB, tableName string, schema string) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ` + tableName + ` 
		(` + schema + `)
	`)

	if err != nil {
		log.Fatalf("Error creating table %s: %s", tableName, err)
	}
}

func querySetting(db *sql.DB, settingKey string) (string, error) {
	var settingValue string
	err = db.QueryRow("SELECT value FROM settings WHERE key = $1", settingKey).Scan(&settingValue)
	if err != nil {
		return "", err
	}
	return settingValue, nil
}

func getLastUpdateTime(db *sql.DB) (*time.Time, error) {
	lastUpdate, err := querySetting(db, "last_update")
	if err != nil {
		return nil, err
	}
	lastUpdateTime, err := time.Parse(time.RFC3339, lastUpdate)
	return &lastUpdateTime, err
}

func updateSettingValue(db *sql.DB, key, value string) error {
	_, err := querySetting(db, key)
	if err == sql.ErrNoRows {
		_, err1 := db.Exec("INSERT INTO settings (key, value) VALUES ($1, $2)", key, value)
		if err1 != nil {
			return err1
		}
		return nil
	}
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE settings
		SET value = $1
		WHERE key = $2`, 
		value,
		key,
	)
	return err
}

func updateLastUpdateTime(db *sql.DB, t string) error {
	return updateSettingValue(db, "last_update", t)
}

func queryThread(id int64, db *sql.DB) (*NotificationThread, error) {
	var thread NotificationThread
	query := `SELECT ID, Type, Repo, Title, Status, Pinned, Author, IsReviewRequest, URL, UpdatedAt, Unread, Notifications FROM threads WHERE ID = ?`
	row := db.QueryRow(query, id)
	var notificationsJsonStr []byte
	err = row.Scan(&thread.ID, &thread.Type, &thread.Repo, &thread.Title, &thread.Status, &thread.Pinned, &thread.Author, &thread.IsReviewRequest, &thread.URL, &thread.UpdatedAt, &thread.Unread, &notificationsJsonStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal(notificationsJsonStr, &thread.Notifications)
	return &thread, nil
}

func getAllThreads(db *sql.DB) (*[]NotificationThread, error) {
	rows, err := db.Query("SELECT * FROM threads")
	if err != nil {
		log.Println("Error querying threads:", err)
		return nil, err
	}
	defer rows.Close()

	var threads []NotificationThread
	for rows.Next() {
		var thread NotificationThread
		var notificationsJsonStr []byte
		err := rows.Scan(&thread.ID, &thread.Type, &thread.Repo, &thread.Title, &thread.Status, &thread.Pinned, &thread.Author, &thread.IsReviewRequest, &thread.URL, &thread.UpdatedAt, &thread.Unread, &notificationsJsonStr)
		if err != nil {
			log.Println("Error scanning thread:", err)
			return nil, err
		}
		json.Unmarshal(notificationsJsonStr, &thread.Notifications)
		threads = append(threads, thread)
	}

	if err := rows.Err(); err != nil {
		log.Println("Error iterating threads:", err)
		return nil, err
	}
	return &threads, nil
}

func addThreadToDB(thread *NotificationThread, db *sql.DB) error {
	notificationsJson, _ := json.Marshal(thread.Notifications)
	_, err = db.Exec(`
		INSERT INTO threads (ID, Type, Repo, Title, Status, Pinned, Author, IsReviewRequest, URL, UpdatedAt, Unread, Notifications)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		thread.ID,
		thread.Type,
		thread.Repo,
		thread.Title,
		thread.Status,
		thread.Pinned,
		thread.Author,
		thread.IsReviewRequest,
		thread.URL,
		thread.UpdatedAt,
		thread.Unread,
		notificationsJson,
	)
	return err
}

func updateThreadInDB(thread *NotificationThread, db *sql.DB) error {
	notificationsJson, _ := json.Marshal(thread.Notifications)
	_, err = db.Exec(`
		UPDATE threads
		SET Title=$1, Status=$2, Pinned=$3, IsReviewRequest=$4, UpdatedAt=$5, Unread=$6, Notifications=$7
		WHERE ID = $8`, 
		thread.Title,
		thread.Status,
		thread.Pinned,
		thread.IsReviewRequest,
		thread.UpdatedAt,
		thread.Unread,
		notificationsJson,
		thread.ID,
	)
	if err != nil {
		log.Println("Error updating thread in db:", err)
	}
	return err
}

func getNotificationFromID(id int64, db *sql.DB) (*Notification, error) {
	var n Notification
	query := `SELECT ID, ThreadID, Title, URL, UpdatedAt, Reason, Repository, Unread FROM notifications WHERE ID = ?`
	row := db.QueryRow(query, id)
	err := row.Scan(&n.ID, &n.ThreadID, &n.Title, &n.URL, &n.UpdatedAt, &n.Reason, &n.Repository, &n.Unread)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func getNotificationsForThread(threadID int64, db *sql.DB) (*[]Notification, error) {
	rows, err := db.Query("SELECT * FROM notifications WHERE ThreadID = ?", threadID)
	if err != nil {
		log.Println("Error querying notifications:", err)
		return nil, err
	}
	defer rows.Close()

	notifications := []Notification{}
	for rows.Next() {
		var notification Notification
		err := rows.Scan(&notification.ID, &notification.ThreadID, &notification.Title, &notification.URL, &notification.UpdatedAt, &notification.Reason, &notification.Repository, &notification.Unread)
		if err != nil {
			log.Println("Error scanning notification:", err)
			return nil, err
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		log.Println("Error iterating notifications:", err)
		return nil, err
	}

	if len(notifications) == 0 {
		return nil, nil
	}
	return &notifications, nil
}
