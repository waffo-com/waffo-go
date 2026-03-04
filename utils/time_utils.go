package utils

import "time"

const TimestampFormat = "2006-01-02T15:04:05.000Z"

func GetCurrentTimestamp() string {
	return time.Now().UTC().Format(TimestampFormat)
}
