// https://github.com/kosua20/sr_webcam/blob/master/src/sr_webcam_mac.m

#import <Foundation/Foundation.h>
#import <AVFoundation/AVFoundation.h>
#import <Accelerate/Accelerate.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import "camera.h"

@implementation WebcamVideoStream

- (id)init {
	self = [super init];
	if (self) {
		_captureDataIn	= nil;
		_captureDataOut = nil;
		_captureDevice	= nil;

		_parent	   = NULL;
		_width	   = 0;
		_height	   = 0;
		_id		   = 0;
		_framerate = 0;
	}
	return self;
}

-  (BOOL)setupWithID:(int)deviceID rate:(int)framerate width:(int)w height:(int)h {

    // List available devices
    NSArray<AVCaptureDevice *> *devices;

    if (@available(macOS 10.15, *)) {
        NSArray<AVCaptureDeviceType> *deviceTypes = @[AVCaptureDeviceTypeBuiltInWideAngleCamera];
        AVCaptureDeviceDiscoverySession *discoSession =
                [AVCaptureDeviceDiscoverySession discoverySessionWithDeviceTypes:deviceTypes mediaType:AVMediaTypeVideo position:AVCaptureDevicePositionUnspecified];
        devices = [discoSession devices];
    } else {
        AVCaptureDevice *device = [AVCaptureDevice defaultDeviceWithMediaType:AVMediaTypeVideo];
        devices = device ? @[device] : @[];
    }

    if ([devices count] == 0 || deviceID < 0) {
        return NO;
    }

    _id = MIN(deviceID, (int)([devices count]) - 1);
    _captureDevice = [devices objectAtIndex:_id];

    // Setup the device.
    NSError *err = nil;
    [_captureDevice lockForConfiguration:&err];
    if (err) {
        return NO;
    }

    // Look for the best format.
    AVCaptureDeviceFormat *bestFormat = nil;
    int wBest = 0;
    int hBest = 0;
    float bestFit = 1e9f;

    for (AVCaptureDeviceFormat *format in [_captureDevice formats]) {
        CMVideoDimensions dimensions = CMVideoFormatDescriptionGetDimensions(format.formatDescription);
        const int wFormat = dimensions.width;
        const int hFormat = dimensions.height;

        // Found the perfect mode.
        if (wFormat == w && hFormat == h) {
            wBest = wFormat;
            hBest = hFormat;
            bestFormat = format;
            break;
        }

        const float dw = (float)(w - wFormat);
        const float dh = (float)(h - hFormat);
        const float fit = sqrtf(dw * dw + dh * dh);
        if (fit < bestFit) {
            bestFit = fit;
            wBest = wFormat;
            hBest = hFormat;
            bestFormat = format;
        }
    }

    // If we found a valid format, set it.
    if (bestFormat) {
        [_captureDevice setActiveFormat:bestFormat];
        _width = wBest;
        _height = hBest;
    }

    // Now try to adjust the framerate.
    if (framerate > 0) {
        CMTime frameDuration = CMTimeMake(1, framerate);
        _captureDevice.activeVideoMinFrameDuration = frameDuration;
        _captureDevice.activeVideoMaxFrameDuration = frameDuration;
    }

    // Configuration done.
    [_captureDevice unlockForConfiguration];

    // Then, the device is ready.
    _captureDataIn = [AVCaptureDeviceInput deviceInputWithDevice:_captureDevice error:nil];
    _captureDataOut = [[AVCaptureVideoDataOutput alloc] init];
    _captureDataOut.alwaysDiscardsLateVideoFrames = YES;

    // We receive data on a secondary thread.
    dispatch_queue_t queue;
    queue = dispatch_queue_create("VideoStream", NULL);
    dispatch_set_target_queue(queue, dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_HIGH, 0));
    [_captureDataOut setSampleBufferDelegate:self queue:queue];

    // Not sure, OF does this but not the Apple sample.
    dispatch_release(queue);

    // Create video settings for the output, corresponding to the current format.
    NSDictionary *settings = @{
        (id)kCVPixelBufferPixelFormatTypeKey : @(kCVPixelFormatType_32BGRA),
        (id)kCVPixelBufferWidthKey : @(_width),
        (id)kCVPixelBufferHeightKey : @(_height)
    };
    [_captureDataOut setVideoSettings:settings];

    _captureSession = [[AVCaptureSession alloc] init];
    _captureSession.sessionPreset = AVCaptureSessionPresetHigh;
    [_captureSession beginConfiguration];
    [_captureSession addInput:_captureDataIn];
    [_captureSession addOutput:_captureDataOut];

    // Setup output (once added), limit acquisition frequency.
    AVCaptureConnection *connection = [_captureDataOut connectionWithMediaType:AVMediaTypeVideo];
    if ([connection isVideoMinFrameDurationSupported] == YES) {
        [connection setVideoMinFrameDuration:CMTimeMake(1, framerate)];
    }
    if ([connection isVideoMaxFrameDurationSupported] == YES) {
        [connection setVideoMaxFrameDuration:CMTimeMake(1, framerate)];
    }

    // Update framerate.
    int framerateConnection = (int)((float)(connection.videoMinFrameDuration.timescale) / (float)(connection.videoMinFrameDuration.value));
    int framerateDevice = (int)((float)(_captureDevice.activeVideoMinFrameDuration.timescale) / (float)(_captureDevice.activeVideoMinFrameDuration.value));
    _framerate = MIN(framerateDevice, framerateConnection);

    [_captureSession commitConfiguration];
    return YES;
}

