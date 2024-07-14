package financial

import (
	"strconv"
	"strings"
)

func ParseNumber(x string, isUS bool) (float64, error) {
	var y string
	if isUS {
		y = strings.Replace(x, ",", "", -1)
	} else {
		y = strings.Replace(x, ",", ".", -1)
	}
	return strconv.ParseFloat(y, 64)
}
