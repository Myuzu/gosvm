package main

import (
	"gocv.io/x/gocv"
)

type CircularFrameBuffer struct {
	frameBuffer []gocv.Mat // Frames slice
	frozenFrame gocv.Mat   // Last freeze frame
	blendOffset int        // Number of frames to delay when blending
	size        int        // frameBuffer capacity
	head        int        // First frame idx
	tail        int        // Last frame idx
	full        bool       // Determines if CircularFrameBuffer is full
	frozen      bool       // State to check if the blend frame is frozen
}

func NewCircularFrameBuffer(size, blendOffset int) *CircularFrameBuffer {
	// Create new CircularFrameBuffer struct
	cfb := CircularFrameBuffer{
		frameBuffer: make([]gocv.Mat, size),
		frozenFrame: gocv.NewMat(),
		blendOffset: blendOffset,
		size:        size,
	}

	// Prepopulate frameBuffer with new empty Mats
	for i := range cfb.frameBuffer {
		cfb.frameBuffer[i] = gocv.NewMat()
	}

	return &cfb
}

func (cfb *CircularFrameBuffer) IsEmpty() bool {
	return !cfb.full && cfb.head == cfb.tail
}

func (cfb *CircularFrameBuffer) IsFull() bool {
	return cfb.full
}

func (cfb *CircularFrameBuffer) IsFrozen() bool {
	return cfb.frozen
}

// Returns BaseFrame (last frame)
// FIXME: add error
func (cfb *CircularFrameBuffer) BaseFrame() gocv.Mat {
	baseFrameIdx := (cfb.tail - 1 + cfb.size) % cfb.size

	return cfb.frameBuffer[baseFrameIdx]
}

// Calculates and returns frozenFrame or blend frame
// based on blendOffset (delay)
func (cfb *CircularFrameBuffer) CalcBlendFrame() gocv.Mat {
	if cfb.IsFrozen() {
		return cfb.frozenFrame
	}

	blendFrameIdx := (cfb.tail - cfb.blendOffset + cfb.size) % cfb.size
	return cfb.frameBuffer[blendFrameIdx]
}

// Toggle FreezeFrame mode
// In this mode BlendFrame would returns frozenFrame
func (cfb *CircularFrameBuffer) ToggleFreezeFrame() bool {
	if cfb.frozen {
		cfb.frozen = false
		// Delete and close frozenFrame
		cfb.frozenFrame.Close()
		return false
	} else {
		cfb.frozen = true
		// Clone gocv.Mat from tail to frozenFrame
		cfb.frozenFrame = cfb.frameBuffer[cfb.tail].Clone()
		return true
	}
}

func (cfb *CircularFrameBuffer) IncBlendOffset() int {
	if cfb.blendOffset < cfb.tail {
		cfb.blendOffset++
	}

	return cfb.blendOffset
}

func (cfb *CircularFrameBuffer) DecBlendOffset() int {
	// TODO: possibly replace cfb.head with 1
	if cfb.blendOffset < cfb.head {
		cfb.blendOffset--
	}

	return cfb.blendOffset
}

func (cfb *CircularFrameBuffer) Enqueue(frame gocv.Mat) bool {
	if cfb.full {
		return false
	}

	cfb.frameBuffer[cfb.tail] = frame
	cfb.tail = (cfb.tail + 1) % cfb.size
	cfb.full = cfb.tail == cfb.head

	return true
}

func (cfb *CircularFrameBuffer) Dequeue() bool {
	if cfb.IsEmpty() {
		return false
	}

	frame := cfb.frameBuffer[cfb.head]
	cfb.head = (cfb.head + 1) % cfb.size
	cfb.full = false
	frame.Close()

	return true
}
