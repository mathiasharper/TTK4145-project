package fsm

import(
    . "../elevatortypes"
    "../utils"
    "../elevator"
    "../elevio"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
)



func handleNewRequests(localRequests [][]bool, newRequests [][]bool) ([][]bool, bool, ButtonEvent) {
	var btnEvent ButtonEvent
	isNew := false

	for floor := range localRequests {
		for button := range localRequests[floor] {
			//Check if there are any new requests, but don't clear cab orders
			if localRequests[floor][button] != newRequests[floor][button] && !((ButtonType(button) == BTN_CAB) && (localRequests[floor][button])) {
				if !((ButtonType(button) == BTN_CAB) && (localRequests[floor][button])) {
					isNew = true
					btnEvent.Floor = floor
					btnEvent.Button = ButtonType(button)
					localRequests[floor][button] = newRequests[floor][button]
				}
			}
		}
	}
	return localRequests, isNew, btnEvent
}

func writeStateToFile(elev Elevator) {
	requestBytes, err := json.Marshal(elev.Requests)
	if err != nil {
		log.Println(err)
		return
	}
	//Write to two files in case program crashes while writing to one of them
	if err = ioutil.WriteFile("fsm/stateBackup/localElevState1.txt", requestBytes, 0666); err != nil {
		log.Println("Could not write to first file")
	}

	if err = ioutil.WriteFile("fsm/stateBackup/localElevState2.txt", requestBytes, 0666); err != nil {
		log.Println("Could not write to either files")
	}
}

func restoreState(id string, numFloors, numButtons int) Elevator {
	restoredState := elevator.SingleElevatorInit(numFloors, numButtons, id)

	requestBytes, err := ioutil.ReadFile("fsm/stateBackup/localElevState1.txt")
	if err != nil || requestBytes == nil {
		log.Println("fsm: error opening file 1 - ", err)
		requestBytes, err = ioutil.ReadFile("fsm/stateBackup/localElevState2.txt")
		if err != nil {
			log.Println("fsm: error opening file 2 - ", err)
		}
	}

	err = json.Unmarshal(requestBytes, &restoredState.Requests)

	if err != nil {
		log.Println("Could not parse file content of previous state")
		return elevator.SingleElevatorInit(numFloors, numButtons, id)
	}

	fmt.Printf("FSM: Restored previous state - %+v", restoredState)
	return restoredState
}

//InitFsm - Initialize local FSM
func InitFsm(
	updateStateC chan<- Elevator,
	updateFSMRequestsC <-chan [][]bool,
	floorEventC <-chan int,
	doorTimerFinishedC <-chan bool,
	doorTimerStartC chan<- bool,
	elevID string,
	numFloors, numButtons int) {

	elev := restoreState(elevID, numFloors, numButtons)
	elevio.SetDoorOpenLamp(false)

	//Run down until a floor is reached
	elev.Dir = MD_DOWN
	elevio.SetMotorDirection(elev.Dir)
	elev.EState = ES_MOVING

	for {
		//update elevator state to the distributor
		updateStateC <- elev
		writeStateToFile(elev)
		select {

		//New requests received from distributor
		case newRequests := <-updateFSMRequestsC:
			//If there are unknown requests, add them to local state and generate a button event
			tempRequests, isNew, buttonEvent := handleNewRequests(utils.CopySlice(elev.Requests), newRequests)
			if isNew {
				elev.Requests = tempRequests
				switch elev.EState {
				case ES_IDLE:
					if elev.Floor == buttonEvent.Floor {
						for btn := BTN_HALLUP; btn <= BTN_CAB; btn++ {
							elev.Requests[elev.Floor][btn] = false
						}
						elevio.SetDoorOpenLamp(true)
						doorTimerStartC <- true
						elev.EState = ES_DOOR_OPEN
					} else {
						elev.Dir = ChooseDirection(elev)
						elevio.SetMotorDirection(elev.Dir)
						elev.EState = ES_MOVING
					}

				case ES_DOOR_OPEN:
					if elev.Floor == buttonEvent.Floor {
						for btn := BTN_HALLUP; btn <= BTN_CAB; btn++ {
							elev.Requests[elev.Floor][btn] = false
						}
						doorTimerStartC <- true
					}

				default:
				}
			}

		case floor := <-floorEventC:
			elev.Floor = floor
			elevio.SetFloorIndicator(elev.Floor)

			if elev.EState == ES_MOVING {
				if ShouldStop(elev) {
					elevio.SetMotorDirection(MD_STOP)

					if ShouldClearAtCurrentFloor(elev) {
						for btn := BTN_HALLUP; btn <= BTN_CAB; btn++ {
							elev.Requests[floor][btn] = false
						}
						doorTimerStartC <- true
						elev.EState = ES_DOOR_OPEN
						elevio.SetDoorOpenLamp(true)
					} else {
						elev.Dir = ChooseDirection(elev)
						elevio.SetMotorDirection(elev.Dir)

						if elev.Dir == MD_STOP {
							elev.EState = ES_IDLE
						} else {
							elev.EState = ES_MOVING
						}
					}
				}
			}

		case <-doorTimerFinishedC:
			if elev.EState == ES_DOOR_OPEN {
				elevio.SetDoorOpenLamp(false)
				elev.Dir = ChooseDirection(elev)
				elevio.SetMotorDirection(elev.Dir)
			}

			if elev.Dir == MD_STOP {
				elev.EState = ES_IDLE
			} else {
				elev.EState = ES_MOVING
			}
		}
	}
}
