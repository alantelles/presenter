package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"presenter/bible"
	"presenter/flags"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ContentTypeBinary = "application/octet-stream"
	ContentTypeForm   = "application/x-www-form-urlencoded"
	ContentTypeJSON   = "application/json; charset=utf-8"
	ContentTypeHTML   = "text/html; charset=utf-8"
	ContentTypeText   = "text/plain; charset=utf-8"

	AppLocationToken = "{{APP_LOCATION}}"

	TypeCommand = "COMMAND"
	TypeText    = "TEXT"
	TypeImage   = "IMAGE"
	TypeVideo   = "VIDEO"
	TypeAudio   = "AUDIO"
	TypeBinary  = "BINARY"
)

type ProviderData struct {
	Content   string `json:"content"`
	Type      string `json:"type,omitempty"`
	ContentId string `json:"contentId,omitempty"`
}

type media struct {
	Category string `json:"category"`
	Content  string `json:"content"`
	Title    string `json:"title"`
	Author   string `json:"author"`
}

type returnBody struct {
	Status     int     `json:"status"`
	Message    string  `json:"message"`
	Validation *string `json:"validation,omitempty"`
	ProviderId string  `json:"providerId"`
	Type       string  `json:"type"`
	ContentId  string  `json:"contentId,omitempty"`
}

var port = 8080 // TODO: receive this by running argument
var location string
var usePort = true

func CORS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
}

func getScheme(secure bool) string {
	if secure {
		return "https://"
	}
	return "http://"
}

func getPortSuffix() string {
	if usePort {
		return ":" + fmt.Sprint(port)
	}
	return ""
}

func setLocation() string {
	address := getLocalIp()
	scheme := getScheme(false) // TODO: receive this by running argument
	return scheme + address + getPortSuffix()
}

func varSetup() {
	location = flags.GetLocation()
	if location == "" {
		location = setLocation()
	}
}

func getLocalIp() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Print("Não foi possível obter o IP. Será utilizado localhost.")
		log.Print(err)
		return "localhost"
	}
	defer conn.Close()
	full := conn.LocalAddr().String()

	return full[:strings.Index(full, ":")]
}

func insertAddressOnContent(content []byte) []byte {
	return bytes.Replace(
		content,
		[]byte(AppLocationToken),
		[]byte(location),
		-1,
	)
}

// exists returns whether the given file or directory exists
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func createFolder(path string) {
	exists, _ := pathExists(path)
	if exists {
		return
	}
	err := os.MkdirAll(path, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("Diretório %s criado com sucesso", path)
}

func createDefaultFolders() {
	createFolder("media/songs")
}

func main() {

	flags.ProcessFlags()
	varSetup()
	createDefaultFolders()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/api/content"),
		gin.Recovery(),
	)
	router.Static("/static", "./static")
	router.POST("/api/content/set/:providerId", setMediaProviderContent)
	router.GET("/api/content", getMediaProviderContent)
	router.POST("/api/media", saveMedia)

	router.GET("/api/songs", getAllSongs)
	router.GET("/api/songs/content", getSongContent)
	router.GET("/api/songs/folders", getSongsFolderList)
	router.GET("/api/songs/folder", getAllSongsFromFolder)

	router.GET("/controller", viewController)
	router.GET("/live", viewPanel)

	router.GET("/api/discover", discover)

	router.GET("/api/lyrics/letras", getSongLyricsFromLetras)
	router.GET("/api/lyrics/letras/song", getSongLyricFromLetrasByUrl)

	router.GET("/api/bible/books", bible.GetBooksList)
	router.GET("/api/bible/chapter/:version/:book/:chapter", bible.GetChapter)

	log.Print("PRESENTER - Desenvolvido por Alan Telles")
	log.Print("Iniciando serviço...")
	log.Print("Endereço: " + location)
	router.Run("0.0.0.0:" + fmt.Sprint(port))
}
