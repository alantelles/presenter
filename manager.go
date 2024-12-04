package main

import (
	"errors"
	"fmt"
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

func SaveTextFile(category Category, fileName string, content string) {
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

func loadMediaList(categoryName string) []string {
	files, err := os.ReadDir("media/" + categoryName)
	if err != nil {
		log.Fatal(err)
	}

	ret := make([]string, len(files))
	for i, file := range files {
		ret[i] = file.Name()
	}
	return ret
}
