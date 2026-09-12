package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func (a *App) getAuthAsB64() string {
	return base64.StdEncoding.EncodeToString([]byte(a.Config.BasicAuthUser + ":" + a.Config.BasicAuthPass))
}

func (a *App) insertAddressOnContent(content []byte) []byte {
	return bytes.Replace(content, []byte(AppLocationToken), []byte(a.Config.Location), -1)
}

func (a *App) insertAuthTokenOnContent(content []byte) []byte {
	return bytes.Replace(content, []byte(AppAuthToken), []byte(a.getAuthAsB64()), -1)
}

func (a *App) getHtmlPage(path string) ([]byte, error) {
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
	retrieved = a.insertAddressOnContent(retrieved)
	return a.insertAuthTokenOnContent(retrieved), err
}

func (a *App) viewPanel(c *gin.Context) {
	read, _ := a.getHtmlPage("templates/panels/default.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func (a *App) viewController(c *gin.Context) {
	page := c.Param("page")
	if page == "" {
		page = "songs"
	}
	read, _ := a.getHtmlPage("templates/controllers/" + page + ".html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func (a *App) viewHome(c *gin.Context) {
	read, _ := a.getHtmlPage("templates/index.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}
