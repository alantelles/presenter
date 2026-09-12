package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/image/draw"

	"github.com/gin-gonic/gin"
)

func (a *App) getAuthAsB64() string {
	return base64.StdEncoding.EncodeToString([]byte(a.Config.BasicAuthUser + ":" + a.Config.BasicAuthPass))
}

func (a *App) insertAddressOnContent(content []byte) []byte {
	return bytes.Replace(content, []byte(AppLocationToken), []byte(a.Config.Location), -1)
}

func (a *App) insertAuthTokenOnContent(content []byte) []byte {
	return bytes.Replace(content, []byte(AppAuthToken), []byte(a.getAuthAsB64()), -1)
}

func (a *App) getHtmlPage(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		log.Print(err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Print(err)
		}
	}()
	retrieved, err := io.ReadAll(file)
	retrieved = a.insertAddressOnContent(retrieved)
	return a.insertAuthTokenOnContent(retrieved), err
}

func (a *App) viewPanel(c *gin.Context) {
	read, _ := a.getHtmlPage("templates/panels/default.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func (a *App) viewController(c *gin.Context) {
	page := c.Param("page")
	if page == "" {
		page = "songs"
	}
	read, _ := a.getHtmlPage("templates/controllers/" + page + ".html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func (a *App) viewHome(c *gin.Context) {
	read, _ := a.getHtmlPage("templates/index.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func createThumbnail(fileName string) {
	input, _ := os.Open(fileName)
	defer input.Close()
	fmt.Println("input: " + fileName)
	outName := strings.Replace(fileName, "images/", "images/thumbs/", 1)
	outName = strings.Replace(outName, ".JPG", ".png", 1)
	fmt.Println("output: " + outName)
	output, _ := os.Create(outName)
	defer output.Close()
	src, _ := jpeg.Decode(input)
	ratio := 8
	b, h := getThumbnailDimensions(src.Bounds(), ratio)
	dst := image.NewRGBA(image.Rect(0, 0, b, h))
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	png.Encode(output, dst)
}

func getThumbnailDimensions(rect image.Rectangle, ratio int) (int, int) {
	return rect.Max.X / ratio, rect.Max.Y / ratio
}
