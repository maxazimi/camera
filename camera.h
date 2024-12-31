#ifndef CAMERA_H
#define CAMERA_H

#include <string.h>
#include <stdint.h>

struct _webcam_device;
typedef struct _webcam_device webcam_device_t;
typedef void (*webcam_callback_t)(webcam_device_t* device, void* data);

struct _webcam_device {
	int deviceId;
	int width;
	int height;
	int framerate;
	int running;

	void* stream;
	webcam_callback_t callback;
};

static inline void copyImage(uint8_t *dstBuf, void* srcBuf, size_t frame_size) {
    memcpy(dstBuf, srcBuf, frame_size);
}

// Function to convert YUY2 to RGB24
static void YUY2toRGB24(uint8_t *yuy2, uint8_t *rgb, int width, int height) {
    for (int i = 0; i < width * height * 2; i += 4) {
        uint8_t y0 = yuy2[i];
        uint8_t u = yuy2[i + 1];
        uint8_t y1 = yuy2[i + 2];
        uint8_t v = yuy2[i + 3];

        int c0 = y0 - 16;
        int c1 = y1 - 16;
        int d = u - 128;
        int e = v - 128;

        int r0 = (298 * c0 + 409 * e + 128) >> 8;
        int g0 = (298 * c0 - 100 * d - 208 * e + 128) >> 8;
        int b0 = (298 * c0 + 516 * d + 128) >> 8;

        int r1 = (298 * c1 + 409 * e + 128) >> 8;
        int g1 = (298 * c1 - 100 * d - 208 * e + 128) >> 8;
        int b1 = (298 * c1 + 516 * d + 128) >> 8;

        rgb[(i / 2) * 3]     = (uint8_t) (r0 < 0 ? 0 : (r0 > 255 ? 255 : r0));
        rgb[(i / 2) * 3 + 1] = (uint8_t) (g0 < 0 ? 0 : (g0 > 255 ? 255 : g0));
        rgb[(i / 2) * 3 + 2] = (uint8_t) (b0 < 0 ? 0 : (b0 > 255 ? 255 : b0));

        rgb[(i / 2 + 1) * 3]     = (uint8_t) (r1 < 0 ? 0 : (r1 > 255 ? 255 : r1));
        rgb[(i / 2 + 1) * 3 + 1] = (uint8_t) (g1 < 0 ? 0 : (g1 > 255 ? 255 : g1));
        rgb[(i / 2 + 1) * 3 + 2] = (uint8_t) (b1 < 0 ? 0 : (b1 > 255 ? 255 : b1));
    }
}

#if defined(__OBJC__)
#import <Foundation/Foundation.h>
#import <AVFoundation/AVFoundation.h>

@interface WebcamVideoStream : NSObject <AVCaptureVideoDataOutputSampleBufferDelegate> {
	AVCaptureSession* _captureSession;
	AVCaptureDevice* _captureDevice;
	AVCaptureVideoDataOutput* _captureDataOut;
	AVCaptureDeviceInput* _captureDataIn;

@public
	webcam_device_t* _parent;
@public
	int _width;
@public
	int _height;
@public
	int _id;
@public
	int _framerate;
}

- (BOOL)setupWithID:(int)deviceID rate:(int)framerate width:(int)w height:(int)h;
- (void)start;
- (void)stop;

@end

#endif // defined(__OBJC__)
#endif // CAMERA_H
