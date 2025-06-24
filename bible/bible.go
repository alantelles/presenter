package bible

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Verse struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type Chapter struct {
	Number int     `json:"number"`
	Verses []Verse `json:"verses"`
	Book   string  `json:"book"`
}

func CORS(c *gin.Context) {
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
	c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")
}

func GetBooksList(c *gin.Context) {
	CORS(c)
	status := http.StatusOK
	cached, err := LoadTextFile("books.json")
	if err == nil && len(cached) > 0 {
		log.Println("Lista de livros carregada do cache.")
		c.Data(status, "application/json; charset=utf-8", cached)
		return
	}
	result, err := FetchBooksList()
	SaveTextFile("books.json", result)
	if err != nil {
		status = http.StatusInternalServerError
		errMsg := strings.ReplaceAll(err.Error(), `"`, `'`)
		result = fmt.Sprintf(`{"error": "Erro ao buscar lista de livros: %s"}`, errMsg)
	}
	c.Data(status, "application/json; charset=utf-8", []byte(result))
}

func GetChapter(c *gin.Context) {
	CORS(c)
	version := c.Param("version")
	book := c.Param("book")
	chapter := c.Param("chapter")

	if version == "" || book == "" || chapter == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetros inválidos"})
		return
	}

	chapterNumber, err := strconv.Atoi(chapter)
	if err != nil || chapterNumber <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Capítulo inválido"})
		return
	}

	result, status, err := FetchChapter(version, book, chapterNumber)
	if err != nil {
		errMsg := strings.ReplaceAll(err.Error(), `"`, `'`)
		c.JSON(status, gin.H{"error": fmt.Sprintf("Erro ao buscar capítulo: %s", errMsg)})
		return
	}
	if status == http.StatusNotFound {
		c.JSON(status, gin.H{"error": fmt.Sprintf("Capítulo %d do livro %s na versão %s não encontrado", chapterNumber, book, version)})
		return
	}
	SaveChapter(book, version, chapterNumber, result)
	c.Data(status, "application/json; charset=utf-8", []byte(result))
}

func SaveChapter(book, version string, chapter int, content string) {
	fileName := fmt.Sprintf("%s_%s_%d.json", version, book, chapter)
	SaveTextFile(fileName, content)
}
