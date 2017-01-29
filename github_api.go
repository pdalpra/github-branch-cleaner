package main

import (
	"fmt"

	"github.com/google/go-github/github"
	"gopkg.in/fatih/set.v0"
)

func getAllRepositories(client *github.Client, organization string) ([]*github.Repository, error) {
	var (
		repositories []*github.Repository
		resp         = new(github.Response)
		listOpts     = &github.RepositoryListByOrgOptions{"sources", github.ListOptions{PerPage: 100}}
	)

	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		fetched, newResp, err := client.Repositories.ListByOrg(organization, listOpts)
		resp = newResp
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, fetched...)

	}
	return repositories, nil
}

func processRepository(client *github.Client, repository *github.Repository) error {
	var (
		owner         = *repository.Owner.Login
		repoName      = *repository.Name
		exclusionList = set.NewNonTS("develop", "release", "master")
	)
	allClosed, err := pullRequestsByState(client, owner, repoName, "closed")
	if err != nil {
		return err
	}
	allOpen, err := pullRequestsByState(client, owner, repoName, "open")
	if err != nil {
		return err
	}
	allBranches, err := listBranches(client, owner, repoName)
	if err != nil {
		return err
	}
	// Collect branches than are currently in use as target or source branch, to avoid deleting them
	for _, open := range allOpen {
		exclusionList.Add(*open.Base.Ref)
		exclusionList.Add(*open.Head.Ref)
	}
	for _, closedPR := range allClosed {
		for _, branch := range allBranches {
			if branch == *closedPR.Head.Ref && *closedPR.Head.User.Login == owner {
				msgPrefix := fmt.Sprintf("%s/%s#%d => ", owner, repoName, *closedPR.Number)
				if !exclusionList.Has(*closedPR.Head.Ref) {
					if _, err := client.Git.DeleteRef(owner, repoName, fmt.Sprintf("refs/%s", *closedPR.Head.Ref)); err != nil {
						return err
					}
					fmt.Printf("%s merged and unused branch %s deleted.\n", msgPrefix, *closedPR.Head.Ref)
				}
			}
		}
	}
	return nil
}

func pullRequestsByState(client *github.Client, owner string, repoName string, state string) ([]*github.PullRequest, error) {
	var (
		pullRequests []*github.PullRequest
		resp         = new(github.Response)
		listOpts     = &github.PullRequestListOptions{state, "", "", "", "", github.ListOptions{PerPage: 100}}
	)

	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		fetched, newResp, err := client.PullRequests.List(owner, repoName, listOpts)
		resp = newResp
		if err != nil {
			return nil, err
		}
		pullRequests = append(pullRequests, fetched...)

	}
	return pullRequests, nil
}

func listBranches(client *github.Client, owner string, repoName string) ([]string, error) {
	var (
		branchNames []string
		resp        = new(github.Response)
		listOpts    = &github.ListOptions{PerPage: 100}
	)

	// Get all pages
	resp.NextPage = 1
	for resp.NextPage != 0 {
		listOpts.Page = resp.NextPage
		fetched, newResp, err := client.Repositories.ListBranches(owner, repoName, listOpts)
		resp = newResp
		if err != nil {
			return nil, err
		}
		for _, branch := range fetched {
			branchNames = append(branchNames, *branch.Name)
		}
	}
	return branchNames, nil
}
