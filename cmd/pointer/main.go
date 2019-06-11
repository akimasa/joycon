package main

import (
	"log"
	"math"
	"os"
	"os/signal"

	"github.com/akimasa/joycon"
	"github.com/go-vgo/robotgo"
)

var (
	oldButtons uint32
	oldStick   joycon.Vec2
	oldBattery int
	rumbleData = []joycon.RumbleSet{
		{
			{HiFreq: 16, HiAmp: 80, LoFreq: 16, LoAmp: 80}, // Left
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0},   // Right
		},
		{
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // Left
			{HiFreq: 64, HiAmp: 0, LoFreq: 64, LoAmp: 0}, // Right
		},
	}
)

// Joycon ...
type Joycon struct {
	dx, dy    float32
	stop      bool
	scroll    bool
	scrollPos float32
	*joycon.Joycon
}

func (jc *Joycon) stateHandle(s joycon.State) {
	defer func() {
		oldButtons = s.Buttons
		oldStick = s.RightAdj
	}()
	if oldBattery != s.Battery {
		log.Println("battery:", s.Battery, "%")
	}
	oldBattery = s.Battery
	downButtons := s.Buttons & ^oldButtons
	upButtons := ^s.Buttons & oldButtons
	switch {
	case downButtons == 0:
	default:
		log.Printf("down: %06X", downButtons)
	case downButtons>>22&1 == 1: // L
		jc.scroll = true
	case downButtons>>23&1 == 1: // ZL
		robotgo.MouseClick("left")
	case downButtons>>16&1 == 1: // Down
		robotgo.KeyTap("down")
	case downButtons>>17&1 == 1: // Up
		robotgo.KeyTap("up")
	case downButtons>>18&1 == 1: // Right
		robotgo.KeyTap("right")
	case downButtons>>19&1 == 1: // Left
		robotgo.KeyTap("left")
	case downButtons>>20&1 == 1: // SR
		// robotgo.KeyTap("f4", "ctrl")
		robotgo.MouseToggle("down", "right")
	case downButtons>>21&1 == 1: // SL
		robotgo.KeyTap("f4", "alt")
	case downButtons>>8&1 == 1: // -
		robotgo.KeyTap("escape")
	case downButtons>>11&1 == 1: // LStick Push
		robotgo.MouseClick("center")
	case downButtons>>13&1 == 1: // Capture
	}
	switch {
	case upButtons == 0:
	default:
		log.Printf("up  : %06X", upButtons)
	case upButtons>>22&1 == 1: // L
		jc.scroll = false
	case upButtons>>7&1 == 1: // ZL
	case upButtons>>20&1 == 1: // SR
		robotgo.MouseToggle("up", "right")
	case upButtons>>0&1 == 1: // Y
	case upButtons>>1&1 == 1: // X
	case upButtons>>2&1 == 1: // B
	case upButtons>>3&1 == 1: // A
	case downButtons>>5&1 == 1: // SL
	case upButtons>>9&1 == 1: // +
	case upButtons>>10&1 == 1: // RStick Push
	case upButtons>>12&1 == 1: // Home
	}
	if jc.scroll {
		robotgo.Scroll(0, int(s.LeftAdj.Y*3))
	} else {
		x, y := robotgo.GetMousePos()
		x += int(s.LeftAdj.X * s.LeftAdj.X * (s.LeftAdj.X / float32(math.Abs(float64(s.LeftAdj.X)))) * 80)
		y -= int(s.LeftAdj.Y * s.LeftAdj.Y * (s.LeftAdj.Y / float32(math.Abs(float64(s.LeftAdj.Y)))) * 80)
		robotgo.MoveMouse(x, y)
	}
}

func (jc *Joycon) apply() {
	if (jc.dx != 0 || jc.dy != 0) && !jc.stop {
		x, y := robotgo.GetMousePos()
		w, h := robotgo.GetScreenSize()
		x += int(jc.dx)
		y += int(jc.dy)
		if x >= w {
			x = w
		}
		if x < 0 {
			x = 0
		}
		if y >= h {
			y = h
		}
		if y < 0 {
			y = 0
		}
		robotgo.MoveMouse(x, y)
		jc.dx = 0
		jc.dy = 0
	}
}

func (jc *Joycon) sensorHandle(s joycon.Sensor) {
	if jc.IsLeft() || jc.IsProCon() {
		jc.dx -= s.Gyro.Z * 64
		jc.dy += s.Gyro.Y * 64
	}
	if jc.IsRight() {
		jc.dx += s.Gyro.Z * 64
		jc.dy -= s.Gyro.Y * 64
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds)
	devices, err := joycon.Search(joycon.JoyConL)
	if err != nil {
		log.Fatalln(err)
	}
	j, err := joycon.NewJoycon(devices[0].Path, false)
	if err != nil {
		log.Fatalln(err)
	}
	defer j.Close()
	jc := &Joycon{Joycon: j}
	log.Println("connected:", jc.Name())
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	for {
		select {
		case <-sig:
			return
		case s, ok := <-jc.State():
			if !ok {
				return
			}
			jc.stateHandle(s)
			// jc.apply()
		case s, ok := <-jc.Sensor():
			if !ok {
				return
			}
			jc.sensorHandle(s)
		}
	}
}
