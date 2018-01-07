package fredlog

import (
	// standard packages
	"log"
	"strconv"
)

type FredConfig struct {
	Global_log bool `toml:"global_log"`
}

func PrintLog(conf *FredConfig, logmsg string) {
	if conf.Global_log != false {
		log.Println(logmsg)
	}
}

func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}
