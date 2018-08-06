package main

import (
	"betterweb"
	"btrzaws"
	"fmt"
	"logging"
	"os"
)

var ()

func main() {
	sess, err := btrzaws.GetAWSSession()
	if err != nil {
		logging.RecordLogLine(fmt.Sprintf("%v while creating a session", err))
		os.Exit(1)
	}
	logging.RecordLogLine("monitor starting")
	server := betterweb.CreateHealthCheckServer()
	server.SetSession(sess)
	logging.RecordLogLine("monitor started.\r\n")
	server.Start()
}
