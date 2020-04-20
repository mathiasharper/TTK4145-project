package reasigner

import (
	"bytes"
	. "../../elevatortypes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)


type elevStruct struct {
	Behaviour   string
	Floor       int
	Direction   string
	CabRequests []bool
}

type jsonElevStruct struct {
	HallRequests [][]bool
	States       map[string]elevStruct
}

//ReasignOrders - use hall_request_assigner to decide which elevator takes which hall orders
func ReasignOrders(globalState GlobalElevator, localIP string) [][]bool {

	numFloors := len(globalState.HallRequests)

	elevStateMap := map[ElevState]string{
		ES_IDLE:      "idle",
		ES_MOVING:    "moving",
		ES_DOOR_OPEN: "doorOpen",
	}

	motorDirMap := map[MotorDirection]string{
		MD_UP:   "up",
		MD_DOWN: "down",
		MD_STOP: "stop",
	}

	elevStatesMap := make(map[string]elevStruct)

	cabRequests := make(map[string][]bool)

	//Parse elevatortypes.Elevator to jsonElevStruct used by reasigner program
	for elevName, elev := range globalState.Elevators {

		//Find cab requests for each elevator
		cabRequests[elevName] = make([]bool, numFloors)
		for floor := range elev.Requests {
			cabRequests[elevName][floor] = elev.Requests[floor][BTN_CAB]
		}

		tempElevStruct := elevStruct{
			Behaviour:   elevStateMap[elev.EState],
			Floor:       elev.Floor,
			Direction:   motorDirMap[elev.Dir],
			CabRequests: cabRequests[elevName],
		}

		elevStatesMap[elevName] = tempElevStruct
	}

	//Create the struct that has the correct format for using hall_request_assigner
	jElevStruct := jsonElevStruct{
		HallRequests: globalState.HallRequests,
		States:       elevStatesMap,
	}

	//Convert the struct to a JSON object
	jsonElevBytes, err := json.Marshal(jElevStruct)
	if err != nil {
		log.Println(err)
	}

	//Format json elevator string to work with hall_request_assigner
	jElevStr := strings.ToLower(string(jsonElevBytes))
	jElevStr = strings.Replace(jElevStr, "hallrequests", "hallRequests", -1)
	jElevStr = strings.Replace(jElevStr, "cabrequests", "cabRequests", -1)
	jElevStr = strings.Replace(jElevStr, "dooropen", "doorOpen", -1)

	//Run hall_request_assigner with the json elev string as input and get the STDOUT output
	cmd := exec.Command("./hall_request_assigner", "-i", jElevStr)
	//cmd := exec.Command("./hello.txt")
	var output bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr
	e := cmd.Run()
	if e != nil {
		log.Panic(fmt.Sprint(e) + ": " + stderr.String())
	}

	//Convert the result (JSON string) result back to a map
	newHallOrders := make(map[string][][]bool)
	json.Unmarshal(output.Bytes(), &newHallOrders)

	//Get the hall orders assigned to the local elevator
	newLocalHallOrders := newHallOrders[localIP]

	//Add the returned hall orders to a slice fitting the format of the local fsm
	newLocalRequests := make([][]bool, numFloors)
	for floor := range newLocalRequests {
		newLocalRequests[floor] = make([]bool, len(globalState.Elevators[localIP].Requests[floor]))

		newLocalRequests[floor][0] = newLocalHallOrders[floor][0]
		newLocalRequests[floor][1] = newLocalHallOrders[floor][1]
		newLocalRequests[floor][2] = cabRequests[localIP][floor]
	}
	return newLocalRequests
}
