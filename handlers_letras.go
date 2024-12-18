package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func getSongLyricFromLetras(c *gin.Context) {
	CORS(c)
	artist := c.Query("artista")
	songName := c.Query("musica")
	fetchedLyrics := getSongLyrics(artist, songName)
	c.Data(http.StatusOK, "text/plain; charset=UTF-8", []byte(fetchedLyrics))
}
