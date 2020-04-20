package elevator

import (
	. "../elevatortypes"
)

// GlobalElevatorInit - Intitialize global elevator state and add local elevator
func GlobalElevatorInit(numFloors, numButtons int, id string) GlobalElevator {

	var global GlobalElevator

	global.HallRequests = make([][]bool, numFloors)
	for floor := range global.HallRequests {
		global.HallRequests[floor] = make([]bool, numButtons-1)
	}

	global.Elevators = make(map[string]Elevator)

	global.Elevators[id] = SingleElevatorInit(numFloors, numButtons, id)
	global.ID = id

	return global
}

// SingleElevatorInit - Initialize a single elevator
func SingleElevatorInit(numFloors, numButtons int, id string) Elevator {
	var e Elevator
	e.ID = id
	e.Requests = make([][]bool, numFloors)
	for floor := range e.Requests {
		e.Requests[floor] = make([]bool, numButtons)
	}
	return e
}
