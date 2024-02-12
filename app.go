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

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const USERNAME = "johnmatthiggins"
const GITHUB_URL = "api.github.com"

const CREATE_SCHEMA_SQL = `
    CREATE TABLE IF NOT EXISTS commit (
	hash      varchar(255) PRIMARY KEY,
	date      timestamp,
	repo_name varchar(255)
    );
`

type RepositoryData struct {
	Id       uint64
	Name     string
	FullName string
}

type CommitData struct {
	Hash string    `db:"hash"`
	Date time.Time `db:"date"`
	Repo string    `db:"repo_name"`
}

type DatabaseConfig struct {
	user     string
	database string
	host     string
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

	err = writeToDatabase(commits)
	if err != nil {
		log.Fatal(err)
	}
}

func parseDatabaseConfig() *DatabaseConfig {
	host := os.Getenv("DB_HOST")
	database := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")

	config := DatabaseConfig{
		user:     user,
		database: database,
		host:     host,
	}
	return &config
}

func (config *DatabaseConfig) toString() string {
	return fmt.Sprintf("database=%s user=%s sslmode=disable host=%s", config.database, config.user, config.host)
}

func writeToDatabase(commits []CommitData) error {
	db, err := sqlx.Connect("postgres", parseDatabaseConfig().toString())

	if err != nil {
		return err
	}

	db.MustExec(CREATE_SCHEMA_SQL)

	transaction := db.MustBegin()

	for _, commit := range commits {
		_, err := transaction.NamedExec("INSERT INTO commit (hash, date, repo_name) VALUES (:hash, :date, :repo_name);", &commit)
		if err != nil {
			return err
		}
	}
	transaction.Commit()

	db.Close()

	return nil
}

// func getDayCounts(commitTimes []CommitData) ([]DayCount, error) {
// 	for _, commitTime := range commitTimes {
// 		fmt.Println(commitTime)
// 	}
// 	return nil, nil
// }

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
	endpoint := fmt.Sprintf("https://%s", path.Join(GITHUB_URL, "repos", fullName, "commits"))
	fiftyTwoWeeksAgo := time.Now().Add(-dayDuration * 52 * 7)

	startISOTime := fiftyTwoWeeksAgo.Format(time.RFC3339)

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
	day, _ := time.ParseDuration("24h")
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
		truncatedCommitTime := commitTime.Truncate(day)

		if err != nil {
			return nil, err
		}

		newCommitData := CommitData{
			Hash: hash,
			Date: truncatedCommitTime,
			Repo: repoName,
		}

		commitData[i] = newCommitData
		i += 1
	}

	return commitData, nil
}
