package main

import (
	"errors"
	"net/http"
	"presenter/videos"

	"github.com/gin-gonic/gin"
)

// uploadVideo saves one or more uploaded videos (multipart field "files"),
// validating each one's format and size via the videos package.
func uploadVideo(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, videos.MaxUploadSize+(4<<20)) // +4MB framing overhead
	form, err := c.MultipartForm()
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
			return
		}
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
		savedName, err := videos.Save(fileHeader.Filename, file)
		file.Close()
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, videos.ErrTooLarge) {
				status = http.StatusRequestEntityTooLarge
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		saved = append(saved, savedName)
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK", "names": saved})
}

// listVideos returns every uploaded video's file name.
func listVideos(c *gin.Context) {
	names, err := videos.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"videos": names})
}

// getVideoContent serves the video file for ?name=.
func getVideoContent(c *gin.Context) {
	path, err := videos.ContentPath(c.Query("name"))
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, videos.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.File(path)
}

// deleteVideo deletes a video by name.
func deleteVideo(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		name = c.Query("name")
	}
	err := videos.Delete(name)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, videos.ErrNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "OK"})
}

type downloadVideoRequest struct {
	URL string `json:"url" binding:"required"`
}

// getDownloaderStatus reports whether yt-dlp and ffmpeg are installed on the server.
func getDownloaderStatus(c *gin.Context) {
	c.JSON(http.StatusOK, videos.CheckDownloaderTools())
}

// downloadVideo downloads a video from a URL using yt-dlp and merges into MP4.
func downloadVideo(c *gin.Context) {
	var req downloadVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL inválida ou ausente"})
		return
	}

	status := videos.CheckDownloaderTools()
	if !status.Available {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":         status.Message,
			"installScript": status.InstallScript,
			"tools":         status,
		})
		return
	}

	savedName, err := videos.DownloadVideo(c.Request.Context(), req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
		"name":    savedName,
	})
}
