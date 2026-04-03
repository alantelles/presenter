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
		log.Print(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Print(err)
		}
	}()
	retrieved, err := io.ReadAll(file)
	retrieved = insertAddressOnContent(retrieved)
	return insertAuthTokenOnContent(retrieved), err
}

func viewPanel(c *gin.Context) {
	read, _ := getHtmlPage("templates/panels/default.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func viewController(c *gin.Context) {
	page := c.Param("page")
	if page == "" {
		page = "songs"
	}
	read, _ := getHtmlPage("templates/controllers/" + page + ".html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func viewHome(c *gin.Context) {
	read, _ := getHtmlPage("templates/index.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}
