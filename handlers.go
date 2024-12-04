package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func getHtmlPage(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	retrieved, err := io.ReadAll(file)
	return insertAddressOnContent(retrieved), err
}

func viewPanel(c *gin.Context) {
	read, _ := getHtmlPage("templates/panels/default.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func viewController(c *gin.Context) {
	read, _ := getHtmlPage("templates/controllers/default.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}
