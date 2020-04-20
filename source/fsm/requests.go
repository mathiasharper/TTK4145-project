package fsm

import (
	. "../elevatortypes"
)

func ChooseDirection(elev Elevator) MotorDirection {
	switch elev.Dir {
	case MD_UP:
		if requests_above(elev) {
			return MD_UP
		}
		if requests_below(elev) {
			return MD_DOWN
		}
	case MD_DOWN:
		fallthrough
	case MD_STOP:
		if requests_below(elev) {
			return MD_DOWN
		}
		if requests_above(elev) {
			return MD_UP
		}
	}
	return MD_STOP

}

func ShouldStop(elev Elevator) bool {
	switch elev.Dir {
	case MD_DOWN:
		return (elev.Requests[elev.Floor][BTN_HALLDOWN] || elev.Requests[elev.Floor][BTN_CAB] || !requests_below(elev))
	case MD_UP:
		return (elev.Requests[elev.Floor][BTN_HALLUP] || elev.Requests[elev.Floor][BTN_CAB] || !requests_above(elev))
	default:
		return true
	}
}

func ShouldClearAtCurrentFloor(elev Elevator) bool {
	for btn := range elev.Requests[elev.Floor] {
		if elev.Requests[elev.Floor][btn] {
			return true
		}
	}
	return false
}

func requests_above(elev Elevator) bool {
	for floor := elev.Floor + 1; floor < len(elev.Requests); floor++ {
		for btn := range elev.Requests[floor] {
			if elev.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}

func requests_below(elev Elevator) bool {
	for floor := 0; floor < elev.Floor; floor++ {
		for btn := range elev.Requests[floor] {
			if elev.Requests[floor][btn] {
				return true
			}
		}
	}
	return false
}
