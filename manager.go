package main

import (
	"errors"
	"io"
	"log"
	"os"
	"presenter/fsutil"
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
	if err := fsutil.WriteTextFile(path, content); err != nil {
		log.Print(err)
		return
	}
	log.Printf("%d bytes written successfully to %s", len(content), path)
}

// listDirEntries lists the names directly under path, keeping only entries
// whose IsDir() matches wantDirs (i.e. only subfolders, or only files).
func listDirEntries(path string, wantDirs bool) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() != wantDirs {
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

func loadMediaList(categoryName string) ([]string, error) {
	return listDirEntries(MediaPath+categoryName, false)
}

func loadMediaListFromFolder(categoryName string, archivePath string, folder string) ([]string, error) {
	return listDirEntries(MediaPath+categoryName+"/"+archivePath+"/"+folder, false)
}

func loadSongFolders(category string, archivePath string) ([]string, error) {
	return listDirEntries(MediaPath+category+"/"+archivePath, true)
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
