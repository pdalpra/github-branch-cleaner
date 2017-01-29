package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/nozzle/throttler"
	"golang.org/x/oauth2"
)

func main() {
	var (
		startTime     = time.Now()
		token         = requiredEnvVar("GITHUB_TOKEN")
		organization  = requiredEnvVar("GITHUB_ORGANIZATION")
		exclusionList = parseExclusionList(optionalEnvVar("EXCLUSION_LIST", "master"))
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
			t.Done(processRepository(c, repo, exclusionList))
		}(client, repository)
		if errCount := t.Throttle(); errCount != 0 {
			logAndExitIfError(t.Err())
		}
	}

	fmt.Printf("Total proccessing time: %.2fs.\n", time.Now().Sub(startTime).Seconds())
}
