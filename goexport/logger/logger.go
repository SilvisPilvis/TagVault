package logger

import (
	"log"
	"os"
)

func InitLogger() *log.Logger {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return log.New(os.Stdout, "", log.LstdFlags)
}
