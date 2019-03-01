package util

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Simple log function for logging to a file
func Log(level int, logmessage interface{}) (newerr error) {
	logLevel := level
	message := fmt.Sprint(logmessage)
	newerr = nil
	if logmessage == nil {
		return
	}

	if test, ok := logmessage.(error); ok {
		logLevel = 3
		switch test {
		case nil:
			logLevel = 1
			return
		case sql.ErrNoRows:
			message = "No records found"
			newerr = errors.New(message)
		default:
			message = test.Error()
			newerr = test
		}
	}

	if test, ok := logmessage.(string); ok {
		message = test
	}

	if strings.HasPrefix(message, "Fatal") {
		log.Fatalf(message)
		os.Exit(1)
	}

	if strings.Contains(message, "no rows in result set") {
		message = "No records found"
		newerr = errors.New(message)
		logLevel = 3
	}

	if strings.Contains(message, "SQL syntax") {
		newerr = errors.New("Server Error")
		logLevel = 3
	}

	if strings.Contains(message, "Error 1062: Duplicate entry") {
		logLevel = 3
		parts := strings.Split(message, "for key ")
		if len(parts) == 2 {
			newerr = errors.New("Duplicate field " + parts[1])
		} else {
			newerr = errors.New("Duplicate entry")
		}

	}

	trackingIDString := ""
	f, err := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	log.SetOutput(f)

	// Check logLevel
	if logLevel > 4 {
		// Default to highest available to avoid returning errors
		logLevel = 4
	}

	// Check log level based on config
	// logLevel is an int: 0 debug, 1 info, 2 warning, 3 error, 4 critical
	// List of colours: https://radu.cotescu.com/coloured-log-outputs/
	// Default Blue
	colourBegin := "\033[0;34m DEBUG - "
	switch Config.LogLevel {
	case "critical":
		if logLevel <= 4 {
			return
		}
		break
	case "error":
		if logLevel <= 3 {
			return
		}
		break
	case "warning":
		if logLevel <= 2 {
			return
		}
		break
	case "info":
		if logLevel <= 1 {
			return
		}
		break
	case "debug":
		// Log everything
		break
	}

	// Set colours
	switch logLevel {
	case 4:
		// High intensity red
		colourBegin = "\033[0;91m CRITICAL - "
		break
	case 3:
		// Red
		colourBegin = "\033[0;31m ERROR - "
		break
	case 2:
		// Yellow
		colourBegin = "\033[0;33m WARNING - "
		break
	case 1:
		// Cyan
		colourBegin = "\033[0;36m INFO - "
		break
	case 0:
		// Log everything
		break
	}

	nano := strconv.FormatInt(time.Now().UnixNano(), 10)

	colourEnd := " :: time " + nano + " \033[39m"

	// Construct message
	message = MyCaller() + " :: " + message

	log.Printf("%s#%s %s %s", colourBegin, trackingIDString, message, colourEnd)
	fmt.Println(message)

	return
}

func MyCaller() string {

	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// skip 3 levels to get to the caller of whoever called Caller()
	n := runtime.Callers(3, fpcs)
	if n == 0 {
		return "n/a" // proper error her would be better
	}

	// get the info of the actual function that's in the pointer
	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "n/a"
	}
	file, line := fun.FileLine(fpcs[0] - 1)

	// return its name
	//parts := strings.Split(file, "github.com/figassis/gopress/")
	return file + ":" + fmt.Sprint(line) + " - " + filepath.Base(fun.Name())
}

func FunctionName() string {

	// we get the callers as uintptrs - but we just need 1
	fpcs := make([]uintptr, 1)

	// skip 3 levels to get to the caller of whoever called FunctionName()
	n := runtime.Callers(2, fpcs)
	if n == 0 {
		return "n/a" // proper error her would be better
	}

	// get the info of the actual function that's in the pointer
	fun := runtime.FuncForPC(fpcs[0] - 1)
	if fun == nil {
		return "n/a"
	}

	return filepath.Base(fun.Name())
}
