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
	frameBufferSize  = 110 // Buffer size to hold past frames
	maxCameraRetries = 5   // Maximum number for exponential read retries
	minBlendOffset   = 1   // Minimum frame delay
	maxBlendOffset   = 109 // Maximum frame delay
)

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
	// defer profile.Start(profile.MemProfile).Stop()

	if len(os.Args) < 2 {
		fmt.Println("How to run:\n\n\tgosvm [camera ID]")
		return
	}

	// Parse args
	deviceID, _ := strconv.Atoi(os.Args[1])

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
	fpsCalculator := NewFPSCalculator()

	fmt.Printf("start reading camera device: %v\n", deviceID)

	// Number of frames to delay when blending
	blendOffset := 3

	// Current frame index inside the frameBuffer
	currentIndex := 0

	// State to check if the frame is frozen
	freezeFrame := false

	// Frame to compare with
	mainFrame := gocv.NewMat()

	// frozenFrame
	frozenFrame := gocv.NewMat()

	for {
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
		if frameBuffer[currentIndex].Empty() == false {
			frameBuffer[currentIndex].Close()
		}

		// Denoise currentFrame
		gocv.FastNlMeansDenoisingColoredWithParams(currentFrame, &currentFrame, 28.0, 12.0, 12, 7)

		// Store the current frame in the buffer
		frameBuffer[currentIndex] = currentFrame.Clone()
		currentFrame.Close()

		if !freezeFrame {
			mainFrame = frameBuffer[currentIndex]
			frozenFrame = frameBuffer[currentIndex]
		} else {
			mainFrame = frozenFrame
		}

		// gocv.Multiply(mainFrame, gocv.NewMatWithSizeFromScalar(
		// 	gocv.NewScalar(0.5, 0.5, 0.5, 0),
		// 	mainFrame.Rows(),
		// 	mainFrame.Cols(),
		// 	mainFrame.Type()), &mainFrame)

		// mainFrame.MultiplyFloat(0.5) //.CopyTo(&mainFrame)

		// Determine the frame to blend with based on the desired delay
		blendIndex := (currentIndex - blendOffset + frameBufferSize) % frameBufferSize
		blendFrame := frameBuffer[blendIndex]

		if !blendFrame.Empty() {
			// Create a half-transparent inverted version of the frame to blend
			halfTransparentFrame := gocv.NewMat()

			// Invert the frame
			gocv.BitwiseNot(blendFrame, &halfTransparentFrame)
			gocv.AddWeighted(halfTransparentFrame, 0.5, mainFrame, 0.0, 0, &halfTransparentFrame)

			// Blend the current frame with the delayed frame
			blendedFrame := gocv.NewMat()

			blendFrames(mainFrame, halfTransparentFrame, &blendedFrame, 0.4)

			// Apply emobss effect
			applyEmbossEffect(blendedFrame, &blendedFrame)

			// Calculate FPS
			if fpsCalculator != nil {
				fps := fpsCalculator.calculateFPS()
				gocv.PutText(&blendedFrame, fmt.Sprintf("FPS: %.2f, Delay: %d (A/D keys to inc/dec), Freeze: %t", fps, blendOffset, freezeFrame), image.Pt(10, 40), gocv.FontHersheyPlain, 1.9, fpsCounterColor, 2)
			}

			// Display the resulting frame in the window
			window.IMShow(blendedFrame)

			// Close Mats manualy
			blendedFrame.Close()
			halfTransparentFrame.Close()
		}

		// Move to the next index in the circular buffer
		currentIndex = (currentIndex + 1) % frameBufferSize

		// Handle user input to change the blend offset
		key := window.WaitKey(10)
		if key == 27 || key == 113 { // ESC or Q key to exit
			break
		} else if key == 97 { // "A" key decreses blendOffset
			if blendOffset > minBlendOffset {
				blendOffset--
			}
		} else if key == 100 { // "D" key increses blendOffset
			if blendOffset < maxBlendOffset {
				blendOffset++
			}
		} else if key == 32 { // Spacebar key to togle freeze frame
			freezeFrame = !freezeFrame
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
