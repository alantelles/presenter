package main

import (
	"log"
	"net/http"
	"os"

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

func uploadImage(c *gin.Context) {
	CORS(c)
	form, _ := c.MultipartForm()
	files := form.File["files"]
	for _, file := range files {
		log.Println(file.Filename)
		savedName := "./media/images/" + file.Filename
		c.SaveUploadedFile(file, savedName)
		createThumbnail(savedName)
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK"})
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
		ContentId:  newContent.ContentID,
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
	var command Media
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

func moveMedia(c *gin.Context) {
	CORS(c)
	var command MoveMediaCommand
	if err := c.BindJSON(&command); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	categoryP, _ := findCategoryByName(command.Category)
	category := *categoryP
	src := MediaPath + category.Path + "/" + command.MediaID
	dest := MediaPath + category.Path + "/" + command.Destination
	createFolder(dest)
	dest = dest + "/" + command.MediaID
	err := os.Rename(src, dest)
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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
