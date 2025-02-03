package parsers

import (
	"time"

	"github.com/sirupsen/logrus"
)

func GetTimeUnit(unit string) time.Duration {
	switch unit {
	case "h":
		return time.Hour
	case "m":
		return time.Minute
	case "s":
		return time.Second
	case "ms":
		return time.Millisecond
	case "micros":
		return time.Microsecond
	}

	logrus.Fatal("Unknown time unit format")
	return 0
}
