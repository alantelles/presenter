package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type mediaList struct {
	MediaList []string `json:"mediaList"`
}

func getSongsFolderList(c *gin.Context) {
	CORS(c)
	archivePath := c.Query("archive")
	folders, _ := loadSongFolders(CategorySongs.Name, archivePath)
	response := mediaList{
		MediaList: folders,
	}
	c.JSON(http.StatusOK, response)
}

func setMediaProviderContent(c *gin.Context) {
	providerId := c.Param("providerId")
	var newContent ProviderData
	if err := c.BindJSON(&newContent); err != nil {
		return
	}
	err := CopyIncomingProviderToExistent(providerId, newContent)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	response := returnBody{
		Status:     http.StatusCreated,
		Message:    "New content setup",
		ProviderId: providerId,
		Type:       newContent.Type,
		ContentId:  newContent.ContentId,
	}
	CORS(c)
	c.JSON(http.StatusCreated, response)
}

func getMediaProviderContent(c *gin.Context) {
	CORS(c)
	providerIds := c.QueryArray("providerId")
	responseData := map[string]ProviderData{}
	for _, providerId := range providerIds {
		responseData[providerId] = providers[providerId]
	}
	c.JSON(http.StatusOK, &responseData)
}

func saveMedia(c *gin.Context) {
	var command media
	if err := c.BindJSON(&command); err != nil {
		return
	}
	categoryP, _ := findCategoryByName(command.Category)
	category := *categoryP
	title := command.Title + " - " + command.Author
	saveTextFile(category, title, command.Content)
	response := returnBody{
		Status:  http.StatusCreated,
		Message: "New media saved",
	}
	CORS(c)
	c.JSON(http.StatusOK, response)
}

func getAllSongs(c *gin.Context) {
	songNames, _ := loadMediaList(CategorySongs.Name)
	CORS(c)
	response := mediaList{MediaList: songNames}
	c.JSON(http.StatusOK, response)
}

func getAllSongsFromFolder(c *gin.Context) {
	songNames, _ := loadMediaListFromFolder(
		CategorySongs.Name,
		c.Query("archive"),
		c.Query("folder"),
	)
	CORS(c)
	response := mediaList{MediaList: songNames}
	c.JSON(http.StatusOK, response)
}

func getSongContent(c *gin.Context) {
	song := c.Query("song")
	CORS(c)
	c.Data(http.StatusOK, "text/plain; charset=UTF-8", loadSongFile(song))
}

func discover(c *gin.Context) {
	validationCode := c.Query("validationCode")
	var vcr *string
	vcr = nil
	if validationCode != "" {
		vcr = &validationCode
	}
	response := returnBody{
		Status:     http.StatusOK,
		Message:    "Presenter up!",
		Validation: vcr,
	}
	CORS(c)
	c.JSON(http.StatusOK, response)
}
