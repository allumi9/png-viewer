package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
)

var IHDR_EXPECTED_TYPE = [...]byte{73, 72, 68, 82}
var PLTE_EXPECTED_TYPE = [...]byte{80, 76, 84, 69}
var IEND_EXPECTED_TYPE = [...]byte{73, 69, 78, 68}
var IDAT_EXPECTED_TYPE = [...]byte{73, 68, 65, 84}

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
	readPngFile("marsh_small.png")
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
	bytesRead, err := io.ReadFull(file, actualSignature)
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
	var read_buffer_int = make([]byte, 4)
	_, err := io.ReadFull(file, read_buffer_int)
	if err != nil {
		return ImageAttributes{}, err
	}
	// var length_uint = binary.BigEndian.Uint32(read_buffer_int)

	_, err = io.ReadFull(file, read_buffer_int)
	if err != nil {
		return ImageAttributes{}, err
	}
	if [4]byte(read_buffer_int) != IHDR_EXPECTED_TYPE {
		log.Fatal("The first chunk wasn't a IHDR chunk. Invalid png file.")
	}

	// Reading size part
	var image_attr = new(ImageAttributes)

	_, err = io.ReadFull(file, read_buffer_int)
	image_attr.width = int(binary.BigEndian.Uint32(read_buffer_int))

	_, err = io.ReadFull(file, read_buffer_int)
	image_attr.height = int(binary.BigEndian.Uint32(read_buffer_int))

	// Reading image properties
	var read_buffer_byte = make([]byte, 1)
	_, err = io.ReadFull(file, read_buffer_byte)
	image_attr.bitDepth = read_buffer_byte[0]

	_, err = io.ReadFull(file, read_buffer_byte)
	image_attr.colorType = read_buffer_byte[0]

	_, err = io.ReadFull(file, read_buffer_byte)
	image_attr.compressionMethod = read_buffer_byte[0]

	_, err = io.ReadFull(file, read_buffer_byte)
	image_attr.filterMethod = read_buffer_byte[0]

	_, err = io.ReadFull(file, read_buffer_byte)
	image_attr.interlaceMethod = read_buffer_byte[0]

	_, err = io.ReadFull(file, read_buffer_int)
	// todo: Handle the crc ccheck, there's some sample code in the rfc somewhere i think

	return *image_attr, nil
}

func readPngFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Couldn't find the file %s", filename)
	}
	defer file.Close()

	err = readFileSignature(file)
	if err != nil {
		log.Print(err)
	}

	var image_attr, ihdr_err = readIhdrChunk(file)
	if ihdr_err != nil {
		log.Print(ihdr_err)
	}

	fmt.Println(image_attr)
	var image_data_full = []byte{}

	// Reading all other chunks
	var found_iend_chunk = false
	for found_iend_chunk == false {
		var read_buffer_int = make([]byte, 4)
		_, err = io.ReadFull(file, read_buffer_int)
		var chunk_length = binary.BigEndian.Uint32(read_buffer_int)

		_, err = io.ReadFull(file, read_buffer_int)

		switch [4]byte(read_buffer_int) {
		case IEND_EXPECTED_TYPE:
			found_iend_chunk = true
			handleIendChunk(file, chunk_length)
		case PLTE_EXPECTED_TYPE:
		case IDAT_EXPECTED_TYPE:
			handleIdatChunk(file, chunk_length, &image_data_full)
		}
	}

	fmt.Print(image_data_full)

	return nil
}

func handleIdatChunk(file *os.File, chunk_length uint32, image_data *[]byte) {
	var read_buffer_int = make([]byte, chunk_length)
	_, err := io.ReadFull(file, read_buffer_int)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < int(chunk_length); i++ {
		(*image_data)[len(*image_data)] = read_buffer_int[i]
	}
}

func handleIendChunk(file *os.File, chunk_length uint32) {
	// Maybe finalize the image or something here
}
func handlePlteChunk(file *os.File, chunk_length uint32) {
	// todo: just read it through but dont do nothing for now
	// im not sure i need it
}
