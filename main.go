package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofri/go-github-pagination/githubpagination"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_primary_ratelimit"
	"github.com/gofri/go-github-ratelimit/v2/github_ratelimit/github_secondary_ratelimit"
	"github.com/google/go-github/v71/github"
)

type datapoint struct {
	Day          time.Time `json:"day"`
	OpenIssues   int       `json:"open_issues"`
	ClosedIssues int       `json:"closed_issues"`
	OpenPRs      int       `json:"open_prs"`
	ClosedPRs    int       `json:"closed:prs"`
}

type report struct {
	Timeline []datapoint `json:"timeline"`
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("expected the Github repo path as one argument. `go run main.go example-org/example-repo`")
	}
	path := strings.Split(os.Args[1], "/")
	if len(path) != 2 {
		log.Fatalf("expected Github repo path to contain one slash")
	}
	org := path[0]
	repo := path[1]

	ctx := context.Background()

	rateLimiter := github_ratelimit.New(nil,
		github_primary_ratelimit.WithLimitDetectedCallback(func(ctx *github_primary_ratelimit.CallbackContext) {
			now := time.Now()
			timeUntilReset := ctx.ResetTime.Sub(now)
			fmt.Printf("Primary rate limit detected: category %s, reset time: %v\n", ctx.Category, ctx.ResetTime)

			if timeUntilReset > 0 {
				fmt.Printf("Waiting %.fs until rate limit reset...\n",
					timeUntilReset.Seconds())
				time.Sleep(timeUntilReset)
				fmt.Println("Rate limit reset completed, continuing...")
			}
		}),
		github_secondary_ratelimit.WithLimitDetectedCallback(func(ctx *github_secondary_ratelimit.CallbackContext) {
			fmt.Printf("Secondary rate limit detected: reset time: %v, total sleep time: %v\n", ctx.ResetTime, ctx.TotalSleepTime)
		}),
	)
	paginator := githubpagination.NewClient(rateLimiter,
		githubpagination.WithPerPage(50), // default to 100 results per page
		githubpagination.WithPaginationEnabled(),
	)

	token := os.Getenv("GITHUB_TOKEN")
	client := github.NewClient(paginator).WithAuthToken(token)

	opt := &github.IssueListByRepoOptions{
		State: "all",
	}
	var allIssues []*github.Issue
	for {
		issues, resp, err := client.Issues.ListByRepo(context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true), org, repo, opt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Fetched page %v / %v\n", resp.NextPage-1, resp.LastPage-1)
		allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	oldestTime := allIssues[len(allIssues)-1].CreatedAt
	oldestDay := time.Date(oldestTime.Year(), oldestTime.Month(), oldestTime.Day(), 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	numDays := int(now.Sub(oldestDay).Hours() / 24)

	data := make([]datapoint, numDays+1)
	for i := 0; i < numDays+1; i++ {
		data[i] = datapoint{
			Day: oldestDay.Add(time.Duration(i*24) * time.Hour),
		}
	}
	for _, issue := range allIssues {
		createdDay := time.Date(issue.CreatedAt.Year(), issue.CreatedAt.Month(), issue.CreatedAt.Day(), 0, 0, 0, 0, time.UTC)
		closedDay := now
		if issue.ClosedAt != nil {
			closedDay = time.Date(issue.ClosedAt.Year(), issue.ClosedAt.Month(), issue.ClosedAt.Day(), 0, 0, 0, 0, time.UTC)
		}
		for d := createdDay; !d.After(closedDay); d = d.AddDate(0, 0, 1) {
			index := int(d.Sub(oldestDay).Hours() / 24)
			if issue.IsPullRequest() {
				data[index].OpenPRs++
			} else {
				data[index].OpenIssues++
			}
		}
	}
	r := report{
		Timeline: data,
	}
	m, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join("data", org), os.ModePerm); err != nil {
		log.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join("data", org, fmt.Sprintf("%v.json", repo)), m, 0644); err != nil {
		log.Fatal(err)
	}
}
