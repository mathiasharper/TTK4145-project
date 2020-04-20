package conf

import (
//	"fmt"
	//"bufio"
	//"os"
	"time"
	//"strconv"
)


const (
	N_FLOORS               = 4
	N_BUTTONS              = 3
	SERVER_PORT            = 15657
	BROADCAST_PORT         = 15898
	BCAST_INTERVAL         = 200 * time.Millisecond
	DOOR_OPEN_TIME         = 4 * time.Second
	WATCHDOG_TIMEOUT       = 7 * time.Second
	BROADCAST_TIMEOUT      = 5 * time.Second
	PACKET_LOSS_PERCENTAGE = 0.0
	USE_LOCAL_IP_AS_ID     = true
	REASIGN_ON_CAB_ORDERS  = false
)
