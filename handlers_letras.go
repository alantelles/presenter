package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func getSongLyricsFromLetras(c *gin.Context) {
	artist := c.Query("artista")
	songName := c.Query("musica")
	fetchedLyrics := getSongLyrics(artist, songName)
	c.Data(http.StatusOK, ContentTypeText, []byte(fetchedLyrics))
}

func getSongLyricFromLetrasByUrl(c *gin.Context) {
	url := c.Query("url")
	fetchedLyrics := getSongLyricsByUrl(url)
	c.Data(http.StatusOK, ContentTypeText, []byte(fetchedLyrics))
}
