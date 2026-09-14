package main

import (
	"presenter/bible"

	"github.com/gin-gonic/gin"
)

// registerMediaRoutes wires up song/provider content management: the
// endpoints controllers use to publish, list, save and move media.
func registerMediaRoutes(router *gin.Engine, app *App) {
	router.POST("/api/content/set/:providerId", app.AuthMiddleware, app.setMediaProviderContent)
	router.GET("/api/content", app.getMediaProviderContent)
	router.POST("/api/media", app.AuthMiddleware, saveMedia)
	router.PUT("/api/media/move", app.AuthMiddleware, moveMedia)

	router.GET("/api/songs", getAllSongs)
	router.GET("/api/songs/content", getSongContent)
	router.GET("/api/songs/folders", getSongsFolderList)
	router.GET("/api/songs/folder", getAllSongsFromFolder)
}

// registerImageRoutes wires up image upload, listing and retrieval — the
// endpoints the images controller and the live panel use to publish, list
// and read back uploaded images and their thumbnails.
func registerImageRoutes(router *gin.Engine, app *App) {
	router.POST("/api/images", app.AuthMiddleware, uploadImage)
	router.GET("/api/images", listImages)
	router.GET("/api/images/content", getImageContent)
	router.GET("/api/images/thumb", getImageThumb)
}

// registerProviderRoutes wires up provider management: creating, listing
// and deleting the channels controllers publish content to.
func registerProviderRoutes(router *gin.Engine, app *App) {
	router.GET("/api/providers", app.listProviders)
	router.POST("/api/providers", app.AuthMiddleware, app.createProvider)
	router.DELETE("/api/providers/:id", app.AuthMiddleware, app.deleteProvider)
	router.DELETE("/api/providers", app.AuthMiddleware, app.deleteAllProviders)
}

// registerViewRoutes wires up the HTML pages: controller UIs, the home page
// and the live presentation panel.
func registerViewRoutes(router *gin.Engine, app *App) {
	router.GET("/controller", app.AuthMiddleware, app.viewController)
	router.GET("/controller/:page", app.AuthMiddleware, app.viewController)
	router.GET("/", app.viewHome)
	router.GET("/live", app.viewPanel)
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
