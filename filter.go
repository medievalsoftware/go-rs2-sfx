package snd

import (
	"fmt"
	"math"
)

// Infinite Impulse Response Filter (IIR Filter)
// https://www.youtube.com/watch?v=9yNQBWKRSs4
type filter struct {
	Pairs       [2]int
	Frequencies [2][2][4]float64
	Ranges      [2][2][4]float64
	Unities     [2]int
}

func (f *filter) read(in *buffer, envelope *envelope) error {
	count := in.u8()

	f.Pairs[0] = int(count >> 4)
	f.Pairs[1] = int(count) & 0xF

	if f.Pairs[0] > 4 || f.Pairs[1] > 4 {
		return fmt.Errorf("IIR filter invalid pair interval [%d, %d]", f.Pairs[0], f.Pairs[1])
	}

	if count != 0 {
		f.Unities[0] = int(in.u16())
		f.Unities[1] = int(in.u16())

		migration := in.u8()

		for direction := 0; direction < 2; direction++ {
			for pair := 0; pair < f.Pairs[direction]; pair++ {
				f.Frequencies[direction][0][pair] = float64(in.u16())
				f.Ranges[direction][0][pair] = float64(in.u16())
			}
		}

		for direction := 0; direction < 2; direction++ {
			for pair := 0; pair < f.Pairs[direction]; pair++ {
				if (migration & (1 << (direction * 4) << pair)) != 0 {
					f.Frequencies[direction][1][pair] = float64(in.u16())
					f.Ranges[direction][1][pair] = float64(in.u16())
				} else {
					f.Frequencies[direction][1][pair] = f.Frequencies[direction][0][pair]
					f.Ranges[direction][1][pair] = f.Ranges[direction][0][pair]
				}
			}
		}

		if migration != 0 || f.Unities[0] != f.Unities[1] {
			return envelope.readShape(in)
		}
	} else {
		f.Unities[0] = 0
		f.Unities[1] = 0
	}
	return nil
}

var _unity float64
var _unity16 int64
var _coef [2][8]float64
var _coef16 [2][8]int64

func (f *filter) eval(dir int, del float64) (order int) {
	var u float64

	if dir == 0 {
		u = float64(f.Unities[0]) + float64(f.Unities[1]-f.Unities[0])*del
		u *= 0.0030517578
		_unity = math.Pow(0.1, u/20.0)
		_unity16 = int64(_unity * 65536.0)
	}

	if f.Pairs[dir] == 0 {
		return 0
	} else {
		u = f.gain(dir, 0, del)

		_coef[dir][0] = -2.0 * u * math.Cos(f.phase(dir, 0, del))
		_coef[dir][1] = u * u

		var n int

		for n = 1; n < f.Pairs[dir]; n++ {
			u = f.gain(dir, n, del)

			a := -2.0 * u * math.Cos(f.phase(dir, n, del))
			b := u * u

			_coef[dir][n*2+1] = _coef[dir][n*2-1] * b
			_coef[dir][n*2] = _coef[dir][n*2-1]*a + _coef[dir][n*2-2]*b

			for pair := n*2 - 1; pair >= 2; pair-- {
				_coef[dir][pair] += _coef[dir][pair-1]*a + _coef[dir][pair-2]*b
			}

			_coef[dir][1] += _coef[dir][0]*a + b
			_coef[dir][0] += a
		}

		if dir == 0 {
			for n = 0; n < f.Pairs[0]*2; n++ {
				_coef[0][n] *= _unity
			}
		}

		for n = 0; n < f.Pairs[dir]*2; n++ {
			_coef16[dir][n] = int64(_coef[dir][n] * 65536.0)
		}

		return f.Pairs[dir] * 2
	}
}

func (f *filter) gain(direction, pair int, delta float64) float64 {
	a := f.Ranges[direction][0][pair]
	b := f.Ranges[direction][1][pair]

	// linear interpolation a->b
	g := a + ((b - a) * delta)

	// conversion to some other unit
	g *= 100.0 / 65536.0

	return 1.0 - math.Pow(10, -g/20.0)
}

func (f *filter) phase(direction, pair int, delta float64) float64 {
	a := f.Frequencies[direction][0][pair]
	b := f.Frequencies[direction][1][pair]

	// linear interpolation a->b
	g := a + ((b - a) * delta)

	// conversion to some other unit
	g *= 1.0 / 8192.0
	return normalize(g)
}

func normalize(f float64) float64 {
	a := 32.703197 * math.Pow(2.0, f)
	return a * math.Pi / 11025.0
}
