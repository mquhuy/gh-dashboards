package main

import (
    "net/http"
)

type Subject struct {
    Title string `json`
    URL string `json`
    Type string `json`
}

type Notification struct {
    ID string `json: id`
    Unread bool `json`
    Reason string `json:", omitempty"`
    Subject Subject `json`
}

type User struct {
    Name string `json: login`
    Avatar string `json: avatar_url`
}

type PullRequest struct {
    URL string `json`
    Title string `json`
    User User `json`
    ClosedAt string `json: closed_at", omitempty`
    MergedAt string `json: merged_at", omitempty`
}

func sendGetRequestWithCustomHeader(url string, headers *map[string]string) (*http.Response, error) {
	client := &http.Client{}

	// Create a new request with the desired method and URL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Set custom headers (e.g., User-Agent)
        for key, value := range *headers {
            req.Header.Set(key, value)
        }

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
	        resp.Body.Close()
		return nil, err
	}

	return resp, nil
}
