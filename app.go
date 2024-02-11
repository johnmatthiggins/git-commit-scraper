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
const GITHUB_URL = "https://api.github.com"

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	token := os.Getenv("GITHUB_API_TOKEN")

	// Get list of public repositories...
	// Fetch commits from each repository...
	// Save all the information to database...
	const reposEndpoint = path.Join(GITHUB_URL, fmt.Sprintf("users/%s", USERNAME))

	request, err := http.NewRequest("GET", fmt.Sprintf("%s?per_page=100&type=owner", reposEndpoint))
	request.Header.Add("Bearer", "")
}
