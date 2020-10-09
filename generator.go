package snd

import (
	"math"
	"math/rand"
)

var noise [32768]int
var sin [32768]int

func init() {
	for i := range noise {
		noise[i] = (rand.Int() & 0x2) - 1
	}

	for i := range sin {
		sin[i] = int(math.Sin(float64(i)/(16384.0/math.Pi)) * 16384.0)
	}
}

func generate(form uint8, phase, amplitude int) int {
	switch form {
	case 1: // square wave
		if (phase & 0x7FFF) < 16384 {
			return amplitude
		} else {
			return -amplitude
		}
	case 2: // sine wave
		return (sin[phase&0x7FFF] * amplitude) >> 14
	case 3: // saw wave
		return (((phase & 0x7FFF) * amplitude) >> 14) - amplitude
	case 4: // noise
		return noise[(phase/2607)&0x7FFF] * amplitude
	}
	return 0
}
