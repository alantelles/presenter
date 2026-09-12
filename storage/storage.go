// Package storage holds the on-disk storage of songs/lyrics text files:
// categories, listing folders/files and saving/loading text content.
package storage

import (
	"errors"
	"io"
	"log"
	"os"
	"presenter/fsutil"
	"strings"
)

const defaultBasePath = "media/"

var basePath = defaultBasePath

// SetBasePath overrides the base directory under which all media (songs,
// images, etc.) is stored. An empty path resets it to the default ("media/").
// A trailing "/" is added automatically if missing. Call this once at
// startup, before serving any request.
func SetBasePath(path string) {
	if path == "" {
		path = defaultBasePath
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	basePath = path
}

// BasePath returns the currently configured base directory.
func BasePath() string {
	return basePath
}

type Category struct {
	Name, DisplayName, Path string
}

var Songs = Category{Name: "songs", DisplayName: "Músicas", Path: "songs"}

var categories = map[string]Category{
	Songs.Name: Songs,
}

// FindCategoryByName looks up a category by its Name, returning an error if
// it doesn't exist.
func FindCategoryByName(name string) (*Category, error) {
	category, ok := categories[name]
	if !ok {
		return nil, errors.New("category not found")
	}
	return &category, nil
}

func textPath(category Category, fileName string) string {
	return basePath + category.Path + "/" + fileName + ".txt"
}

func textPathNoPrefix(category Category, fileName string) string {
	return basePath + category.Path + "/" + fileName
}

// SaveTextFile writes content as a .txt file under the category's folder.
func SaveTextFile(category Category, fileName string, content string) {
	path := textPath(category, fileName)
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

// List returns the file names directly under a category's folder.
func List(categoryName string) ([]string, error) {
	return listDirEntries(basePath+categoryName, false)
}

// ListFromFolder returns the file names under a category/archive/folder path.
func ListFromFolder(categoryName string, archivePath string, folder string) ([]string, error) {
	return listDirEntries(basePath+categoryName+"/"+archivePath+"/"+folder, false)
}

// ListFolders returns the subfolder names under a category/archive path.
func ListFolders(category string, archivePath string) ([]string, error) {
	return listDirEntries(basePath+category+"/"+archivePath, true)
}

// LoadSongFile reads a song's raw text content by file name.
func LoadSongFile(fileName string) []byte {
	path := textPathNoPrefix(Songs, fileName)
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
