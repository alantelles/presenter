package bible

import (
	"fmt"
	"log"
	"presenter/fsutil"
)

func CreateFolder(path string) {
	path = "bible/fetched/" + path
	created, err := fsutil.CreateFolder(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	if created {
		log.Printf("Diretório %s criado com sucesso", path)
	}
}

func SaveTextFile(fileName string, content string) {
	path := "bible/fetched/" + fileName
	if err := fsutil.WriteTextFile(path, content); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(len(content), "bytes written successfully")
}

func LoadTextFile(fileName string) ([]byte, error) {
	path := "bible/content/" + fileName
	log.Print(path)
	content, err := fsutil.ReadTextFile(path)
	if err != nil {
		log.Print(err)
	}
	return content, err
}
