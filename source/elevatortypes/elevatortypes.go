package elevatortypes

import (
	"../utils"
)

type ElevState int

const (
	ES_IDLE ElevState = iota
	ES_MOVING
	ES_DOOR_OPEN
)

type MotorDirection int

const (
	MD_UP   MotorDirection = 1
	MD_DOWN                = -1
	MD_STOP                = 0
)

type ButtonType int

const (
	BTN_HALLUP   ButtonType = iota
	BTN_HALLDOWN
	BTN_CAB
)

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type GlobalElevator struct {
	HallRequests [][]bool
	Elevators    map[string]Elevator
	ID           string
}

func (g GlobalElevator) Copy() GlobalElevator {
	var cpy GlobalElevator
	cpy.ID = g.ID
	cpy.Elevators = make(map[string]Elevator)
	for k, v := range g.Elevators {
		cpy.Elevators[k] = v.Copy()
	}

	cpy.HallRequests = utils.CopySlice(g.HallRequests)
	return cpy
}

func (g GlobalElevator) IsHallOrder(floor int, btn ButtonType) bool {
	return g.HallRequests[floor][btn]
}

func (g GlobalElevator) IsCabOrder(id string, floor int) bool {
	return g.Elevators[id].Requests[floor][BTN_CAB]
}

type Elevator struct {
	ID       string
	Floor    int
	Dir      MotorDirection
	EState   ElevState
	Requests [][]bool
}

func (e Elevator) Copy() Elevator {
	var cpy Elevator
	cpy.ID = e.ID
	cpy.Floor = e.Floor
	cpy.Dir = e.Dir
	cpy.EState = e.EState

	cpy.Requests = utils.CopySlice(e.Requests)
	return cpy
}
