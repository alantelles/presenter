package main

import (
	"log"
	"net/http"
	"os"
	"presenter/storage"

	"github.com/gin-gonic/gin"
)

type mediaList struct {
	MediaList []string `json:"mediaList"`
}

func getSongsFolderList(c *gin.Context) {
	archivePath := c.Query("archive")
	folders, _ := storage.ListFolders(storage.Songs.Name, archivePath)
	response := mediaList{
		MediaList: folders,
	}
	c.JSON(http.StatusOK, response)
}

func uploadImage(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files"]
	for _, file := range files {
		log.Println(file.Filename)
		savedName := "./" + storage.BasePath() + "images/" + file.Filename
		c.SaveUploadedFile(file, savedName)
		createThumbnail(savedName)
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}

func (a *App) setMediaProviderContent(c *gin.Context) {
	providerId := c.Param("providerId")
	var newContent ProviderData
	if err := c.BindJSON(&newContent); err != nil {
		return
	}
	err := a.CopyIncomingProviderToExistent(providerId, newContent)
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
	c.JSON(http.StatusCreated, response)
}

func (a *App) getMediaProviderContent(c *gin.Context) {
	providerIds := c.QueryArray("providerId")
	responseData := map[string]ProviderData{}
	for _, providerId := range providerIds {
		responseData[providerId] = a.Providers.Get(providerId)
	}
	c.JSON(http.StatusOK, &responseData)
}

func saveMedia(c *gin.Context) {
	var command Media
	if err := c.BindJSON(&command); err != nil {
		return
	}
	category, err := storage.FindCategoryByName(command.Category)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	title := command.Title + " - " + command.Author
	storage.SaveTextFile(*category, title, command.Content)
	response := returnBody{
		Status:  http.StatusCreated,
		Message: "New media saved",
	}
	c.JSON(http.StatusOK, response)
}

func moveMedia(c *gin.Context) {
	var command MoveMediaCommand
	if err := c.BindJSON(&command); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category, err := storage.FindCategoryByName(command.Category)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	src := storage.BasePath() + category.Path + "/" + command.MediaID
	dest := storage.BasePath() + category.Path + "/" + command.Destination
	createFolder(dest)
	dest = dest + "/" + command.MediaID
	err = os.Rename(src, dest)
	if err != nil {
		log.Print(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func getAllSongs(c *gin.Context) {
	songNames, _ := storage.List(storage.Songs.Name)
	response := mediaList{MediaList: songNames}
	c.JSON(http.StatusOK, response)
}

func getAllSongsFromFolder(c *gin.Context) {
	songNames, _ := storage.ListFromFolder(
		storage.Songs.Name,
		c.Query("archive"),
		c.Query("folder"),
	)
	response := mediaList{MediaList: songNames}
	c.JSON(http.StatusOK, response)
}

func getSongContent(c *gin.Context) {
	song := c.Query("song")
	c.Data(http.StatusOK, ContentTypeText, storage.LoadSongFile(song))
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
	c.JSON(http.StatusOK, response)
}
