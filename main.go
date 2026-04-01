package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
)

var IHDR_EXPECTED_TYPE = [...]byte{73, 72, 68, 82}

const CHUNK_LENGTH_SIZE = 4
const CHUNK_TYPE_SIZE = 4

type ImageAttributes struct {
	height            int
	width             int
	bitDepth          uint8
	colorType         uint8
	compressionMethod uint8
	filterMethod      uint8
	interlaceMethod   uint8
}

func main() {
	// fileName := getRequestedFileNameFromArgs()
	// println(fileName)
	readPngFile("marsh.png")
}

func getRequestedFileNameFromArgs() string {
	if len(os.Args) == 1 {
		log.Fatal("No input file was provided.")
	}
	return os.Args[1]
}

func readFileSignature(file *os.File) error {
	const FILE_SIG_SIZE = 8
	var actualSignature = make([]byte, FILE_SIG_SIZE)
	bytesRead, err := io.ReadAtLeast(file, actualSignature, FILE_SIG_SIZE)
	if err != nil {
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

func readIhdrChunk(file *os.File) (ImageAttributes, error) {
	var buffer_size = 4
	var read_buffer_int = make([]byte, buffer_size)
	_, err := io.ReadAtLeast(file, read_buffer_int, buffer_size)
	if err != nil {
		return ImageAttributes{}, err
	}
	// var length_uint = binary.BigEndian.Uint32(read_buffer_int)

	_, err = io.ReadAtLeast(file, read_buffer_int, buffer_size)
	if err != nil {
		return ImageAttributes{}, err
	}
	if [4]byte(read_buffer_int) != IHDR_EXPECTED_TYPE {
		log.Fatal("The first chunk wasn't a IHDR chunk. Invalid png file.")
	}

	// Reading size part
	var image_attr = new(ImageAttributes)

	_, err = io.ReadAtLeast(file, read_buffer_int, 4)
	image_attr.width = int(binary.BigEndian.Uint32(read_buffer_int))

	_, err = io.ReadAtLeast(file, read_buffer_int, 4)
	image_attr.height = int(binary.BigEndian.Uint32(read_buffer_int))

	// Reading image properties
	buffer_size = 1
	var read_buffer_byte = make([]byte, buffer_size)
	_, err = io.ReadAtLeast(file, read_buffer_byte, buffer_size)
	image_attr.bitDepth = read_buffer_byte[0]

	_, err = io.ReadAtLeast(file, read_buffer_byte, buffer_size)
	image_attr.colorType = read_buffer_byte[0]

	_, err = io.ReadAtLeast(file, read_buffer_byte, buffer_size)
	image_attr.compressionMethod = read_buffer_byte[0]

	_, err = io.ReadAtLeast(file, read_buffer_byte, buffer_size)
	image_attr.filterMethod = read_buffer_byte[0]

	_, err = io.ReadAtLeast(file, read_buffer_byte, buffer_size)
	image_attr.interlaceMethod = read_buffer_byte[0]

	_, err = io.ReadAtLeast(file, read_buffer_int, 4)
	// todo: Handle the crc ccheck, there's some sample code in the rfc somewhere i think

	return *image_attr, nil
}

func readPngFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Couldn't find the file %s", filename)
	}

	err = readFileSignature(file)
	if err != nil {
		log.Print(err)
	}

	var image_attr, ihdr_err = readIhdrChunk(file)
	if ihdr_err != nil {
		log.Print(ihdr_err)
	}

	fmt.Println(image_attr)

	// Reading Idat chunks
	var found_iend_chunk bool = false
	for found_iend_chunk == false {

	}

	return nil
}
