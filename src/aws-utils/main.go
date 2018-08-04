package main

import (
	"betterweb"
	"btrzaws"
	"fmt"
	"github.com/bsphere/le_go"
	"log"
	"os"
)

var ()

func main() {
	sess, err := btrzaws.GetAWSSession()
	if err != nil {
		log.Fatal(err)
	}
	leToken := os.Getenv("LE_TOKEN")
	if leToken != "" {
		le, _ := le_go.Connect(leToken)
		le.Print("monitor starting")
	}
	server := betterweb.CreateHealthCheckServer()
	server.SetSession(sess)
	fmt.Printf("monitor started.\r\n")
	server.Start()
}
