package main

import (
	"bytes"
	"compress/zlib"
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
	readPngFile("marsh.png")
}

func getBytesPerPixelForColorType(color_type uint8) uint8 {
	switch color_type {
	case 0:
		return 1
	case 2:
		return 3
	case 4:
		return 2
	case 6:
		return 4
	}
	return 0
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
	var compressed_image_data = []byte{}

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
			// Nothing to handle
		case IDAT_EXPECTED_TYPE:
			handleIdatChunk(file, chunk_length, &compressed_image_data)
		default:
			file.Seek(int64(chunk_length), 1) // Plte will be ignored here as optional for now
		}

		// Read and ignore crc
		_, err = io.ReadFull(file, read_buffer_int)
	}

	// Decompressing
	reader, err := zlib.NewReader(bytes.NewReader(compressed_image_data))
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()

	var pixel_size = getBytesPerPixelForColorType(image_attr.colorType)
	var scanline_size = image_attr.width * int(pixel_size)
	fmt.Printf("%d, %d, %d\n", pixel_size, image_attr.width, scanline_size)

	var unfiltered_canvas = []byte{}

	for i := 0; i < image_attr.height; i++ {
		var filter_type = make([]byte, 1)
		_, err = io.ReadFull(file, filter_type)

		var line_bytes = make([]byte, scanline_size)
		_, err = io.ReadFull(file, filter_type)

		switch filter_type[0] {
		case 0:
			// Already raw bytes
		case 1:
			applySubFilter(line_bytes)
		case 2:
			applyUpFilter(line_bytes)
		case 3:
			applyAverageFilter(line_bytes)
		case 4:
			applyPaethFilter(line_bytes)

		}
	}

	return nil
}

func handleIdatChunk(file *os.File, chunk_length uint32, compressed_image_data *[]byte) {
	var read_buffer_int = make([]byte, chunk_length)
	_, err := io.ReadFull(file, read_buffer_int)
	if err != nil {
		log.Fatal(err)
	}

	*compressed_image_data = append(*compressed_image_data, read_buffer_int...)
}

func handlePlteChunk(file *os.File, chunk_length uint32) {
	// todo: just read it through but dont do nothing for now
	// im not sure i need it
}

func applySubFilter(line_bytes []byte)     {}
func applyUpFilter(line_bytes []byte)      {}
func applyAverageFilter(line_bytes []byte) {}
func applyPaethFilter(line_bytes []byte)   {}
