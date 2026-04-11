package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

var IHDR_EXPECTED_TYPE = [...]byte{73, 72, 68, 82}
var PLTE_EXPECTED_TYPE = [...]byte{80, 76, 84, 69}
var IEND_EXPECTED_TYPE = [...]byte{73, 69, 78, 68}
var IDAT_EXPECTED_TYPE = [...]byte{73, 68, 65, 84}

// bytes per pixel, defined by color type
var bpp = 0

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
	runtime.LockOSThread()
	fileName := getRequestedFileNameFromArgs()
	var image_attr, unfiltered_canvas, err = readPngFile(fileName)
	if err != nil {
		panic(err)
	}

	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err := sdl.CreateWindow("PNG Viewer", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, int32(image_attr.width), int32(image_attr.height), sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	var rmask, gmask, bmask, amask uint32 = 0x000000ff, 0x0000ff00, 0x00ff0000, 0xff000000
	if len(unfiltered_canvas) == 0 {
		panic("Coudln't read from image. Unfiltered canvas is of size 0.")
	}
	surface, err := sdl.CreateRGBSurfaceFrom(
		unsafe.Pointer(&unfiltered_canvas[0]),
		int32(image_attr.width),
		int32(image_attr.height), getBitsPerPixel(image_attr),
		int(image_attr.width)*bpp, rmask, gmask, bmask, amask)
	if err != nil {
		panic(err)
	}

	renderer, err := sdl.CreateRenderer(window, 0, 0)
	if err != nil {
		panic(err)
	}

	texture, err := renderer.CreateTextureFromSurface(surface)
	if err != nil {
		panic(err)
	}
	rect := sdl.Rect{X: 0, Y: 0, W: int32(image_attr.width), H: int32(image_attr.height)}

	renderer.Copy(texture, nil, &rect)
	renderer.Present()

	window.UpdateSurface()

	running := true
	for running {
		renderer.Copy(texture, nil, &rect)
		renderer.Present()
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch event_type := event.(type) {
			case *sdl.QuitEvent:
				println("Quit")
				running = false
			case *sdl.KeyboardEvent:
				if event_type.Type == sdl.KEYDOWN {
					if event_type.Keysym.Sym == sdl.K_ESCAPE {
						running = false
					}
				}

			}

		}
		renderer.Clear()

		sdl.Delay(33)
	}
}

func getBitsPerPixel(image_attr ImageAttributes) int {
	switch image_attr.colorType {
	case 2:
		return 24
	case 6:
		return 32
	}

	return 0
}

func getBytesPerPixelForColorType(color_type uint8) int {
	switch color_type {
	case 0:
		return 1 // grayscale
	case 2:
		return 3 // rgb
	case 3:
		println("Color type = 3, palette, exiting. Not supported yet.")
		os.Exit(0)
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

func readPngFile(filename string) (ImageAttributes, []byte, error) {
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
	var read_buffer_int = make([]byte, 4)
	for found_iend_chunk == false {
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
			file.Seek(int64(chunk_length), 1) // Plte be ignored here as optional for now
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

	bpp = getBytesPerPixelForColorType(image_attr.colorType)
	var scanline_size = image_attr.width * int(bpp)
	// fmt.Printf("%d, %d, %d\n", bpp, image_attr.width, scanline_size)

	var unfiltered_canvas = make([]byte, scanline_size*image_attr.height)

	var previous_row []byte = make([]byte, scanline_size)
	var destination_buffer = make([]byte, scanline_size)

	for i := 0; i < image_attr.height; i++ {
		var filter_type = make([]byte, 1)
		_, err = io.ReadFull(reader, filter_type)

		var current_row = make([]byte, scanline_size)
		_, err = io.ReadFull(reader, current_row)

		// Unfiltering
		switch filter_type[0] {
		case 0:
			// Already raw bytes
			copy(destination_buffer, current_row)
		case 1:
			applySubFilter(current_row, destination_buffer)
		case 2:
			applyUpFilter(current_row, previous_row, destination_buffer)
		case 3:
			applyAverageFilter(current_row, previous_row, destination_buffer)
		case 4:
			applyPaethFilter(current_row, previous_row, destination_buffer)
		default:
			log.Fatal("Unrecognized filtering type. Not 0-4")
		}

		copy(previous_row, destination_buffer)

		copy(unfiltered_canvas[i*scanline_size:], destination_buffer)
	}

	return image_attr, unfiltered_canvas, nil
}

func applySubFilter(current_row []byte, unfiltered_buffer []byte) {
	for index, _ := range current_row {
		if index < bpp {
			unfiltered_buffer[index] = current_row[index]
			continue
		}

		unfiltered_buffer[index] = current_row[index] + unfiltered_buffer[index-bpp]
	}
}

func applyUpFilter(current_row []byte, previous_row []byte, unfiltered_buffer []byte) {
	for index, _ := range current_row {
		unfiltered_buffer[index] = current_row[index] + previous_row[index]
	}
}

func applyAverageFilter(current_row []byte, previous_row []byte, unfiltered_buffer []byte) {
	for index, _ := range current_row {
		if index < bpp {
			unfiltered_buffer[index] = current_row[index] + byte(math.Floor(float64(previous_row[index]/2)))
			continue
		}

		unfiltered_buffer[index] = current_row[index] + byte(math.Floor((float64(previous_row[index])+float64(unfiltered_buffer[index-bpp]))/2))
	}
}

func applyPaethFilter(current_row []byte, previous_row []byte, unfiltered_buffer []byte) {
	for index, _ := range current_row {
		if index < bpp {
			unfiltered_buffer[index] = current_row[index] + calculatePaethPredictor(0, int16(previous_row[index]), 0)
			continue
		}

		unfiltered_buffer[index] = current_row[index] + calculatePaethPredictor(int16(unfiltered_buffer[index-bpp]), int16(previous_row[index]), int16(previous_row[index-bpp]))
	}
}

func calculatePaethPredictor(left int16, above int16, up_left int16) byte {
	initial_estimate := left + above - up_left
	pa := math.Abs(float64(initial_estimate) - float64(left))
	pb := math.Abs(float64(initial_estimate) - float64(above))
	pc := math.Abs(float64(initial_estimate) - float64(up_left))

	if pa <= pb && pa <= pc {
		return byte(left)
	} else if pb <= pc {
		return byte(above)
	} else {
		return byte(up_left)
	}
}
