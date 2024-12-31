package camera

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AVFoundation -framework CoreMedia -framework Accelerate
#include "camera.h"

// Global device pointer
static webcam_device_t* _device = NULL;

static inline size_t webcam_format_size() {
	return (size_t)(_device->width) * (uint32_t)(_device->height) * 4;
}

// Forward declaration of the Go callback function
extern void onFrameAvailableGo(void* data, int width, int height);

// Callback function
static void callback(webcam_device_t* device, void* data) {
	onFrameAvailableGo(data, device->width, device->height);
}

static int webcam_open(int deviceId, int width, int height, int framerate) {
	if (_device != NULL) {
		return -1;
	}

	_device = (webcam_device_t*)malloc(sizeof(webcam_device_t));
	if (!_device) {
		return -1;
	}

	memset(_device, 0, sizeof(*_device));
	_device->deviceId  = deviceId;
	_device->width	   = width;
	_device->height	   = height;
	_device->framerate = framerate;

	// already setup
	if (_device->stream) {
		return -1;
	}

	WebcamVideoStream* stream = [[WebcamVideoStream alloc] init];
	stream->_parent				= _device;
	BOOL res					= [stream setupWithID:_device->deviceId rate:_device->framerate width:_device->width height:_device->height];
	if (res == NO) {
		_device->stream = NULL;
		return -1;
	}

	_device->stream	  = stream;
	_device->width	  = stream->_width;
	_device->height	  = stream->_height;
	_device->deviceId  = stream->_id;
	_device->framerate = stream->_framerate;
	_device->callback  = callback;

	return 0;
}

static int webcam_start() {
	if (_device == NULL) {
		return -1;
	}
	if (_device->stream && _device->running == 0) {
		WebcamVideoStream* stream = (WebcamVideoStream*)(_device->stream);
		[stream start];
		_device->running = 1;
		return 0;
	}
	return -1;
}

static int webcam_stop() {
	if (_device && _device->stream && _device->running == 1) {
		WebcamVideoStream* stream = (WebcamVideoStream*)(_device->stream);
		[stream stop];
		_device->running = 0;
		return 0;
	}
	return -1;
}

static void webcam_delete() {
	if (_device == NULL) {
		return;
	}
	if (_device->running == 1) {
		webcam_stop();
	}
	if (_device->stream) {
		WebcamVideoStream* stream = (WebcamVideoStream*)(_device->stream);
		[stream release];
	}

	free(_device);
	_device = NULL;
}

*/
import "C"
import (
	"fmt"
	"unsafe"
)

//export onFrameAvailableGo
func onFrameAvailableGo(data unsafe.Pointer, w, h C.int) {
	go func() {
		size := C.webcam_format_size()
		buf := make([]byte, size)
		C.copyImage((*C.uint8_t)(unsafe.Pointer(&buf[0])), data, size)

		// Convert the buffer to an image.RGBA
		rgba := convertBGRAToRGBA(buf, int(w), int(h))

		select {
		case frameBufferChan <- rgba:
		default:
			// Drop the frame if the channel is full
		}
	}()
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
	return nil
}

func stopCamera() error {
	if C.webcam_stop() != 0 {
		return fmt.Errorf("failed to stop camera")
	}
	return nil
}

func closeCamera() {
	C.webcam_delete()
}
