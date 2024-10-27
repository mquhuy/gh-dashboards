package main

import (
	"log"
	"database/sql"
	"encoding/json"
	"net/http"
        "strconv"
)

type UpdateThreadRequest struct {
	ThreadID int64 `json:"thread_id"`
	Action	 string	`json:"action"`
}

type UpdateSettingRequest struct {
	Key string `json:"setting_key"`
	Value string `json:"setting_value,omitempty"`
}

func getThreadsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	log.Println("Serving all threads")
	threads, err := getAllThreads(db)
	if err != nil {
		http.Error(w, "Error retrieving threads", http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(threads)
}

func getNotificationsForThreadHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	threadID := r.URL.Query().Get("thread_id")
	if threadID == "" {
		http.Error(w, "Missing thread_id parameter", http.StatusBadRequest)
		return
	}
	i, err := strconv.ParseInt(threadID, 10, 64)
	if err != nil {
		http.Error(w, "Bad form thread_id parameter", http.StatusBadRequest)
		return
	}
	notifications, err := getNotificationsForThread(i, db)
	if err != nil {
		http.Error(w, "Error retrieving notifications", http.StatusInternalServerError)
		return
	}
	if notifications == nil {
		http.Error(w, "No notifications found for the given thread_id", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(*notifications)
}

func updateThreadHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var t UpdateThreadRequest
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
	    http.Error(w, err.Error(), http.StatusBadRequest)
	    return
	}

	thread, err := queryThread(t.ThreadID, db)
	if err != nil || thread == nil {
		http.Error(w, "Error query the thread", http.StatusInternalServerError)
	}
	switch t.Action {
	case "pin":
		thread.Pinned = true
	case "unpin":
		thread.Pinned = false
	case "read":
		thread.Unread = false
	case "unread":
		thread.Unread = true
	case "togglePin":
		thread.Pinned = !thread.Pinned
	case "toggleRead":
		thread.Unread = !thread.Unread
	}
	updateThreadInDB(thread, db)
}

func updateSettingHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var t UpdateSettingRequest
	err := json.NewDecoder(r.Body).Decode(&t)
	if err != nil {
	    http.Error(w, err.Error(), http.StatusBadRequest)
	    return
	}
	switch r.Method {
	case http.MethodGet:
		value, err := querySetting(db, t.Key)
		if err != nil {
			http.Error(w, "Error retrieving notifications", http.StatusInternalServerError)
			return
		}
		t.Value = value
		json.NewEncoder(w).Encode(t)
	case http.MethodPost:
		updateSettingValue(db, t.Key, t.Value)
	}
}
