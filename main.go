package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"presenter/flags"
	"presenter/fsutil"
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

	AppAuthToken = "{{BASIC_AUTH_TOKEN}}"

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
	ContentID string `json:"contentId,omitempty"`
}

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

var port = 8080 // TODO: receive this by running argument
var location string
var usePort = true
var basicAuthUser string
var basicAuthPass string

func CORSMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
	c.Next()
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

func setBasicAuthCredentials() {
	basicAuthUser = flags.GetUsername()
	basicAuthPass = flags.GetPassword()
}

func varSetup() {
	location = flags.GetLocation()
	if location == "" {
		location = setLocation()
	}
	setBasicAuthCredentials()
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

func insertAuthTokenOnContent(content []byte) []byte {
	return bytes.Replace(
		content,
		[]byte(AppAuthToken),
		[]byte(getAuthAsB64()),
		-1,
	)
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
	createFolder("media/songs")
	createFolder("media/images")
	createFolder("media/images/thumbs")
}

func getAuthAsB64() string {
	return base64.StdEncoding.EncodeToString([]byte(basicAuthUser + ":" + basicAuthPass))
}

func AuthMiddleware(c *gin.Context) {
	user, pass, ok := c.Request.BasicAuth()
	if !(user == basicAuthUser && pass == basicAuthPass && ok) {
		c.Writer.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
		c.JSON(401, gin.H{"status": 401, "message": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
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
		CORSMiddleware,
	)
	router.Static("/static", "./static")

	registerMediaRoutes(router)
	registerViewRoutes(router)
	registerMiscRoutes(router)
	registerLyricsRoutes(router)
	registerBibleRoutes(router)

	log.Print("PRESENTER - Desenvolvido por Alan Telles")
	log.Print("Iniciando serviço...")
	log.Print("Endereço: " + location)
	router.Run("0.0.0.0:" + fmt.Sprint(port))
}
