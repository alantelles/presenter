package bible

import (
	"fmt"
	"io"
	"log"
	"os"
)

// exists returns whether the given file or directory exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CreateFolder(path string) {
	exists, _ := PathExists(path)
	if exists {
		return
	}
	err := os.MkdirAll(path, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("Diretório %s criado com sucesso", path)
}

func SaveTextFile(fileName string, content string) {
	path := "bible/fetched/" + fileName
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

func LoadTextFile(fileName string) ([]byte, error) {
	path := "bible/fetched/" + fileName
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
	return io.ReadAll(file)
}
