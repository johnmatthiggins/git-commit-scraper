package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/joho/godotenv"
)

const USERNAME = "johnmatthiggins"
const GITHUB_URL = "api.github.com"

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("GITHUB_API_TOKEN")

	// Get list of public repositories...
	// Fetch commits from each repository...
	// Save all the information to database...
	var reposEndpoint = path.Join(GITHUB_URL, fmt.Sprintf("users/%s", USERNAME))

	client := &http.Client{}
	request, err := http.NewRequest("GET", fmt.Sprintf("https://%s?per_page=100&type=owner", reposEndpoint), nil)
	request.Header.Add("Bearer", token)

	response, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(response)
	}
}
