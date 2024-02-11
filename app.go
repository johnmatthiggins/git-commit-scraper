package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/joho/godotenv"
)

const USERNAME = "johnmatthiggins"
const GITHUB_URL = "api.github.com"

type RepositoryData struct {
	Id       uint64
	Name     string
	FullName string
}

type CommitData struct {
	Hash string
	Date time.Time
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("GITHUB_API_TOKEN")
	commits, err := getAllCommits(token)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(commits)
}

func getAllCommits(token string) ([]CommitData, error) {
	repos, err := getRepos(token)

	if err != nil {
		return nil, err
	}

	var commits []CommitData

	for _, repo := range repos {
		repoCommits, err := getCommitsFromRepo(repo.FullName, token)
		if err != nil {
			return nil, err
		}

		commits = append(commits, repoCommits...)
	}

	return commits, nil
}

func getRepos(token string) ([]RepositoryData, error) {
	// Get list of public repositories...
	// Fetch commits from each repository...
	// Save all the information to database...
	var reposEndpoint = path.Join(GITHUB_URL, "user/repos")

	client := &http.Client{}
	request, err := http.NewRequest("GET", fmt.Sprintf("https://%s?per_page=100&type=public", reposEndpoint), nil)

	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	request.Header.Add("Accept", "application/vnd.github+json")
	request.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	response, err := client.Do(request)
	if err == nil {
		var repositories []interface{}
		bytes, err := io.ReadAll(response.Body)

		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(bytes), &repositories)
		if err != nil {
			return nil, err
		}

		var repositoryData []RepositoryData

		for _, repository := range repositories {
			id, _ := repository.(map[string]interface{})["id"].(uint64)
			name, _ := repository.(map[string]interface{})["name"].(string)
			fullName, _ := repository.(map[string]interface{})["full_name"].(string)
			repo := RepositoryData{
				Id:       id,
				Name:     name,
				FullName: fullName,
			}

			repositoryData = append(repositoryData, repo)
		}

		return repositoryData, nil
	}

	return nil, err
}

// Full name is {owner}/{slug}
// Example: johnmatthiggins/git-commit-scraper
func getCommitsFromRepo(fullName string, token string) ([]CommitData, error) {
	dayDuration, _ := time.ParseDuration("24h")
	var endpoint string = fmt.Sprintf("https://%s", path.Join(GITHUB_URL, "repos", fullName, "commits"))
	var fiftyTwoWeeksAgo = time.Now().Add(-dayDuration * 52 * 7)
	var startISOTime = fiftyTwoWeeksAgo.Format(time.RFC3339)
	var url = fmt.Sprintf("%s?committer=johnmatthiggins&since=%s", endpoint, startISOTime)

	client := &http.Client{}
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	request.Header.Add("Accept", "application/vnd.github+json")
	request.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	var commits []interface{}
	bytes, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(bytes), &commits)
	if err != nil {
		return nil, err
	}

	var commitData []CommitData

	for _, commit := range commits {
		// commit -> author -> date
		hash, _ := commit.(map[string]interface{})["sha"].(string)

		// getting commit time
		innerCommit, _ := commit.(map[string]interface{})["commit"]
		commitAuthor, _ := innerCommit.(map[string]interface{})["author"]
		commitTimeStr, _ := commitAuthor.(map[string]interface{})["date"].(string)
		commitTime, err := time.Parse(time.RFC3339, commitTimeStr)

		if err != nil {
			return nil, err
		}

		newCommitData := CommitData{
			Hash: hash,
			Date: commitTime,
		}

		commitData = append(commitData, newCommitData)
	}

	return commitData, nil
}
