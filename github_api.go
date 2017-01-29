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

func processRepository(client *github.Client, repository *github.Repository, excludedBranches []string, dryRun bool) error {
	var (
		owner    = *repository.Owner.Login
		repoName = *repository.Name
	)
	// Collect branches than are currently in use as target or source branch in open PRs, to avoid deleting them
	openPRs, err := pullRequestsByState(client, owner, repoName, "open")
	if err != nil {
		return err
	}
	excluded := buildExclusionList(excludedBranches, openPRs)

	// Collect all closed PRs to scan
	closedPRs, err := pullRequestsByState(client, owner, repoName, "closed")
	if err != nil {
		return err
	}

	// Collect all existing branches, try not to delete already deleted branches
	existingBranches, err := listBranches(client, owner, repoName)
	if err != nil {
		return err
	}

	for _, closedPR := range closedPRs {
		var (
			sourceBranch = *closedPR.Head.Ref
			sourceRepo   = *closedPR.Head.User.Login
		)
		for _, branch := range existingBranches {
			// Delete if:
			// -> the old source branch matches an existing source branch
			// -> the source branch was on the same repository (don't touch forks, leave it to jessfraz/ghb0t)
			// -> the branch is not in the exclusion list
			if branch == sourceBranch && sourceRepo == owner && !excluded.Has(sourceBranch) {
				if !dryRun {
					if _, err := client.Git.DeleteRef(owner, repoName, fmt.Sprintf("refs/%s", sourceBranch)); err != nil {
						return err
					}
				}
				fmt.Printf("%s/%s#%d => unused branch %s deleted.\n", owner, repoName, *closedPR.Number, sourceBranch)

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

func buildExclusionList(excludedBranches []string, openPRs []*github.PullRequest) *set.SetNonTS {
	excluded := set.NewNonTS()
	for _, branch := range excludedBranches {
		excluded.Add(branch)
	}
	for _, openPR := range openPRs {
		excluded.Add(*openPR.Base.Ref)
		excluded.Add(*openPR.Head.Ref)
	}
	return excluded
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
