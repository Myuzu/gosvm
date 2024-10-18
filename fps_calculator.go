package main

import "time"

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
