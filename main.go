package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	// fileName := getRequestedFileNameFromArgs()
	// println(fileName)
	readPngFile("not-a-png.png")
}

func getRequestedFileNameFromArgs() string {
	if len(os.Args) == 1 {
		log.Fatal("No input file was provided.")
	}
	return os.Args[1]
}

func readPngFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Couldn't find the file %s", filename)
	}

	const FILE_SIG_SIZE = 8
	var actualSignature = make([]byte, FILE_SIG_SIZE)
	bytesRead, err := io.ReadAtLeast(file, actualSignature, FILE_SIG_SIZE)
	if err != nil {
		log.Println(err)
		return err
	}
	if bytesRead < FILE_SIG_SIZE {
		log.Fatal("The file signature couldn't be recognized. Size < 8")
	}
	fmt.Print(actualSignature)

	var expectedSignature = [FILE_SIG_SIZE]byte{137, 80, 78, 71, 13, 10, 26, 10}
	if expectedSignature != [8]byte(actualSignature) {
		log.Fatal("Coudldn't recognize the signature. Bytes didn't match")
	}

	return nil
}
