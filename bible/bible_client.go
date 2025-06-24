package bible

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	BibleApiUrl = "https://www.abibliadigital.com.br/api"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
}

func FetchBooksList() (string, error) {
	url := BibleApiUrl + "/books"
	log.Println("Fetching books list from: " + url)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	auth := fmt.Sprintf("Bearer %s", "your-api-token-here") // Replace with your actual API token
	req.Header.Add("Authorization", auth)
	resp, err := client.Do(req)
	if err != nil {
		return "Error fetching books list: " + err.Error(), err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading response body: " + err.Error(), err
	}
	return string(body), nil
}
