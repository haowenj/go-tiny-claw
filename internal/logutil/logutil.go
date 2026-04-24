package logutil

import "log"

// Init configures the standard logger with timestamp and caller information.
func Init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}
