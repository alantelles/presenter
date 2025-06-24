package bible

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"presenter/flags"
	"time"
)

const (
	BibleApiUrl = "https://www.abibliadigital.com.br/api"
)

var client = &http.Client{
	Timeout: 10 * time.Second,
}

func GetTokenBibliaDigital() (string, error) {
	token := flags.GetTokenBibliaDigital()
	error := error(nil)
	if token == "" {
		error = errors.New("erro: nenhum token fornecido para a API Biblia Digital. Por favor, defina a flag --tokenBibliaDigital")
	}
	return token, error
}

func FetchBooksList() (string, error) {
	url := BibleApiUrl + "/books"
	log.Println("Buscando lista de livros em: " + url)
	req, err := GetRequest(url)
	if err != nil {
		return "Erro ao buscar a lista de livros: " + err.Error(), err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "Erro ao buscar a lista de livros: " + err.Error(), err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading response body: " + err.Error(), err
	}
	return string(body), nil
}

func FetchChapter(version, book string, chapter int) (string, error) {
	url := fmt.Sprintf("%s/verses/%s/%s/%d", BibleApiUrl, version, book, chapter)
	log.Println("Buscando capítulo em: " + url)
	req, err := GetRequest(url)
	if err != nil {
		return "Erro ao buscar o capítulo: " + err.Error(), err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "Erro ao buscar o capítulo: " + err.Error(), err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading response body: " + err.Error(), err
	}
	return string(body), nil
}

func GetRequest(url string) (*http.Request, error) {
	log.Println("Fazendo requisição GET para: " + url)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	token, err := GetTokenBibliaDigital()
	if err != nil {
		return nil, errors.New("token da API abibliadigital.com.br não ajustado")
	}
	auth := fmt.Sprintf("Bearer %s", token) // Replace with your actual API token
	req.Header.Add("Authorization", auth)
	return req, nil
}
