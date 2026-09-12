package main

import (
	"fmt"
	"log"
	"net"
	"presenter/flags"
	"presenter/providers"
	"strings"

	"github.com/gin-gonic/gin"
)

// Config holds the runtime configuration resolved once at startup, replacing
// the package-level vars (port, location, usePort, basicAuthUser/Pass) that
// used to be mutated by main()/varSetup() and read from anywhere.
type Config struct {
	Port          int
	UsePort       bool
	Location      string
	BasicAuthUser string
	BasicAuthPass string
}

// App bundles the resolved Config with the in-memory provider state, and is
// what handlers now depend on instead of package-level globals.
type App struct {
	Config    Config
	Providers *providers.Store
}

func buildScheme(secure bool) string {
	if secure {
		return "https://"
	}
	return "http://"
}

func buildPortSuffix(port int, usePort bool) string {
	if usePort {
		return ":" + fmt.Sprint(port)
	}
	return ""
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

func detectLocation(port int, usePort bool) string {
	address := getLocalIp()
	scheme := buildScheme(false) // TODO: receive this by running argument
	return scheme + address + buildPortSuffix(port, usePort)
}

// NewConfig resolves the app configuration from flags, falling back to
// auto-detecting the local address when none was given.
func NewConfig() Config {
	port := 8080 // TODO: receive this by running argument
	usePort := true

	location := flags.GetLocation()
	if location == "" {
		location = detectLocation(port, usePort)
	}

	return Config{
		Port:          port,
		UsePort:       usePort,
		Location:      location,
		BasicAuthUser: flags.GetUsername(),
		BasicAuthPass: flags.GetPassword(),
	}
}

// NewApp builds an App with the default set of provider channels.
func NewApp(cfg Config) *App {
	return &App{
		Config:    cfg,
		Providers: providers.NewStore(),
	}
}

func (a *App) AuthMiddleware(c *gin.Context) {
	user, pass, ok := c.Request.BasicAuth()
	if !(user == a.Config.BasicAuthUser && pass == a.Config.BasicAuthPass && ok) {
		c.Writer.Header().Set("WWW-Authenticate", "Basic realm=Restricted")
		c.JSON(401, gin.H{"status": 401, "message": "Unauthorized"})
		c.Abort()
		return
	}
	c.Next()
}
