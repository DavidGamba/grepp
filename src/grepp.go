package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func getMimeType(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		println("cannot open", filename)
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Scan()
	b := scanner.Bytes()
	return http.DetectContentType(b)
}

func isText(filename string) bool {
	s := getMimeType(filename)
	return strings.HasPrefix(s, "text/")
}

func getFileList(filename string) bool {
	fInfo, err := os.Stat(filename)
	if err != nil {
		println("cannot stat", filename)
		log.Fatal(err)
	}
	if fInfo.IsDir() {
		file, err := os.Open(filename)
		if err != nil {
			println("cannot open", filename)
			log.Fatal(err)
		}
		defer file.Close()
		dirInfo, err := file.Readdir(-1)
		if err != nil {
			println("cannot open", filename)
			log.Fatal(err)
		}
		for _, file := range dirInfo {
			fmt.Println(file.Name())
			getFileList(file.Name())
		}
	} else {
		fmt.Printf("%s -> %s\n", filename, getMimeType(filename))
	}
	return true
}

func main() {
	log.Printf("args: %s", os.Args[1:])
	for _, v := range os.Args[1:] {
		getFileList(v)
	}
}
