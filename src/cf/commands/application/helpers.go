package application

import (
	term "cf/terminal"
	"errors"
	"fmt"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

func byteSize(bytes uint64) string {
	unit := ""
	value := float32(bytes)

	switch {
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes == 0:
		return "0"
	}

	stringValue := fmt.Sprintf("%.1f", value)
	stringValue = strings.TrimRight(stringValue, ".0")
	return fmt.Sprintf("%s%s", stringValue, unit)
}

func bytesFromString(s string) (bytes uint64, err error) {
	unit := string(s[len(s)-1])
	stringValue := s[0 : len(s)-1]

	value, err := strconv.ParseUint(stringValue, 10, 0)
	if err != nil {
		return
	}

	switch unit {
	case "T":
		bytes = value * TERABYTE
	case "G":
		bytes = value * GIGABYTE
	case "M":
		bytes = value * MEGABYTE
	case "K":
		bytes = value * KILOBYTE
	}

	if bytes == 0 {
		err = errors.New("Could not parse byte string")
	}

	return
}

func coloredState(state string) (colored string) {
	switch state {
	case "started", "running":
		colored = term.SuccessColor("running")
	case "stopped":
		colored = term.StoppedColor("stopped")
	case "flapping":
		colored = term.WarningColor("flapping")
	case "starting":
		colored = term.AdvisoryColor("starting")
	default:
		colored = term.FailureColor(state)
	}

	return
}

func logMessageOutput(appName string, lm logmessage.LogMessage) string {
	sourceTypeNames := map[logmessage.LogMessage_SourceType]string{
		logmessage.LogMessage_CLOUD_CONTROLLER: "API",
		logmessage.LogMessage_ROUTER:           "Router",
		logmessage.LogMessage_UAA:              "UAA",
		logmessage.LogMessage_DEA:              "Executor",
		logmessage.LogMessage_WARDEN_CONTAINER: "App",
	}

	sourceType, _ := sourceTypeNames[*lm.SourceType]
	sourceId := "?"
	if lm.SourceId != nil {
		sourceId = *lm.SourceId
	}
	msg := lm.GetMessage()

	t := time.Unix(0, *lm.Timestamp)
	timeString := t.Format("Jan 2 15:04:05")

	channel := ""
	if lm.MessageType != nil && *lm.MessageType == logmessage.LogMessage_ERR {
		channel = "STDERR "
	}

	if lm.GetSourceType() == logmessage.LogMessage_WARDEN_CONTAINER {
		return fmt.Sprintf("%s %s %s/%s %s%s", timeString, appName, sourceType, sourceId, channel, msg)
	}

	return fmt.Sprintf("%s %s %s %s%s", timeString, appName, sourceType, channel, msg)
}

func envVarFound(varName string, existingEnvVars map[string]string) (found bool) {
	for name, _ := range existingEnvVars {
		if name == varName {
			found = true
			return
		}
	}
	return
}

func MapStr(args interface{}) []string {
	r := reflect.ValueOf(args)
	rval := make([]string, r.Len())
	for i := 0; i < r.Len(); i++ {
		rval[i] = r.Index(i).Interface().(fmt.Stringer).String()
	}
	return rval

}
