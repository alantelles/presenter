package main

import (
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

func getHtmlPage(path string) ([]byte, error) {
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
	return insertAddressOnContent(retrieved), err
}

func viewPanel(c *gin.Context) {
	read, _ := getHtmlPage("templates/panels/default.html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func viewController(c *gin.Context) {
	page := c.Param("page")
	if page == "" {
		page = "songs"
	}
	read, _ := getHtmlPage("templates/controllers/" + page + ".html")
	c.Data(http.StatusOK, ContentTypeHTML, read)
}

func viewHome(c *gin.Context) {
	read, _ := getHtmlPage("templates/index.html")
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
