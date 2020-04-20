package main

import(
    "./conf"
    "./elevio"
    . "./elevatortypes"
    "./elevator"
    "./network/bcast"
    "./network/localip"
    "./distributor"
    "./distributor/watchdogtimer"
    "./fsm"
    "time"
    "flag"
    "log"
    "strconv"
)
func main() {

	// Elevator id -> IP
	var LOCAL_ID string
	if conf.USE_LOCAL_IP_AS_ID {
		ip, err := localip.LocalIP()
        log.Println(ip)
		if err != nil {
			log.Panic("Main: Couldn't find IP!")
		}
		LOCAL_ID = ip
	} else {
		flag.StringVar(&LOCAL_ID, "id", "", "Local id for elevator")
		if LOCAL_ID == "" {
			log.Fatal("Main: No id input argument")
		}
	}


	// Global State
	globalState := elevator.GlobalElevatorInit(conf.N_FLOORS, conf.N_BUTTONS, LOCAL_ID)

	// Initialize hardware
	localAddress := "localhost:" + strconv.Itoa(conf.SERVER_PORT)
	elevio.Init(localAddress, conf.N_FLOORS)
	buttonOrderC := make(chan ButtonEvent)
	floorEventC := make(chan int)
	go elevio.PollButtons(buttonOrderC)
	go elevio.PollFloorSensor(floorEventC)

	// Door timer
	doorTimerFinishedC := make(chan bool)
	doorTimerStartC := make(chan bool, 10) //Buffered to avoid blocking
	go fsm.InitDoorTimer(doorTimerFinishedC, doorTimerStartC, conf.DOOR_OPEN_TIME)

	// Initialize local FSM
	localStateUpdateC := make(chan Elevator)
	updateFSMRequestsC := make(chan [][]bool, 10)
	go fsm.InitFsm(
		localStateUpdateC,
		updateFSMRequestsC,
		floorEventC,
		doorTimerFinishedC,
		doorTimerStartC,
		LOCAL_ID,
		conf.N_FLOORS,
		conf.N_BUTTONS)

	// Just works nice
	tid := time.Now()
	for time.Now().Sub(tid) < 2*time.Second {
		select {
		case <-localStateUpdateC:
		default:
		}
	}

	// Initialize the network
	networkTxC := make(chan GlobalElevator)
	networkRxC := make(chan GlobalElevator)
	go bcast.Transmitter(conf.BROADCAST_PORT, networkTxC)
	go bcast.Receiver(conf.BROADCAST_PORT, networkRxC)

	//Initialize broadcasting
	updateBCastPacketC := make(chan GlobalElevator, 10)
	netStateUpdateC := make(chan GlobalElevator)
	elevatorLostEventC := make(chan string)
	go bcast.BroadcastState(updateBCastPacketC, networkTxC, conf.BCAST_INTERVAL)
	updateBCastPacketC <- globalState
	go bcast.BroadcastListener(
		networkRxC,
		elevatorLostEventC,
		netStateUpdateC,
		conf.BROADCAST_TIMEOUT,
		LOCAL_ID,
		conf.PACKET_LOSS_PERCENTAGE)

	// Initialize watchdog timer
	watchdogTimeoutC := make(chan bool)
	watchdogUpdateStateC := make(chan GlobalElevator, 10)
	go watchdogtimer.InitWatchdogTimer(watchdogTimeoutC, watchdogUpdateStateC, conf.WATCHDOG_TIMEOUT)

	// Merge the state update channels from network and local FSM 
	stateUpdateC := make(chan GlobalElevator)
	go mergeUpdateChannels(netStateUpdateC, localStateUpdateC, stateUpdateC)
	localStateUpdateC <- globalState.Elevators[LOCAL_ID]

	log.Println("Main: System initiated")
	distributor.RunDistributor(
		globalState,
		updateBCastPacketC,
		stateUpdateC,
		elevatorLostEventC,
		watchdogTimeoutC,
		watchdogUpdateStateC,
		buttonOrderC,
		updateFSMRequestsC,
		conf.N_FLOORS,
		conf.N_BUTTONS,
		conf.REASIGN_ON_CAB_ORDERS)
}

func mergeUpdateChannels(networkC <-chan GlobalElevator, fsmC <-chan Elevator, outC chan<- GlobalElevator) {

	localElev := <-fsmC
	globalDummy := elevator.GlobalElevatorInit(conf.N_FLOORS, conf.N_BUTTONS, localElev.ID)

	for {
		select {
		case update := <-networkC:
			outC <- update

		case globalDummy.Elevators[globalDummy.ID] = <-fsmC:
			outC <- globalDummy
		}
	}
}
