package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/nozzle/throttler"
	"golang.org/x/oauth2"
)

func main() {
	var (
		startTime    = time.Now()
		token        = getEnvVar("GITHUB_TOKEN")
		organization = getEnvVar("GITHUB_ORGANIZATION")
	)

	// Setup Github API client, with persistent caching
	var (
		cache          = diskcache.New("./github-cache")
		cacheTransport = httpcache.NewTransport(cache)
		tokenSource    = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		authTransport  = oauth2.Transport{Source: tokenSource, Base: cacheTransport}
		client         = github.NewClient(&http.Client{Transport: &authTransport})
	)

	// 1. List all repositories
	fmt.Print("Fetching all repositories...\n")
	repositories, err := getAllRepositories(client, organization)
	logAndExitIfError(err)
	fmt.Printf("Fetched all repositories, found %d.\n", len(repositories))
	t := throttler.New(2, len(repositories))

	for _, repository := range repositories {
		go func(c *github.Client, repo *github.Repository) {
			t.Done(processRepository(c, repo))
		}(client, repository)
		if errCount := t.Throttle(); errCount != 0 {
			logAndExitIfError(t.Err())
		}
	}

	fmt.Printf("Total proccessing time: %.2fs.\n", time.Now().Sub(startTime).Seconds())
}

func logAndExitIfError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getEnvVar(varName string) string {
	v := os.Getenv(varName)
	if v == "" {
		logAndExitIfError(fmt.Errorf("%s environment variable is not defined !", varName))
	}
	return v
}
