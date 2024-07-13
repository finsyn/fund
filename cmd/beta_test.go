package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestReturnPairs(t *testing.T) {
	a := []Quote{
		{Date: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC), Close: 102},
		{Date: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC), Close: 101},
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Close: 100},
		// missing date
	}
	b := []Quote{
		{Date: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC), Close: 203},
		// missing one date
		{Date: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), Close: 201},
	}
	x, y := returnPairs(a, b)
	assert.Equal(t, []float64{103}, x)
	assert.Equal(t, []float64{203}, y)
}