- (void)start {
	[_captureSession startRunning];
	[_captureDataIn.device lockForConfiguration:nil];

	if ([_captureDataIn.device isFocusModeSupported:AVCaptureFocusModeAutoFocus]) {
		[_captureDataIn.device setFocusMode:AVCaptureFocusModeAutoFocus];
	}
}

- (void)stop {
	if (_captureSession) {
		if (_captureDataOut) {
			// Remove the delegate.
			if (_captureDataOut.sampleBufferDelegate != nil) {
				[_captureDataOut setSampleBufferDelegate:nil queue:NULL];
			}
		}
		// Remove the input and output.
		for(AVCaptureInput* input in _captureSession.inputs) {
			[_captureSession removeInput:input];
		}
		for(AVCaptureOutput* output in _captureSession.outputs) {
			[_captureSession removeOutput:output];
		}
		[_captureSession stopRunning];
	}
}

// Implement the delegate
- (void)captureOutput:(AVCaptureOutput*)captureOutput didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer fromConnection:(AVCaptureConnection*)connection {
    #pragma unused(captureOutput)
    #pragma unused(connection)

	if (!_parent) {
		return;
	}
	@autoreleasepool {
		CVImageBufferRef imgBuffer = CMSampleBufferGetImageBuffer(sampleBuffer);
		CVPixelBufferLockBaseAddress(imgBuffer, 0);

		const int wBuffer = (int)CVPixelBufferGetWidth(imgBuffer);
		const int hBuffer = (int)CVPixelBufferGetHeight(imgBuffer);
		if (wBuffer == _parent->width && hBuffer == _parent->height) {
			unsigned char* baseBuffer = (unsigned char*)CVPixelBufferGetBaseAddress(imgBuffer);
            _parent->callback(_parent, baseBuffer);
		}
		CVPixelBufferUnlockBaseAddress(imgBuffer, kCVPixelBufferLock_ReadOnly);
	}
}

- (void)dealloc {
	if (_captureSession) {
		[self stop];
		[_captureSession release];
		_captureSession = nil;
	}
	if (_captureDataOut) {
		if (_captureDataOut.sampleBufferDelegate != nil) {
			[_captureDataOut setSampleBufferDelegate:nil queue:NULL];
		}
		[_captureDataOut release];
		_captureDataOut = nil;
	}
	if (_parent) {
		_parent = NULL;
	}
	if (_captureDataIn) {
		[_captureDataIn release];
		_captureDataIn = nil;
	}

	if (_captureDevice) {
		[_captureDevice release];
		_captureDevice = nil;
	}
	[super dealloc];
}

@end
