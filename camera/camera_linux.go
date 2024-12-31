//go:build linux && !android && !ios

package camera

/*
#cgo LDFLAGS: -lv4l2
#include "camera_linux.c"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

var (
	running = false
)

func startCapture() {
	if running {
		return
	}

	running = true
	go func() {
		for running {
			C.capture_frame()
		}
		running = false
	}()
}

//export onFrameAvailableGo
func onFrameAvailableGo(data unsafe.Pointer, width, height, bytesPerPixel C.int) {
	frameSize := int(width) * int(height) * int(bytesPerPixel)
	buf := make([]byte, frameSize)
	C.copyImage((*C.uint8_t)(unsafe.Pointer(&buf[0])), data, C.size_t(frameSize))

	// Convert the buffer to an image.RGBA
	rgba := convertAndMirrorRGB24ToRGBA(buf, int(width), int(height))

	select {
	case frameBufferChan <- rgba:
	default:
		// Drop the frame if the channel is full
	}
}

func openCamera(id, width, height int) error {
	if C.webcam_open(C.int(id), C.int(width), C.int(height), 30) != 0 {
		return fmt.Errorf("failed to initialize camera")
	}
	return nil
}

func startCamera() error {
	if C.webcam_start() != 0 {
		return fmt.Errorf("failed to start camera")
	}

	startCapture()
	return nil
}

func stopCamera() error {
	running = false
	if C.webcam_stop() != 0 {
		return fmt.Errorf("failed to stop camera")
	}
	return nil
}

func closeCamera() {
	C.webcam_close()
}
