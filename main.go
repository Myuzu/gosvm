package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"gocv.io/x/gocv"
)

const (
	frameBufferSize    = 110 // Buffer size to hold past frames
	defaultBlendOffset = 3   // Default number of frames to delay when blending
	maxCameraRetries   = 5   // Maximum number for exponential read retries
	minBlendOffset     = 1   // Minimum frame delay
	maxBlendOffset     = 109 // Maximum frame delay
)

var HUDColor = color.RGBA{0, 255, 0, 0}

func main() {
	// defer profile.Start(profile.MemProfile).Stop()

	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\n\tgosvm [camera ID]")
		return
	}

	// Parse args
	deviceID, _ := strconv.Atoi(os.Args[1])

	// Try to open video capture device
	webcam, err := gocv.VideoCaptureDevice(int(deviceID))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	// Open display window
	window := gocv.NewWindow("Motion Extraction")
	defer window.Close()

	// Initialize CircularFrameBuffer to hold past frames
	frameBuffer := NewCircularFrameBuffer(frameBufferSize, defaultBlendOffset)

	// Initialize FPS calculator
	fpsCalculator := NewFPSCalculator()

	fmt.Printf("Start reading from camera device: %v\n", deviceID)

	// Main program loop
	for {
		// Initialize and read current frame from device
		currentFrame := gocv.NewMat()

		if !ReadWebcamWithRetry(webcam, &currentFrame, maxCameraRetries) {
			log.Fatal("Failed to read from webcam after multiple attempts")
			break
		}

		if currentFrame.Empty() {
			currentFrame.Close()
			break
		}

		// Close the Mat at the current buffer position to avoid memory leaks
		// if frameBuffer[currentIndex].Empty() == false {
		// 	frameBuffer[currentIndex].Close()
		// }

		// Denoise currentFrame
		// gocv.FastNlMeansDenoisingColoredWithParams(currentFrame, &currentFrame, 28.0, 12.0, 12, 7)

		// Enqueue the current frame in the buffer
		if !frameBuffer.Enqueue(currentFrame.Clone()) {
			log.Fatal("CircularFrameBuffer is full")
		}

		currentFrame.Close()

		blendFrame := frameBuffer.CalcBlendFrame()

		if !blendFrame.Empty() {
			// Calculate base frame
			baseFrame := frameBuffer.BaseFrame()

			// Create a half-transparent inverted version of the blending frame
			halfTransparentFrame := gocv.NewMat()

			// Invert current frame
			gocv.BitwiseNot(blendFrame, &halfTransparentFrame)
			gocv.AddWeighted(halfTransparentFrame, 0.5, baseFrame, 0.0, 0, &halfTransparentFrame)

			// Blend the current frame with the delayed frame
			blendedFrame := gocv.NewMat()

			blendFrames(baseFrame, halfTransparentFrame, &blendedFrame, 0.4)

			// Apply emobss effect
			// applyEmbossEffect(blendedFrame, &blendedFrame)

			// Calculate FPS
			if fpsCalculator != nil {
				fps := fpsCalculator.calculateFPS()
				gocv.PutText(&blendedFrame,
					fmt.Sprintf("FPS: %.2f, Delay: %d (A/D keys to inc/dec), Freeze: %t",
						fps,
						frameBuffer.blendOffset,
						frameBuffer.IsFrozen()),
					image.Pt(10, 40),
					gocv.FontHersheyPlain, 1.9, HUDColor, 2)
			}

			// Display the resulting frame in the window
			window.IMShow(blendedFrame)

			// Close Mats manualy
			blendedFrame.Close()
			halfTransparentFrame.Close()
		}

		// Move to the next index in the circular buffer
		// frameBuffer.Dequeue()

		// Handle user input to change the blend offset
		key := window.WaitKey(10)
		if key == 27 || key == 113 { // ESC or Q key to exit
			break
		} else if key == 97 { // "A" key decreses blendOffset
			frameBuffer.DecBlendOffset()
		} else if key == 100 { // "D" key increses blendOffset
			frameBuffer.IncBlendOffset()
		} else if key == 32 { // Spacebar key to togle freeze frame
			frameBuffer.ToggleFreezeFrame()
		}

		time.Sleep(30 * time.Millisecond)
	}
}

// ReadWithRetry attempts to read from the webcam with exponential backoff retries
func ReadWebcamWithRetry(webcam *gocv.VideoCapture, img *gocv.Mat, maxRetries int) bool {
	var delay time.Duration
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if webcam.Read(img) {
			return true
		}
		if img.Empty() {
			delay = time.Duration(math.Pow(2, float64(attempt))) * time.Millisecond
			fmt.Printf("Retrying in %v...\n", delay)
			time.Sleep(delay)
		}
	}
	return false
}

// blendFrames blends two frames with a specified alpha value
func blendFrames(frame1, frame2 gocv.Mat, result *gocv.Mat, alpha float64) {
	gocv.AddWeighted(frame1, alpha, frame2, 1-alpha, 0, result)
}

// applyEmbossEffect applies an emboss effect to the input frame
func applyEmbossEffect(input gocv.Mat, output *gocv.Mat) {
	// Define an emboss kernel
	kernel := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(0, 0, 0, 0), 3, 3, gocv.MatTypeCV32F)
	defer kernel.Close()

	kernel.SetFloatAt(0, 0, -2)
	kernel.SetFloatAt(0, 1, -1)
	kernel.SetFloatAt(0, 2, 0)
	kernel.SetFloatAt(1, 0, -1)
	kernel.SetFloatAt(1, 1, 1)
	kernel.SetFloatAt(1, 2, 1)
	kernel.SetFloatAt(2, 0, 0)
	kernel.SetFloatAt(2, 1, 1)
	kernel.SetFloatAt(2, 2, 2)

	// Apply the emboss kernel to the input image
	gocv.Filter2D(input, output, -1, kernel, image.Pt(-1, -1), 0.0, gocv.BorderReflect)
}
