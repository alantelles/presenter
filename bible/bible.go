package bible

import (
	"fmt"
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
	result, err := FetchBooksList()
	if err != nil {
		status = http.StatusInternalServerError
		errMsg := strings.ReplaceAll(err.Error(), `"`, `'`)
		result = fmt.Sprintf(`{"error": "Error fetching books list: %s"}`, errMsg)
	}
	c.Data(status, "application/json; charset=utf-8", []byte(result))
}
