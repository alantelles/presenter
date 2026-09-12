package main

import (
	"presenter/bible"

	"github.com/gin-gonic/gin"
)

// registerMediaRoutes wires up song/image/provider content management: the
// endpoints controllers use to publish, list, save and move media.
func registerMediaRoutes(router *gin.Engine) {
	router.POST("/api/content/set/:providerId", AuthMiddleware, setMediaProviderContent)
	router.GET("/api/content", getMediaProviderContent)
	router.POST("/api/media", AuthMiddleware, saveMedia)
	router.PUT("/api/media/move", AuthMiddleware, moveMedia)

	router.POST("/api/images", AuthMiddleware, uploadImage)

	router.GET("/api/songs", getAllSongs)
	router.GET("/api/songs/content", getSongContent)
	router.GET("/api/songs/folders", getSongsFolderList)
	router.GET("/api/songs/folder", getAllSongsFromFolder)
}

// registerViewRoutes wires up the HTML pages: controller UIs, the home page
// and the live presentation panel.
func registerViewRoutes(router *gin.Engine) {
	router.GET("/controller", AuthMiddleware, viewController)
	router.GET("/controller/:page", AuthMiddleware, viewController)
	router.GET("/", viewHome)
	router.GET("/live", viewPanel)
}

// registerMiscRoutes wires up standalone endpoints that don't belong to a
// bigger group.
func registerMiscRoutes(router *gin.Engine) {
	router.GET("/api/discover", discover)
}

// registerLyricsRoutes wires up the letras.mus.br lyrics lookup endpoints.
func registerLyricsRoutes(router *gin.Engine) {
	router.GET("/api/lyrics/letras", getSongLyricsFromLetras)
	router.GET("/api/lyrics/letras/song", getSongLyricFromLetrasByUrl)
}

// registerBibleRoutes wires up the bible package's endpoints.
func registerBibleRoutes(router *gin.Engine) {
	router.GET("/api/bible/books", bible.GetBooksList)
	router.GET("/api/bible/chapter/:version/:book/:chapter", bible.GetChapter)
}
