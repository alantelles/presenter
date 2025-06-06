package main

import (
	"io"
	"net/http"
	"regexp"
	"strings"
)

const (
	LetrasSite = "https://www.letras.mus.br"
)

func buildLetterIndexUrl(letter string) string {
	return LetrasSite + "/letra/" + strings.ToUpper(letter) + "/artists_ajax.html"
}

func buildArtistSongListUrl(path string) string {
	return LetrasSite + "/" + path
}

func buildSongLyricUrl(artistPath string, songIdPath string) string {
	return LetrasSite + "/" + artistPath + "/" + songIdPath
}

func fetchLetterIndex(letter string) string {
	resp, err := http.Get(buildLetterIndexUrl(letter))
	if err != nil {
		return "Houve um erro ao buscar a lista de artistas: " + err.Error()
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Houve um erro ao ler a lista de artistas: " + err.Error()
	}
	return string(body)
}

func findArtistPathInIndex(artist string, content string) string {
	pattern, _ := regexp.Compile("(?i)<a href=\"\\/([a-z0-9-]+)\\/\">" + artist + "<\\/a>")
	matches := pattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func getArtistPath(artist string) string {
	firstLetter := strings.ToUpper(string(artist[0]))
	content := fetchLetterIndex(firstLetter)
	return findArtistPathInIndex(artist, content)
}

func fetchArtistsSongs(path string) string {
	resp, err := http.Get(buildArtistSongListUrl(path))
	if err != nil {
		return "Houve um erro ao buscar as músicas do artista: " + err.Error()
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Houve um erro ao ler as músicas do artista: " + err.Error()
	}
	return string(body)
}

func findSongLyricsId(artistPath string, content string, songName string) string {
	noBreakLines := strings.ReplaceAll(content, "\n", "")
	strPattern := `(?i)href="\/` + artistPath + `\/([a-z0-9-]+)\/"(?: title="` + strings.ToLower(songName) + `")?>\s*<span>` + strings.ToLower(songName) + `<\/span>`
	pattern, _ := regexp.Compile(strPattern)
	matches := pattern.FindStringSubmatch(noBreakLines)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func getSongId(artistPath string, songName string) string {
	content := fetchArtistsSongs(artistPath)
	songId := findSongLyricsId(artistPath, content, songName)
	return songId
}

func fetchSongLyrics(artistPath string, songId string) string {
	return fetchSongLyricsByUrl(buildSongLyricUrl(artistPath, songId))
}

func fetchSongLyricsByUrl(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return "Houve um erro ao buscar a letra da música: " + err.Error()
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Houve um erro ao ler a letra da música: " + err.Error()
	}
	return string(body)
}

func extractRawSongLyrics(content string) string {
	if len(content) == 0 {
		return ""
	}
	startStr := `<div class="lyric-original">`
	start := strings.Index(content, startStr)
	content = content[start+len(startStr):]
	end := strings.Index(content, "</div>")
	content = content[0:end]
	return content
}

func processSongLyricsToPresent(rawLyrics string) string {
	rawLyrics = strings.ReplaceAll(rawLyrics, "\n", "")
	rawLyrics = strings.ReplaceAll(rawLyrics, "<p>", "")
	rawLyrics = strings.ReplaceAll(rawLyrics, "<br/>", "\n")
	rawLyrics = strings.ReplaceAll(rawLyrics, "</p>", "\n\n")
	rawLyrics = strings.TrimSpace(rawLyrics)
	return rawLyrics
}

func getSongNameAndArtistName(content string) (string, string) {
	strSongPattern := `"track_name":"([^"]+)"`
	songPattern, _ := regexp.Compile(strSongPattern)
	songMatches := songPattern.FindStringSubmatch(content)
	songName := songMatches[1]
	strArtistPattern := `"artist_name":"([^"]+)"`
	artistPattern, _ := regexp.Compile(strArtistPattern)
	artistMatches := artistPattern.FindStringSubmatch(content)
	artistName := artistMatches[1]
	return songName, artistName
}

func getProcessedSongLyrics(artistPath string, songId string) string {
	content := fetchSongLyrics(artistPath, songId)
	rawLyrics := extractRawSongLyrics(content)
	if rawLyrics == "" {
		return ""
	}
	songName, artistName := getSongNameAndArtistName(content)
	processedLyrics := songName + "\n" + artistName + "\n\n" + processSongLyricsToPresent(rawLyrics)
	return processedLyrics
}

func getSongLyrics(artist string, songName string) string {
	songName = strings.ToLower(songName)
	artist = strings.ToLower(artist)
	artistPath := getArtistPath(artist)
	if artistPath == "" {
		return ""
	}
	songId := getSongId(artistPath, songName)
	if songId == "" {
		return ""
	}
	return getProcessedSongLyrics(artistPath, songId)
}

func getSongLyricsByUrl(url string) string {
	content := fetchSongLyricsByUrl(url)
	rawLyrics := extractRawSongLyrics(content)
	songName, artistName := getSongNameAndArtistName(content)
	processedLyrics := songName + "\n" + artistName + "\n\n" + processSongLyricsToPresent(rawLyrics)
	return processedLyrics
}
