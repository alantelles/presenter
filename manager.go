package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	MediaPath = "media/"
)

type Category struct {
	Name, DisplayName, Path string
}

var CategorySongs = Category{Name: "songs", DisplayName: "Músicas", Path: "songs"}
var categories = [1]Category{CategorySongs}

func findCategoryByName(name string) (*Category, error) {
	length := len(categories)
	i := 0
	for i < length {
		if categories[i].Name == name {
			return &categories[i], nil
		}
		i += 1
	}
	return nil, errors.New("category not found")
}

func getTextPath(category Category, fileName string) string {
	return MediaPath + category.Path + "/" + fileName + ".txt"
}

func getTextPathNoPrefix(category Category, fileName string) string {
	return MediaPath + category.Path + "/" + fileName
}

func saveTextFile(category Category, fileName string, content string) {
	path := getTextPath(category, fileName)
	f, err := os.Create(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	l, err := f.WriteString(content)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return
	}
	fmt.Println(l, "bytes written successfully")
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func loadMediaList(categoryName string) ([]string, error) {
	files, err := os.ReadDir("media/" + categoryName)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	count := len(files)
	ret := make([]string, count)
	index := 0
	for i := 0; i < count; i++ {
		if files[i].IsDir() {
			continue
		}
		ret[index] = files[i].Name()
		index++
	}
	return ret[:index], nil
}

// TODO: isto precisa ser mais genérico
func loadSongFolders(category string, archivePath string) ([]string, error) {
	files, err := os.ReadDir("media/" + category + "/" + archivePath)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	count := len(files)
	ret := make([]string, count)
	index := 0
	for i := 0; i < count; i++ {
		if !files[i].IsDir() {
			continue
		}
		ret[index] = files[i].Name()
		index++
	}
	return ret[:index], nil
}

func loadSongFile(fileName string) []byte {
	path := getTextPathNoPrefix(CategorySongs, fileName)
	log.Print(path)
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
	return retrieved
}
