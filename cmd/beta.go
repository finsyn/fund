package main

import (
	"encoding/csv"
	"gonum.org/v1/gonum/stat"
	"io"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Quote struct {
	Date  time.Time
	Close float64
}

func main() {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	qStock := readQuotes("data/GENO-2023-07-13-2024-07-13.csv", "Date", "Closing price")
	qMarket := readQuotes("data/_SE0003045640_2024-07-13.csv", "Date", "Closingprice")
	slog.Info("stock quotes", "num", len(qStock))
	slog.Info("market quotes", "num", len(qStock))
	x, y := returnPairs(qStock, qMarket)
	slog.Info("return sample", "x", x[0], "y", y[0])
	slog.Info("return pairs", "num_x", len(x), "num_y", len(y))
	cov := stat.Covariance(x, y, nil)
	slog.Info("stats", "cov", cov, "beta", cov/stat.Variance(y, nil))
}

func readQuotes(path, dateCol, priceCol string) []Quote {
	f, err := os.Open(path)
	if err != nil {
		log.Fatal("Unable to read input file ", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'

	// skip separator meta row
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil {
		log.Fatal("Unable to read separator row ", err)
	}
	// skip separator meta row
	//r.FieldsPerRecord = 12
	h, err := r.Read()
	if err != nil {
		log.Fatal("Unable to read header row ", err)
	}
	// find column indexes
	dateIndex := -1
	priceIndex := -1
	for i, h := range h {
		if h == dateCol {
			dateIndex = i
		}
		if h == priceCol {
			priceIndex = i
		}
	}
	if dateIndex == -1 {
		log.Fatal("Unable to find column ", dateCol)
	}
	if priceIndex == -1 {
		log.Fatal("Unable to find column ", priceCol)
	}

	var quotes []Quote
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("failed reading data row ", err)
		}
		d, err := time.Parse("2006-01-02", record[dateIndex])
		if err != nil {
			log.Fatal("failed parsing date ", err)
		}
		p, err := strconv.ParseFloat(strings.ReplaceAll(record[priceIndex], ",", "."), 64)
		if err != nil {
			slog.Warn("failed parsing price", "value", p)
			continue
		}
		quotes = append(quotes, Quote{
			Date:  d,
			Close: p,
		})
	}
	return quotes
}

func returnPairs(a, b []Quote) ([]float64, []float64) {
	var x, y []float64
	var aIdx, bIdx int
	var d time.Time
	if a[aIdx].Date.Before(b[bIdx].Date) {
		d = b[bIdx].Date
	} else {
		d = a[bIdx].Date
	}
	for {
		if a[aIdx].Date.After(d) {
			aIdx++
			if aIdx+1 >= len(a) {
				break
			}
			continue
		}
		if b[bIdx].Date.After(d) {
			bIdx++
			if bIdx+1 >= len(b) {
				break
			}
			continue
		}
		if a[aIdx].Date.Equal(b[bIdx].Date) {
			ra := a[aIdx].Close/a[aIdx+1].Close - 1
			rb := b[bIdx].Close/b[bIdx+1].Close - 1
			x = append(x, ra)
			y = append(y, rb)
			slog.Debug("date match", "date", d, "ra", ra, "rb", rb)
		}
		d = d.AddDate(0, 0, -1)
	}
	return x, y
}
