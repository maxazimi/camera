package camera

/*
#cgo LDFLAGS: -landroid -llog -lcamera2ndk -lmediandk

#include <camera/NdkCameraManager.h>
#include <camera/NdkCameraDevice.h>
#include <camera/NdkCameraCaptureSession.h>
#include <media/NdkImageReader.h>
#include <android/log.h>
#include <jni.h>

#define LOG_TAG "NDKCamera"
#define LOGI(...) __android_log_print(ANDROID_LOG_INFO, LOG_TAG, __VA_ARGS__)
#define LOGD(...) __android_log_print(ANDROID_LOG_DEBUG, LOG_TAG, __VA_ARGS__)
#define LOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)

// Helper function for YUV_420 to RGB conversion. Courtesy of Tensorflow
// ImageClassifier Sample:
// https://github.com/tensorflow/tensorflow/blob/master/tensorflow/examples/android/jni/yuv2rgb.cc
// The difference is that here we have to swap UV plane when calling it.
#ifndef MAX
#define MAX(a, b)           \
  ({                        \
    __typeof__(a) _a = (a); \
    __typeof__(b) _b = (b); \
    _a > _b ? _a : _b;      \
  })
#define MIN(a, b)           \
  ({                        \
    __typeof__(a) _a = (a); \
    __typeof__(b) _b = (b); \
    _a < _b ? _a : _b;      \
  })
#endif

// This value is 2 ^ 18 - 1, and is used to clamp the RGB values before their ranges
// are normalized to eight bits.
static const int kMaxChannelValue = 262143;

static inline uint32_t YUV2RGB(int nY, int nU, int nV) {
  nY -= 16;
  nU -= 128;
  nV -= 128;
  if (nY < 0) nY = 0;

  // This is the floating point equivalent. We do the conversion in integer
  // because some Android devices do not have floating point in hardware.
  // nR = (int)(1.164 * nY + 1.596 * nV);
  // nG = (int)(1.164 * nY - 0.813 * nV - 0.391 * nU);
  // nB = (int)(1.164 * nY + 2.018 * nU);

  int nR = (int)(1192 * nY + 1634 * nV);
  int nG = (int)(1192 * nY - 833 * nV - 400 * nU);
  int nB = (int)(1192 * nY + 2066 * nU);

  nR = MIN(kMaxChannelValue, MAX(0, nR));
  nG = MIN(kMaxChannelValue, MAX(0, nG));
  nB = MIN(kMaxChannelValue, MAX(0, nB));

  nR = (nR >> 10) & 0xff;
  nG = (nG >> 10) & 0xff;
  nB = (nB >> 10) & 0xff;

  return 0xff000000 | (nB << 16) | (nG << 8) | nR;
}

// Converting yuv to RGB
// No rotation: (x,y) --> (x, y)
// Refer to: https://mathbits.com/MathBits/TISection/Geometry/Transformations2.htm
static void copyImage(uint8_t *buffer, AImage *image) {
  AImageCropRect srcRect;
  AImage_getCropRect(image, &srcRect);

  int32_t yStride, uvStride;
  uint8_t *yPixel, *uPixel, *vPixel;
  int32_t yLen, uLen, vLen;
  AImage_getPlaneRowStride(image, 0, &yStride);
  AImage_getPlaneRowStride(image, 1, &uvStride);
  AImage_getPlaneData(image, 0, &yPixel, &yLen);
  AImage_getPlaneData(image, 1, &uPixel, &uLen);
  AImage_getPlaneData(image, 2, &vPixel, &vLen);
  int32_t uvPixelStride;
  AImage_getPlanePixelStride(image, 1, &uvPixelStride);

  int32_t height = srcRect.bottom - srcRect.top;
  int32_t width = srcRect.right - srcRect.left;
  uint32_t *out = (uint32_t *)buffer;
  int32_t stride = width;

  for (int32_t y = 0; y < height; y++) {
    const uint8_t *pY = yPixel + yStride * (y + srcRect.top) + srcRect.left;

    int32_t uv_row_start = uvStride * ((y + srcRect.top) >> 1);
    const uint8_t *pU = uPixel + uv_row_start + (srcRect.left >> 1);
    const uint8_t *pV = vPixel + uv_row_start + (srcRect.left >> 1);

    for (int32_t x = 0; x < width; x++) {
      const int32_t uv_offset = (x >> 1) * uvPixelStride;
      out[x] = YUV2RGB(pY[x], pU[uv_offset], pV[uv_offset]);
    }
    out += stride;
  }
}

// Forward declaration of the Go function
extern void onImageAvailableGo(void* reader);

static void onImageAvailable(void* context, AImageReader* reader) {
	onImageAvailableGo(reader);
}

static ACameraManager* cameraManager = NULL;
static ACameraDevice* cameraDevice = NULL;
static ACameraOutputTarget* textureTarget = NULL;
static ACaptureRequest* request = NULL;
static ACameraCaptureSession* textureSession = NULL;
static ACaptureSessionOutput* textureOutput = NULL;
static ANativeWindow* imageWindow = NULL;
static ACameraOutputTarget* imageTarget = NULL;
static AImageReader* imageReader = NULL;
static ACaptureSessionOutput* imageOutput = NULL;
static ACaptureSessionOutput* output = NULL;
static ACaptureSessionOutputContainer* outputs = NULL;

// Device listeners
static void onDisconnected(void* context, ACameraDevice* device) {
	LOGD("onDisconnected");
}

static void onError(void* context, ACameraDevice* device, int error) {
	LOGD("error %d", error);
}

static ACameraDevice_stateCallbacks cameraDeviceCallbacks = {
	.context = NULL,
	.onDisconnected = onDisconnected,
	.onError = onError,
};

// Session state callbacks
static void onSessionActive(void* context, ACameraCaptureSession *session) {
	LOGD("onSessionActive()");
}

static void onSessionReady(void* context, ACameraCaptureSession *session) {
	LOGD("onSessionReady()");
}

static void onSessionClosed(void* context, ACameraCaptureSession *session) {
	LOGD("onSessionClosed()");
}

static ACameraCaptureSession_stateCallbacks sessionStateCallbacks = {
	.context = NULL,
	.onActive = onSessionActive,
	.onReady = onSessionReady,
	.onClosed = onSessionClosed
};

static AImageReader* createImageReader(int width, int height) {
    AImageReader* reader = NULL;
    media_status_t status = AImageReader_new(width, height, AIMAGE_FORMAT_YUV_420_888, 4, &reader);

    if (status != AMEDIA_OK) {
		LOGE("createImageReader() failed");
	}

    AImageReader_ImageListener listener = {
		.context = NULL,
		.onImageAvailable = onImageAvailable,
    };

    AImageReader_setImageListener(reader, &listener);
    return reader;
}

static ANativeWindow* createSurface(AImageReader* reader) {
    ANativeWindow *nativeWindow;
    AImageReader_getWindow(reader, &nativeWindow);
    return nativeWindow;
}

// Capture callbacks
static void onCaptureFailed(void* context, ACameraCaptureSession* session, ACaptureRequest* request, ACameraCaptureFailure* failure) {
	LOGE("onCaptureFailed ");
}

static void onCaptureSequenceCompleted(void* context, ACameraCaptureSession* session, int sequenceId, int64_t frameNumber) {
}

static void onCaptureSequenceAborted(void* context, ACameraCaptureSession* session, int sequenceId) {
}

static void onCaptureCompleted(void* context, ACameraCaptureSession* session, ACaptureRequest* request,
								const ACameraMetadata* result) {
	LOGD("Capture completed");
}

static ACameraCaptureSession_captureCallbacks captureCallbacks = {
	.context = NULL,
	.onCaptureStarted = NULL,
	.onCaptureProgressed = NULL,
	.onCaptureCompleted = onCaptureCompleted,
	.onCaptureFailed = onCaptureFailed,
	.onCaptureSequenceCompleted = onCaptureSequenceCompleted,
	.onCaptureSequenceAborted = onCaptureSequenceAborted,
	.onCaptureBufferLost = NULL,
};

static const char* getBackFacingCamId(ACameraManager *cameraManager) {
    ACameraIdList *cameraIds = NULL;
    ACameraManager_getCameraIdList(cameraManager, &cameraIds);

    const char* backId = NULL;
    LOGD("found camera count %d", cameraIds->numCameras);

    for (int i = 0; i < cameraIds->numCameras; i++) {
        const char *id = cameraIds->cameraIds[i];
        ACameraMetadata *metadataObj;
        ACameraManager_getCameraCharacteristics(cameraManager, id, &metadataObj);

        ACameraMetadata_const_entry lensInfo = {0};
        ACameraMetadata_getConstEntry(metadataObj, ACAMERA_LENS_FACING, &lensInfo);

        acamera_metadata_enum_android_lens_facing_t facing =
			(acamera_metadata_enum_android_lens_facing_t)(lensInfo.data.u8[0]);

        // Found a back-facing camera?
        if (facing == ACAMERA_LENS_FACING_BACK) {
            backId = id;
            break;
        }
    }

    ACameraManager_deleteCameraIdList(cameraIds);
    return backId;
}

static int openCamera(int cameraId, int width, int height) {
	cameraManager = ACameraManager_create();
	if (cameraManager == NULL) {
		LOGE("initCamera() failed");
		return -1;
	}

	const char* id = getBackFacingCamId(cameraManager);
	if (id == NULL) {
		LOGE("initCamera() failed");
		return -1;
	}

	char buf[10];
	sprintf(buf, "%s", id);
	if (ACameraManager_openCamera(cameraManager, buf, &cameraDeviceCallbacks, &cameraDevice) != ACAMERA_OK) {
		LOGE("Failed to open camera");
		return -1;
	}

	// Prepare request for texture target
	ACameraDevice_createCaptureRequest(cameraDevice, TEMPLATE_PREVIEW, &request);

	// Prepare outputs for session
	ACaptureSessionOutputContainer_create(&outputs);

	imageReader = createImageReader(width, height);
	imageWindow = createSurface(imageReader);
	ANativeWindow_acquire(imageWindow);
	ACameraOutputTarget_create(imageWindow, &imageTarget);
	ACaptureRequest_addTarget(request, imageTarget);

	// Set auto-focus mode
	uint8_t afMode = ACAMERA_CONTROL_AF_MODE_CONTINUOUS_PICTURE;
	ACaptureRequest_setEntry_u8(request, ACAMERA_CONTROL_AF_MODE, 1, &afMode);

	ACaptureSessionOutput_create(imageWindow, &imageOutput);
	ACaptureSessionOutputContainer_add(outputs, imageOutput);

	// Create the session
	ACameraDevice_createCaptureSession(cameraDevice, outputs, &sessionStateCallbacks, &textureSession);

	return 0;
}

static void closeCamera() {
	if (!cameraManager) {
		return;
	}

	// Stop recording to SurfaceTexture and do some cleanup
	ACameraCaptureSession_stopRepeating(textureSession);
	ACameraCaptureSession_close(textureSession);
	ACaptureSessionOutputContainer_free(outputs);
	ACaptureSessionOutput_free(output);

	ACameraDevice_close(cameraDevice);
	ACameraManager_delete(cameraManager);
	cameraManager = NULL;

	AImageReader_delete(imageReader);
	imageReader = NULL;

	// Capture request for SurfaceTexture
	ANativeWindow_release(imageWindow);
	ACaptureRequest_free(request);

	LOGD("Camera closed");
}

static int startPreview() {
	// Start capturing continuously
	if (ACameraCaptureSession_setRepeatingRequest(textureSession, &captureCallbacks, 1, &request, NULL) == ACAMERA_OK) {
		LOGI("Preview started");
		return 0;
	}
	LOGE("Failed to start preview");
	return -1;
}

static int stopPreview() {
	if (ACameraCaptureSession_stopRepeating(textureSession) == ACAMERA_OK) {
		LOGI("Preview stopped");
		return 0;
	}
	LOGE("Failed to stop preview");
	return -1;
}

*/
import "C"
import (
	"fmt"
	"image"
	"unsafe"
)

var (
	temp = image.NewRGBA(image.Rect(0, 0, 640, 480))
)

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
		rgba := rotateImage90(buf, int(width), int(height))

		// Send the frame buffer to the channel
		select {
		case frameBufferChan <- rgba:
		default:
			// Drop the frame if the channel is full
		}
	}()
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

func openCamera(cameraId, width, height int) error {
	if C.openCamera(C.int(cameraId), C.int(width), C.int(height)) != 0 {
		return fmt.Errorf("failed to initialize camera")
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
