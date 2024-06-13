package util

import (
	"fmt"
	"log"
	"strings"
)

var logEnabled = true
var logOptionsEnabled = "D;I;W;E" //D=Debug; I=Info, W=Warn, E=Error

func doLog(levelLabel string, Msg string) {
	if logEnabled || (levelLabel == "L") {
		log.Printf("%s: %s", levelLabel, Msg)
	}
}

//Sempre loga, independente das options, se logEnabled = true
func Log(msg string) {
	doLog("L", msg)
}

func LogConsole(msg string) {
	doLog("L", msg)
	fmt.Println(msg)
}

func LogD(Msg string) {
	if strings.ContainsAny(logOptionsEnabled, "Dd") {
		doLog("D", Msg)
	}
}

func LogI(Msg string) {
	if strings.ContainsAny(logOptionsEnabled, "Ii") {
		doLog("I", Msg)
	}
}

func LogW(Msg string) {
	if strings.ContainsAny(logOptionsEnabled, "Ww") {
		doLog("W", Msg)
	}
}

func LogE(Msg string) {
	if strings.ContainsAny(logOptionsEnabled, "Ee") {
		doLog("E", Msg)
	}
}

func SetLogEnabled(newValue bool) {
	logEnabled = newValue
}

func GetLogEnabled() bool {
	return logEnabled
}

func SetLogOptions(newValue string) {
	logOptionsEnabled = newValue
}