package distributor

import (
	. "../elevatortypes"
	"../utils"
	"./reasigner"
	"../elevio"
	"log"
)
// Parse incomming state update
func updateHallRequests(globalState, stateUpdate GlobalElevator) (bool, [][]bool) {
	knownReq := utils.CopySlice(globalState.HallRequests)
	isNew := false

	// Add unknown hall requests
	for floor := range knownReq {
		for btn := BTN_HALLUP; btn <= BTN_HALLDOWN; btn++ {
			if (!globalState.IsHallOrder(floor, btn)) && stateUpdate.IsHallOrder(floor, btn) {
				log.Printf("Distributor: New unknown request Type: %d Floor: %d ID: %s\n",
					btn, floor, stateUpdate.ID)
				knownReq[floor][btn] = true
				isNew = true
			}

		}
	}

	// Clear completed orders
	for _, elev := range globalState.Elevators {
		if elev.EState == ES_DOOR_OPEN {
			knownReq[elev.Floor][BTN_HALLDOWN] = false
			knownReq[elev.Floor][BTN_HALLUP] = false
		}
	}
	return isNew, knownReq
}

func updateLocalLights(
	gState, stateUpdate GlobalElevator,
	oldLightMatrix [][]bool) [][]bool {

	lightMatrix := make([][]bool, len(oldLightMatrix))
	for floor := range lightMatrix {
		lightMatrix[floor] = make([]bool, len(oldLightMatrix[floor]))
		copy(lightMatrix[floor], oldLightMatrix[floor])
		for btn := BTN_HALLUP; btn <= BTN_CAB; btn++ {
			if btn == BTN_CAB {
				lightMatrix[floor][btn] = gState.IsCabOrder(gState.ID, floor)
			} else if gState.IsHallOrder(floor, btn) == stateUpdate.IsHallOrder(floor, btn) {
				lightMatrix[floor][btn] = gState.IsHallOrder(floor, btn)
			}
		}
	}
	return lightMatrix
}

func lightsCompareEq(sli1, sli2 [][]bool, checkHallLights bool) bool {
	for i := range sli1 {
		for j := range sli1[i] {
			if !checkHallLights && j < 2 {
				continue
			}
			if sli1[i][j] != sli2[i][j] {
				return false
			}
		}
	}
	return true
}

func hasNewCabOrders(old, new [][]bool) bool {
	for i := range old {
		if !old[i][BTN_CAB] && new[i][BTN_CAB] {
			return true
		}
	}
	return false
}

func RunDistributor(
	globalState GlobalElevator,
	updateBroadcastedPacketC chan<- GlobalElevator,
	StateUpdateEventC <-chan GlobalElevator,
	elevatorLostEventC <-chan string,
	watchdogTimeoutC <-chan bool,
	watchdogUpdateStateC chan<- GlobalElevator,
	buttonOrderC <-chan ButtonEvent,
	updateFSMRequestsC chan<- [][]bool,
	numFloors, numButtons int,
	reasignOnCabOrders bool) {

	localID := globalState.ID
	lightMatrix := make([][]bool, numFloors)
	for i := range lightMatrix {
		lightMatrix[i] = make([]bool, numButtons)
		for j := range lightMatrix[i] {
			lightMatrix[i][j] = false
		}
	}
	elevio.SetButtonLights(lightMatrix)

	for {
		watchdogUpdateStateC <- globalState.Copy()
		updateBroadcastedPacketC <- globalState.Copy()

		select {

		//Received state update from an elevator (local or network)
		case stateUpdate := <-StateUpdateEventC:
			_, isNewElevator := globalState.Elevators[stateUpdate.ID]
			newCabOrders := false

			if reasignOnCabOrders {
				newCabOrders = hasNewCabOrders(
					globalState.Elevators[stateUpdate.ID].Requests,
					stateUpdate.Elevators[stateUpdate.ID].Requests)
			}

			globalState.Elevators[stateUpdate.ID] = stateUpdate.Elevators[stateUpdate.ID].Copy()
			isNewRequest, newHallRequests := updateHallRequests(globalState, stateUpdate)
			globalState.HallRequests = newHallRequests

			if isNewRequest || !isNewElevator || newCabOrders {
				newLocalRequests := reasigner.ReasignOrders(globalState, localID)
				updateFSMRequestsC <- newLocalRequests
			}

			tempLightMatrix := updateLocalLights(globalState, stateUpdate, lightMatrix)

			//Don't update hall lights if the update is from FSM or if we are alone on the network
			shouldCheckHallLights := globalState.ID != stateUpdate.ID || len(globalState.Elevators) == 1
			if !lightsCompareEq(tempLightMatrix, lightMatrix, shouldCheckHallLights) {
				lightMatrix = tempLightMatrix
				elevio.SetButtonLights(lightMatrix)
			}

		//A button press has been detected locally
		case buttonEvent := <-buttonOrderC:
			log.Print("Distributor: ButtonEvent: ", buttonEvent)
			if buttonEvent.Button == BTN_CAB {
				newLocalRequests := utils.CopySlice(globalState.Elevators[localID].Requests)
				newLocalRequests[buttonEvent.Floor][buttonEvent.Button] = true
				updateFSMRequestsC <- newLocalRequests

			} else if !globalState.HallRequests[buttonEvent.Floor][buttonEvent.Button] {
				globalState.HallRequests[buttonEvent.Floor][buttonEvent.Button] = true
				newLocalRequests := reasigner.ReasignOrders(globalState, localID)
				updateFSMRequestsC <- newLocalRequests
			}

		//Lost connection to an elevator on the network, remove it from global state and reasign orders

		case id := <-elevatorLostEventC:
			log.Println("Distributor: Elevator Lost: " + id)
			if _, ok := globalState.Elevators[id]; ok {
				delete(globalState.Elevators, id)
				newLocalRequests := reasigner.ReasignOrders(globalState, localID)
				updateFSMRequestsC <- newLocalRequests
			}


		case <-watchdogTimeoutC:
			log.Println("Distributor: Watchdog event, taking all orders")
			tempReq := make([][]bool, numFloors)
			for floor := range tempReq {
				tempReq[floor] = make([]bool, numButtons)
				tempReq[floor][BTN_HALLUP] = globalState.IsHallOrder(floor, BTN_HALLUP)
				tempReq[floor][BTN_HALLDOWN] = globalState.IsHallOrder(floor, BTN_HALLDOWN)
				tempReq[floor][BTN_CAB] = globalState.IsCabOrder(localID, floor)
			}
			updateFSMRequestsC <- tempReq

		}
	}
}
