package main

import (
	"errors"
	"net/http"
	"presenter/images"

	"github.com/gin-gonic/gin"
)

// uploadImage saves one or more uploaded images (multipart field "files"),
// validating each one's format and size via the images package.
func uploadImage(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}
	saved := make([]string, 0, len(files))
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		savedName, err := images.Save(fileHeader.Filename, file)
		file.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		saved = append(saved, savedName)
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK", "names": saved})
}

// listImages returns every uploaded image's file name.
func listImages(c *gin.Context) {
	names, err := images.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"images": names})
}

// getImageContent serves the original file for ?name=.
func getImageContent(c *gin.Context) {
	path, err := images.ContentPath(c.Query("name"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, images.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.File(path)
}

// getImageThumb serves the thumbnail for ?name=.
func getImageThumb(c *gin.Context) {
	path, err := images.ThumbPath(c.Query("name"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, images.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.File(path)
}
