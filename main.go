package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
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
)

type mediaProviderContent struct {
	ProviderId int    `json:"providerId"`
	Content    string `json:"content"`
	IsBinary   bool   `json:"isBinary"`
}

type media struct {
	Category string `json:"category"`
	Content  string `json:"content"`
	Title    string `json:"title"`
	Author   string `json:"author"`
}

type returnBody struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

var provider1 = mediaProviderContent{
	ProviderId: 1, Content: "", IsBinary: false,
}

var address string
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

func varSetup() {
	address = getLocalIp()
	scheme := getScheme(false) // TODO: receive this by running argument
	location = scheme + address + getPortSuffix()
}

func getLocalIp() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Print(err)
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

func main() {
	varSetup()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin.LoggerWithWriter(gin.DefaultWriter, "/api/content"),
		gin.Recovery(),
	)
	router.POST("/api/content/set", setMediaProviderContent)
	router.GET("/api/content", getMediaProviderContent)
	router.POST("/api/media", saveMedia)

	router.GET("/api/songs", getAllSongs)
	router.GET("/api/songs/content", getSongContent)

	router.GET("/controller", viewController)
	router.GET("/live", viewPanel)
	log.Print("PRESENTER - Desenvolvido por Alan Telles")
	log.Print("Iniciando serviço...")
	log.Print("Endereço: " + location)
	router.Run("0.0.0.0:" + fmt.Sprint(port))
}
