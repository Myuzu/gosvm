package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strconv"
	"time"

	"gocv.io/x/gocv"
)

const (
	frameBufferSize  = 325 // Buffer size to hold past frames
	maxCameraRetries = 5   // Maximum number for exponential read retries
)

var faceDetectColor = color.RGBA{0, 255, 0, 70}
var fpsCounterColor = color.RGBA{0, 255, 0, 0}

type FPSCalculator struct {
	frameCount int
	startTime  time.Time
	fps        float64
}

func NewFPSCalculator() *FPSCalculator {
	return &FPSCalculator{
		startTime: time.Now(),
	}
}

func (fpsCalc *FPSCalculator) calculateFPS() float64 {
	fpsCalc.frameCount++
	elapsedTime := time.Since(fpsCalc.startTime).Seconds()
	if elapsedTime > 1.0 {
		fpsCalc.fps = float64(fpsCalc.frameCount) / elapsedTime
		fpsCalc.frameCount = 0
		fpsCalc.startTime = time.Now()
	}
	return fpsCalc.fps
}

func main() {
	if len(os.Args) < 4 {
		fmt.Println("How to run:\n\n\tgosvm [camera ID] blendOffset")
		return
	}

	// Parse args
	deviceID, _ := strconv.Atoi(os.Args[1])

	// Number of frames to delay when blending
	blendOffset, _ := strconv.Atoi(os.Args[2])

	// Open webcam
	webcam, err := gocv.VideoCaptureDevice(int(deviceID))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	// Open display window
	window := gocv.NewWindow("Motion Extraction")
	defer window.Close()

	// Prepare a buffer to hold past frames
	frameBuffer := make([]gocv.Mat, frameBufferSize)
	for i := range frameBuffer {
		frameBuffer[i] = gocv.NewMat()
	}

	// Initialize FPS calculator
	var fpsCalculator *FPSCalculator

	if len(os.Args) == 4 {
		fpsCalculator = NewFPSCalculator()
	}

	fmt.Printf("start reading camera device: %v\n", deviceID)

	currentIndex := 0

	for {
		currentFrame := gocv.NewMat()
		defer currentFrame.Close()

		if !ReadWebcamWithRetry(webcam, &currentFrame, maxCameraRetries) {
			fmt.Println("Failed to read from webcam after multiple attempts")
			break
		}

		if currentFrame.Empty() {
			currentFrame.Close()
			break
		}

		// Close the Mat at the current buffer position to avoid memory leaks
		if frameBuffer[currentIndex].Empty() == false {
			frameBuffer[currentIndex].Close()
		}

		// Store the current frame in the buffer
		frameBuffer[currentIndex] = currentFrame.Clone()

		// Determine the frame to blend with based on the desired delay
		blendIndex := (currentIndex - blendOffset + frameBufferSize) % frameBufferSize
		blendFrame := frameBuffer[blendIndex]

		if !blendFrame.Empty() {
			// Create a half-transparent inverted version of the frame to blend
			halfTransparentFrame := gocv.NewMat()
			defer halfTransparentFrame.Close()

			gocv.BitwiseNot(blendFrame, &halfTransparentFrame)
			gocv.AddWeighted(halfTransparentFrame, 0.5, currentFrame, 0.0, 0, &halfTransparentFrame)

			// Apply emobss effect
			// applyEmbossEffect(halfTransparentFrame, &halfTransparentFrame)

			// Blend the current frame with the delayed frame
			blendedFrame := gocv.NewMat()
			defer blendedFrame.Close()

			blendFrames(currentFrame, halfTransparentFrame, &blendedFrame, 0.4)

			// Calculate FPS
			if fpsCalculator != nil {
				fps := fpsCalculator.calculateFPS()
				gocv.PutText(&blendedFrame, fmt.Sprintf("FPS: %.2f", fps), image.Pt(10, 30), gocv.FontHersheyPlain, 1.5, fpsCounterColor, 1)
			}

			// Display the resulting frame in the window
			window.IMShow(blendedFrame)
		}

		// Move to the next index in the circular buffer
		currentIndex = (currentIndex + 1) % frameBufferSize

		// Wait for a small delay to simulate time difference or until a key is pressed
		if window.WaitKey(10) >= 0 {
			break
		}
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
