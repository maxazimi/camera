package camera

import (
	"fmt"
	"image"
	"image/color"
)

var (
	frameBufferChan chan *image.RGBA
	opened          = false
	stopped         = true
)

func Open(id, width, height int) error {
	if opened {
		return fmt.Errorf("camera already opened")
	}

	if err := openCamera(id, width, height); err != nil {
		return err
	}

	frameBufferChan = make(chan *image.RGBA, 1)
	opened = true
	return nil
}

func StartPreview() error {
	if !opened {
		return fmt.Errorf("camera not initialized")
	}
	if !stopped {
		return fmt.Errorf("camera already running")
	}

	if err := startCamera(); err != nil {
		return err
	}

	stopped = false
	return nil
}

func StopPreview() error {
	if !opened {
		return fmt.Errorf("camera not initialized")
	}
	if stopped {
		return fmt.Errorf("camera already stopped")
	}

	if err := stopCamera(); err != nil {
		return err
	}

	stopped = true
	return nil
}

func Close() {
	if !opened {
		return
	}
	if !stopped {
		_ = StopPreview()
	}

	opened = false
	closeCamera()
	close(frameBufferChan)
}

func GetCameraFrameChan() <-chan *image.RGBA {
	return frameBufferChan
}

func convertRGB24ToRGBA(rgbBuffer []byte, width, height int) *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			rgba.Set(x, y, color.RGBA{
				R: rgbBuffer[i],
				G: rgbBuffer[i+1],
				B: rgbBuffer[i+2],
				A: 255,
			})
		}
	}
	return rgba
}

func convertBGRAToRGBA(rgbBuffer []byte, width, height int) *image.RGBA {
	rgbaImage := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Calculate the index for the mirrored column
			mirroredX := width - 1 - x
			rgbaIndex := (y*width + x) * 4
			mirroredIndex := (y*width + mirroredX) * 4

			b := rgbBuffer[rgbaIndex]
			g := rgbBuffer[rgbaIndex+1]
			r := rgbBuffer[rgbaIndex+2]
			a := rgbBuffer[rgbaIndex+3]

			rgbaImage.Pix[mirroredIndex] = r
			rgbaImage.Pix[mirroredIndex+1] = g
			rgbaImage.Pix[mirroredIndex+2] = b
			rgbaImage.Pix[mirroredIndex+3] = a
		}
	}

	return rgbaImage
}

// convertAndMirrorRGB24ToRGBA convert and mirror RGB24 to RGBA
func convertAndMirrorRGB24ToRGBA(rgbBuffer []byte, width, height int) *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			originalIndex := (y*width + x) * 3
			mirroredX := width - 1 - x
			rgba.Set(mirroredX, y, color.RGBA{
				R: rgbBuffer[originalIndex],
				G: rgbBuffer[originalIndex+1],
				B: rgbBuffer[originalIndex+2],
				A: 255,
			})
		}
	}
	return rgba
}
