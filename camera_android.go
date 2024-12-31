package camera

/*
#cgo LDFLAGS: -landroid -llog -lcamera2ndk -lmediandk
#include "camera_android.c"
*/
import "C"
import (
	"errors"
	"fmt"
	"image"
	"sync"
	"unsafe"
)

var (
	copyImageFn func(buf []byte, width, height int) *image.RGBA
	temp        = image.NewRGBA(image.Rect(0, 0, 640, 480))
)

func init() {
	copyImageFn = rotateImage90
}

//export onImageAvailableGo
func onImageAvailableGo(reader unsafe.Pointer) {
	go func() {
		aImageReader := (*C.AImageReader)(reader)
		var aImage *C.AImage
		C.AImageReader_acquireLatestImage(aImageReader, &aImage)
		if aImage == nil {
			return
		}

		var width, height C.int32_t
		C.AImage_getWidth(aImage, &width)
		C.AImage_getHeight(aImage, &height)

		buf := make([]byte, width*height*4)
		C.copyImage((*C.uint8_t)(unsafe.Pointer(&buf[0])), aImage)
		C.AImage_delete(aImage)

		// Convert the buffer to an image.RGBA
		rgba := copyImageFn(buf, int(width), int(height))

		// Send the frame buffer to the channel
		select {
		case frameBufferChan <- rgba:
		default:
			// Drop the frame if the channel is full
		}
	}()
}

func openCamera(cameraId, width, height int) error {
	if C.openCamera(C.int(cameraId), C.int(width), C.int(height)) != 0 {
		return fmt.Errorf("failed to initialize camera")
	}

	if cameraId != 0 {
		copyImageFn = rotateImageMinus90AndMirror
	}
	return nil
}

func startCamera() error {
	if C.startPreview() != 0 {
		return fmt.Errorf("failed to start camera")
	}
	return nil
}

func stopCamera() error {
	if C.stopPreview() != 0 {
		return fmt.Errorf("failed to stop camera")
	}
	return nil
}

func closeCamera() {
	C.closeCamera()
}

func copyImage(buf []byte, width, height int) *image.RGBA {
	// Create the original RGBA image
	temp := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(temp.Pix, buf)

	return temp
}

func rotateImage90(buf []byte, width, height int) *image.RGBA {
	// Create the original RGBA image
	temp.Rect = image.Rect(0, 0, width, height)
	temp.Stride = width * 4
	temp.Pix = make([]uint8, width*height*4)
	copy(temp.Pix, buf)

	// Create a new RGBA image with swapped width and height for the rotated image
	rotated := image.NewRGBA(image.Rect(0, 0, height, width))

	// Rotate the image by 90 degrees clockwise
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Map the pixel from the original to the rotated image
			rotated.Set(height-1-y, x, temp.At(x, y))
		}
	}

	return rotated
}

func rotateImageMinus90(buf []byte, width, height int) *image.RGBA {
	// Create the original RGBA image
	temp := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(temp.Pix, buf)

	// Create a new RGBA image with swapped width and height for the rotated image
	rotated := image.NewRGBA(image.Rect(0, 0, height, width))

	// Rotate the image by -90 degrees (270 degrees clockwise)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Map the pixel from the original to the rotated image
			rotated.Set(y, width-1-x, temp.At(x, y))
		}
	}

	return rotated
}

func rotateImageMinus90AndMirror(buf []byte, width, height int) *image.RGBA {
	// Create the original RGBA image
	temp := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(temp.Pix, buf)

	// Create a new RGBA image with swapped width and height for the rotated image
	rotated := image.NewRGBA(image.Rect(0, 0, height, width))

	// Rotate the image by -90 degrees (270 degrees clockwise) and mirror it horizontally
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Map the pixel from the original to the rotated and mirrored image
			rotated.Set(height-1-y, width-1-x, temp.At(x, y))
		}
	}

	return rotated
}

var (
	errorStack []error
	mutex      sync.Mutex
)

//export pushError
func pushError(cstr *C.char) {
	goStr := C.GoString(cstr)
	err := errors.New(goStr)

	mutex.Lock()
	defer mutex.Unlock()
	errorStack = append(errorStack, err)
}

// popError retrieves and removes the last error from the stack
func popError() error {
	mutex.Lock()
	defer mutex.Unlock()
	if len(errorStack) == 0 {
		return nil
	}

	err := errorStack[len(errorStack)-1]
	errorStack = errorStack[:len(errorStack)-1]
	return err
}
