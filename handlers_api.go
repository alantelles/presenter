package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type mediaList struct {
	MediaList []string `json:"mediaList"`
}

func setMediaProviderContent(c *gin.Context) {
	var newContent mediaProviderContent
	if err := c.BindJSON(&newContent); err != nil {
		return
	}
	provider1 = newContent
	response := returnBody{
		Status:  http.StatusCreated,
		Message: "New content setup to provider1",
	}
	CORS(c)
	c.JSON(http.StatusCreated, response)
}

func getMediaProviderContent(c *gin.Context) {
	CORS(c)
	c.JSON(http.StatusOK, provider1)
}

func saveMedia(c *gin.Context) {
	var command media
	if err := c.BindJSON(&command); err != nil {
		return
	}
	categoryP, _ := findCategoryByName(command.Category)
	category := *categoryP
	title := command.Title + " - " + command.Author
	SaveTextFile(category, title, command.Content)
	response := returnBody{
		Status:  http.StatusCreated,
		Message: "New media saved",
	}
	CORS(c)
	c.JSON(http.StatusOK, response)
}

func getAllSongs(c *gin.Context) {
	songNames := loadMediaList(CategorySongs.Name)
	CORS(c)
	response := mediaList{MediaList: songNames}
	c.JSON(http.StatusOK, response)
}
