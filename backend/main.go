package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
  "net/url"
  "strings"
  "strconv"
	"reflect"

	"github.com/google/go-github/v55/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	"github.com/gorilla/handlers"
	"github.com/robfig/cron/v3"
)

var (
	client *github.Client
	ctx context.Context
	err error
	lastUpdate time.Time
)

type Notification struct {
	ID          int64 `json:"id"`
	ThreadID    int64
	Title       string `json:"message"`
	URL         string
	UpdatedAt   string `json:"timestamp"`
	Reason      string
	Repository  string
	Unread      bool
}

type NotificationThread struct {
	ID		int64 `json:"id"`
	Type	      	string `json:"type"`
	Repo	      	string `json:"repo"`
	Title         	string `json:"title"`
	Status        	string `json:"status"`
	Pinned        	bool `json:"pinned"`
	Author        	string `json:"author"`
	IsReviewRequest bool `json:"reviewRequested"`
	URL		string `json:"url"`
	UpdatedAt     	string `json:"updatedAt"`
	Unread		bool `json:"unread"`
	Notifications 	[]Notification `json:"notifications"`
}

func main() {
	// Github authentication
	ctx = context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	client = github.NewClient(tc)

	// Database setup
	db, err := sql.Open("sqlite3", "notifications.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableIfNotExists(db, "notifications", `
		ID INTEGER PRIMARY KEY AUTOINCREMENT,
		ThreadID TEXT NOT NULL UNIQUE,
		Title TEXT NOT NULL,
		URL TEXT NOT NULL,
		UpdatedAt TEXT NOT NULL,
		Reason TEXT NOT NULL,
		Repository TEXT NOT NULL,
		Unread INTEGER NOT NULL
	`)

	createTableIfNotExists(db, "threads", `
		ID INTEGER NOT NULL PRIMARY KEY,
		Type TEXT NOT NULL,
		Repo TEXT NOT NULL,
		Title TEXT NOT NULL,
		Status TEXT NOT NULL,
		Pinned BOOLEAN NOT NULL,
		Author TEXT NOT NULL,
		IsReviewRequest BOOLEAN NOT NULL,
		URL TEXT NOT NULL,
		UpdatedAt DATETIME NOT NULL,
		Unread BOOLEAN NOT NULL,
		Notifications JSON
	`)
	
	createTableIfNotExists(db, "settings", `
		id SERIAL PRIMARY KEY,
		key VARCHAR(255) UNIQUE NOT NULL,
		value VARCHAR(255) NOT NULL
	`)
	// Fetch and process notifications every 10 minutes
	fetchAndProcessNotifications(ctx, client, db)
	c := cron.New()
	c.AddFunc("*/10 * * * *", func() { fetchAndProcessNotifications(ctx, client, db) })
	c.Start()

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	corsOrigin := handlers.AllowedOrigins([]string{allowedOrigin})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	corsHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})

	// API endpoints
	http.HandleFunc("/threads", func(w http.ResponseWriter, r *http.Request) {
		fetchAndProcessNotifications(ctx, client, db)
		getThreadsHandler(w, r, db)
	})

	http.HandleFunc("/thread/notifications", func(w http.ResponseWriter, r *http.Request) {
		getNotificationsForThreadHandler(w, r, db)
	})

	http.HandleFunc("/updateThread", func(w http.ResponseWriter, r *http.Request) {
		updateThreadHandler(w, r, db)
	})

	http.HandleFunc("/updateSetting", func(w http.ResponseWriter, r *http.Request) {
		updateSettingHandler(w, r, db)
	})

	http.HandleFunc("/forcePull", func(w http.ResponseWriter, r *http.Request) {
		fetchAndProcessNotifications(ctx, client, db)
	})

	fmt.Println("Server listening on port 5000")
	log.Fatal(http.ListenAndServe(":5000", handlers.CORS(corsOrigin, corsMethods, corsHeaders)(http.DefaultServeMux)))
}

func LogStruct(v interface{}, name string) {
	val := reflect.ValueOf(v)

	// Check if the provided value is a pointer to a struct
	if val.Kind() == reflect.Ptr {
		val = val.Elem() // Dereference the pointer
	}

	if val.Kind() != reflect.Struct {
		log.Println("Provided value is not a struct")
		return
	}

	typ := val.Type()
	log.Printf("Struct %s:\n", name)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		name := typ.Field(i).Name
				// Check if the field is a pointer
		if field.Kind() == reflect.Ptr {
			if !field.IsNil() {
				field = field.Elem() // Dereference the pointer to get the value
			} else {
				log.Printf("%s: nil\n", name) // Log nil pointer
				continue
			}
		}
		log.Printf("%s: %v\n", name, field.Interface())
	}
}

func fetchAndProcessNotifications(ctx context.Context, client *github.Client, db *sql.DB) {
	// Fetch notifications
	lastUpdateTime, err := getLastUpdateTime(db)
	var notiListOptions github.NotificationListOptions
	switch err {
	case nil:
		notiListOptions = github.NotificationListOptions{Since: *lastUpdateTime}
		log.Printf("Fetching notifications from: %v\n", lastUpdateTime)
	case sql.ErrNoRows:
		notiListOptions = github.NotificationListOptions{}
		log.Println("Fetching notifications from beginning")
	default:
		log.Fatal(err)
	}
	notifications, _, err := client.Activity.ListNotifications(ctx, &notiListOptions)
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now()
	updatedAt := now.Format(time.RFC3339)
	updateLastUpdateTime(db, updatedAt)

	// Store in database
	for _, notification := range notifications {
		var threadTitle, threadStatus, owner, threadUrl string
		var threadID int64
		repo := notification.GetRepository()
		repositoryOwner := repo.GetOwner().GetLogin()
		repositoryName := repo.GetName()
		if sub := notification.GetSubject(); sub != nil {
			switch *((*sub).Type) {
			case "Issue":
				issueNumber, err := exactThreadNumber(sub.GetURL())
				if err != nil {
					log.Println("Error extracting issue number:", err)
					continue
				}
				// Fetch the Issue object
				issue, _, err := client.Issues.Get(ctx, repositoryOwner, repositoryName, issueNumber)
				if err != nil {
					log.Fatal(fmt.Printf("Error getting issue: %s", err))
				}
				
				threadID = *issue.ID
				threadTitle = *issue.Title
				threadStatus = *issue.State
				owner = *issue.User.Login
				threadUrl = *(issue.HTMLURL)
			case "PullRequest":
				prNumber, err := exactThreadNumber(sub.GetURL())
				if err != nil {
					log.Println("Error extracting pr number:", err)
					continue
				}
				// Fetch the PR object
				pr, _, err := client.PullRequests.Get(ctx, repositoryOwner, repositoryName, prNumber)
				if err != nil {
					log.Fatal(fmt.Printf("Error getting PR: %s", err))
				}
				threadID = pr.GetID()
				threadTitle = pr.GetTitle()
				threadStatus = pr.GetState()
				if pr.GetMerged() {
					threadStatus = "merged"
				}
				owner = pr.GetUser().GetLogin()
				threadUrl = *(pr.HTMLURL)
			}
			nID, _ := strconv.ParseInt(*(notification.ID), 10, 64)
			n := Notification {
				ID: nID,
				ThreadID: threadID,
				Title: notification.GetReason(),
				URL: *(notification.URL),
				UpdatedAt: updatedAt,
				Unread: true,
			}
			thread, err := queryThread(threadID, db)
			if err != nil {
				log.Fatal(fmt.Printf("Error query the thread: %s", err))
				continue
			}
			if thread == nil {
				thread := NotificationThread{
					ID:              threadID,
					Type:            *sub.Type,
					Repo:            repo.GetFullName(),
					Title:           threadTitle,
					Status:          threadStatus,
					Pinned:          false,
					Author:          owner,
					IsReviewRequest: false,
					URL:             threadUrl,
					UpdatedAt:       updatedAt,
					Unread:		 true,
					Notifications:   []Notification{n},
				}
				err = addOrUpdateThreadToDB(&thread, db)
				if err != nil {
					log.Println("Error creating thread:", err)
				}
				continue
			} else {
				contains := false
				for _, noti := range thread.Notifications {
					if noti.ID == n.ID {
						contains = true
						break
					}
				}
				if !contains {
					thread.Notifications = append(thread.Notifications, n)
				}
				thread.Status = threadStatus
				thread.Unread = true
				err = addOrUpdateThreadToDB(thread, db)
				if err != nil {
					log.Println("Error updating thread:", err)
				}
			}
		}
		return
	}
}

// Function to check if a slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true // Item found
		}
	}
	return false // Item not found
}

// extractThreadNumber extracts the PR or Issue number from the URL
func exactThreadNumber(threadUrl string) (int, error) {
    parsedURL, err := url.Parse(threadUrl)
    if err != nil {
        return 0, err
    }

    // Split the path
    pathParts := strings.Split(parsedURL.Path, "/")
    if len(pathParts) < 1 {
        return 0, fmt.Errorf("invalid PR URL format")
    }

    // The number should be the last element in the path
    threadNumberStr := pathParts[len(pathParts)-1]
    threadNumber, err := strconv.Atoi(threadNumberStr)
    if err != nil {
        return 0, fmt.Errorf("error converting thread number to integer: %v", err)
    }

    return threadNumber, nil
}
