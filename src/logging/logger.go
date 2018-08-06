package logging

import (
	"github.com/bsphere/le_go"
	"log"
	"os"
)

// RecordLogLine - record line to le or stdout
func RecordLogLine(line string) {
	leToken := os.Getenv("LE_TOKEN")
	if leToken != "" {
		le, _ := le_go.Connect(leToken)
		le.Printf(line)
	} else {
		log.Println(line)
	}
}
