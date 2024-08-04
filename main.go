package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "encoding/json"
)

const (
    URL = "https://api.github.com/notifications"
)

func main() {

    gh_token := os.Getenv("GH_TOKEN")

    headers := map[string]string{
        "Authorization": fmt.Sprintf("Bearer %s", gh_token),
    }

    resp, err := sendGetRequestWithCustomHeader(URL, &headers)
    var notifications []Notification

    if err != nil {
        log.Fatal(err)
    }

    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        log.Fatal(err)
    }

    if err = json.Unmarshal(body, &notifications); err != nil {
        log.Fatal(err)
    }

    for _, noti := range notifications {
        if noti.Subject.Type == "PullRequest" {
            resp, err := sendGetRequestWithCustomHeader(noti.Subject.URL, &headers)
            if err != nil {
                log.Fatal(err)
            }
            defer resp.Body.Close()
            body, err := ioutil.ReadAll(resp.Body)
            if err != nil {
                log.Fatal(err)
            }
            var pr PullRequest
            if err = json.Unmarshal(body, &pr); err != nil {
                log.Fatal(err)
            }
            if pr.ClosedAt == "" && pr.MergedAt == "" {
                fmt.Println(pr.Title)
            }
        }
    }
}
