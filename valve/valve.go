package valve

import (
	"math"

	"heatingcontrol/config"
	"heatingcontrol/mosquito"
)

const (
	DefaultValveOpenness uint = 50
)

type MessageOk struct {
	Ok bool
}

type Valve struct {
	client       mosquito.Client
	cfg          *config.Config
	CurrentLevel *uint
	MsgReceived  chan MessageOk
	done         chan struct{}
}

func NewValve(client mosquito.Client, cfg *config.Config) *Valve {
	return &Valve{
		client:      client,
		cfg:         cfg,
		MsgReceived: make(chan MessageOk, 1),
		done:        make(chan struct{}, 1),
	}
}

func (v *Valve) Start() {
	v.client.SubValveLevel()

	go func(cli mosquito.Client) {
		for {
			select {
			case d := <-cli.ValveListener:
				lvl := d.Level
				// impossible situation. just in case.
				if lvl > 100 {
					lvl = 100
				}
				v.CurrentLevel = &lvl
				v.MsgReceived <- MessageOk{Ok: true}
			case <-v.done:
				return
			}
		}
	}(v.client)
}

func (v *Valve) Stop() {
	v.done <- struct{}{}
	close(v.done)
	close(v.MsgReceived)
}

// DefineChangeTemperaturePercentage returns value representing a number on
// how many percent current temperature will be changed. The returning value depends on valve openness level.
// Positive number means that the temperature will be increased, negative number means that
// the temperature will be increased on than percentage.
func DefineChangeTemperaturePercentage(valveLevel uint) int {
	// round up to get higher value
	tmp := float64(valveLevel) / float64(10)
	valveLevel = uint(math.Round(tmp+0.4999) * 10)

	switch valveLevel {
	case 0:
		return -50
	case 10:
		return -40
	case 20:
		return -30
	case 30:
		return -20
	case 40:
		return -10
	case 50:
		return 0
	case 60:
		return 10
	case 70:
		return 20
	case 80:
		return 30
	case 90:
		return 40
	case 100:
		return 50
	default:
		return 0
	}
}
