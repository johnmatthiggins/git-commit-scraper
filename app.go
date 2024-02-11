package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/joho/godotenv"
)

const USERNAME = "johnmatthiggins"
const GITHUB_URL = "api.github.com"

type RepositoryData struct {
	Id       uint64
	Name     string
	FullName string
}

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("GITHUB_API_TOKEN")

	// Get list of public repositories...
	// Fetch commits from each repository...
	// Save all the information to database...
	var reposEndpoint = path.Join(GITHUB_URL, "user/repos")

	client := &http.Client{}
	request, err := http.NewRequest("GET", fmt.Sprintf("https://%s?per_page=100&type=public", reposEndpoint), nil)
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	request.Header.Add("Accept", "application/vnd.github+json")
	request.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		// data, err := io.ReadAll(response.Body)

		// if err != nil {
		// 	log.Fatal(err)
		// }

		// fmt.Println(string(data))

		var repositories []interface{}
		bytes, err := io.ReadAll(response.Body)

		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal([]byte(bytes), &repositories)
		if err != nil {
			log.Fatal(err)
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

		fmt.Println(repositoryData)
	}
}
