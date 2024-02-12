package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
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

type DayCount struct {
	CommitCount uint64
	Day         time.Time
}

type CommitData struct {
	Hash string
	Date time.Time
	Repo string
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("GITHUB_API_TOKEN")
	_, err = getAllCommits(token)

	if err != nil {
		log.Fatal(err)
	}
}

func getDayCounts(commitTimes []CommitData) ([]DayCount, error) {
	for _, commitTime := range commitTimes {
		fmt.Println(commitTime)
	}
	return nil, nil
}

func getAllCommits(token string) ([]CommitData, error) {
	repos, err := getRepos(token)

	if err != nil {
		return nil, err
	}

	var commits []CommitData
	var mtx sync.Mutex
	var wg sync.WaitGroup

	wg.Add(len(repos))

	for _, repo := range repos {
		var specificRepo = repo
		go func() {
			defer wg.Done()

			repoCommits, err := getCommitsFromRepo(specificRepo.FullName, token)

			if err != nil {
				log.Fatal(err)
			}

			mtx.Lock()
			commits = append(commits, repoCommits...)
			mtx.Unlock()
		}()
	}

	wg.Wait()

	return commits, nil
}

func createGithubApiRequest(url string, token string) (*http.Request, error) {
	request, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	request.Header.Add("Accept", "application/vnd.github+json")
	request.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	return request, nil
}

func getRepos(token string) ([]RepositoryData, error) {
	// Get list of public repositories...
	// Fetch commits from each repository...
	// Save all the information to database...
	var reposEndpoint = path.Join(GITHUB_URL, "user/repos")

	client := &http.Client{}
	url := fmt.Sprintf("https://%s?per_page=100&type=public", reposEndpoint)

	request, err := createGithubApiRequest(url, token)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	var repositories []interface{}
	bytes, err := io.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(bytes), &repositories)
	if err != nil {
		return nil, err
	}

	var repositoryData = make([]RepositoryData, len(repositories))

	for i := 0; i < len(repositories); i++ {
		repository := repositories[i]

		id, _ := repository.(map[string]interface{})["id"].(uint64)
		name, _ := repository.(map[string]interface{})["name"].(string)
		fullName, _ := repository.(map[string]interface{})["full_name"].(string)
		repo := RepositoryData{
			Id:       id,
			Name:     name,
			FullName: fullName,
		}

		repositoryData[i] = repo
	}

	return repositoryData, nil
}

// Full name is {owner}/{slug}
// Example: johnmatthiggins/git-commit-scraper
func getCommitsFromRepo(fullName string, token string) ([]CommitData, error) {
	dayDuration, _ := time.ParseDuration("24h")
	var endpoint = fmt.Sprintf("https://%s", path.Join(GITHUB_URL, "repos", fullName, "commits"))
	var fiftyTwoWeeksAgo = time.Now().Add(-dayDuration * 52 * 7)
	var startISOTime = fiftyTwoWeeksAgo.Format(time.RFC3339)

	url := fmt.Sprintf("%s?committer=johnmatthiggins&per_page=100&since=%s", endpoint, startISOTime)

	client := &http.Client{}
	request, err := createGithubApiRequest(url, token)

	response, err := client.Do(request)

	if err != nil {
		return nil, err
	} else if response.StatusCode != 200 {
		log.Fatalf("\"GET\" Request to %s failed...", url)
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

	commitData, err := parseCommitData(commits, fullName)
	if err != nil {
		return nil, err
	}

	return commitData, nil
}

func parseCommitData(commits []interface{}, repoName string) ([]CommitData, error) {
	var commitData = make([]CommitData, len(commits))

	var i = 0
	for _, commit := range commits {
		hash, _ := commit.(map[string]interface{})["sha"].(string)

		// getting commit date
		// commit -> author -> date
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
			Repo: repoName,
		}

		commitData[i] = newCommitData
		i += 1
	}

	return commitData, nil
}
