package bible

import (
	"fmt"
	"log"
	"net/http"
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
