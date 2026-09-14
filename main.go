package main

import (
	"fmt"
	"log"
	"presenter/flags"
	"presenter/fsutil"
	"presenter/providers"
	"presenter/storage"

	"github.com/gin-gonic/gin"
)

const (
	ContentTypeBinary = "application/octet-stream"
	ContentTypeForm   = "application/x-www-form-urlencoded"
	ContentTypeJSON   = "application/json; charset=utf-8"
	ContentTypeHTML   = "text/html; charset=utf-8"
	ContentTypeText   = "text/plain; charset=utf-8"

	AppLocationToken = "{{APP_LOCATION}}"

	AppAuthToken = "{{BASIC_AUTH_TOKEN}}"

	TypeCommand = "COMMAND"
	TypeText    = "TEXT"
	TypeImage   = "IMAGE"
	TypeVideo   = "VIDEO"
	TypeAudio   = "AUDIO"
	TypeBinary  = "BINARY"
)

// ProviderData is an alias for providers.Data, kept under this name since
// it's the JSON shape handlers/templates already speak.
type ProviderData = providers.Data

type Media struct {
	Category string `json:"category"`
	Content  string `json:"content"`
	Title    string `json:"title"`
	Author   string `json:"author"`
}

type MoveMediaCommand struct {
	MediaID     string `json:"mediaId" binding:"required"`
	Destination string `json:"destination" binding:"required"`
	Category    string `json:"category" binding:"required"`
}

type returnBody struct {
	Status     int     `json:"status"`
	Message    string  `json:"message"`
	Validation *string `json:"validation,omitempty"`
	ProviderId string  `json:"providerId"`
	Type       string  `json:"type"`
	ContentId  string  `json:"contentId,omitempty"`
}

func CORSMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	c.Next()
}

func createFolder(path string) {
	created, err := fsutil.CreateFolder(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	if created {
		log.Printf("Diretório %s criado com sucesso", path)
	}
}

func createDefaultFolders() {
	createFolder(storage.BasePath() + "songs")
	createFolder(storage.BasePath() + "images")
	createFolder(storage.BasePath() + "images/thumbs")
}

func main() {
	flags.ProcessFlags()
	app := NewApp(NewConfig())
	storage.SetBasePath(app.Config.MediaPath)
	createDefaultFolders()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/api/content"),
		gin.Recovery(),
		CORSMiddleware,
	)
	router.Static("/static", "./static")

	registerMediaRoutes(router, app)
	registerImageRoutes(router, app)
	registerProviderRoutes(router, app)
	registerViewRoutes(router, app)
	registerMiscRoutes(router)
	registerLyricsRoutes(router)
	registerBibleRoutes(router)

	log.Print("PRESENTER - Desenvolvido por Alan Telles")
	log.Print("Iniciando serviço...")
	log.Print("Endereço: " + app.Config.Location)
	router.Run(fmt.Sprintf("0.0.0.0:%d", app.Config.Port))
}
