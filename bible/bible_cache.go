package bible

import (
	"fmt"
	"io"
	"log"
	"os"
)

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
