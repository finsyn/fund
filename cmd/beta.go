package main

import (
	"encoding/csv"
	"fmt"
	"github.com/finsyn/fund/pkg/financial"
	"gonum.org/v1/gonum/stat"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Quote struct {
	Date  time.Time
	Close float64
}

type Holding struct {
	ISIN  string
	Name  string
	Value float64
}

const dataDir = "data"

func main() {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	accountNumber := os.Getenv("ACCOUNT_NUMBER")
	if accountNumber == "" {
		log.Fatal("ACCOUNT_NUMBER must be set")
	}

	fMarket := read(filepath.Join(dataDir, "omx-allshares"))
	qMarket, err := parseQuotes(fMarket, "Date", "Closingprice", true)
	if err != nil {
		log.Fatal("failed parsing market quotes", err)
	}
	slog.Info("market quotes", "num", len(qMarket))

	fHoldings := read("data/holdings")
	holdings := readHoldings(fHoldings, accountNumber)

	var betaAndValues [][2]float64
	var sumValues float64

	for _, h := range holdings {
		fStock := read(filepath.Join(dataDir, h.ISIN))
		qStock, err := parseQuotes(fStock, "Date", "Closing price", false)
		if err != nil {
			log.Fatal("failed parsing quotes", err)
		}
		slog.Info("parsed stock quotes", "name", h.Name, "num", len(qStock))

		x, y := returnPairs(qMarket, qStock)
		slog.Info("created return sample", "x", x[0], "y", y[0])
		slog.Info("created return pairs", "num_x", len(x), "num_y", len(y))
		alpha, beta := stat.LinearRegression(x, y, nil, false)
		slog.Info("calculated linear regression", "beta", beta)
		betaAndValues = append(betaAndValues, [2]float64{beta, h.Value})
		sumValues += h.Value

		n := float64(len(x))
		res := residual(x, y, alpha, beta)
		rse := math.Sqrt(res / (n - 2)) // should this be -2 or -1 for a one factor model?
		idioVol := 100 * rse * math.Sqrt(n)
		slog.Info("calculated idiosyncratic volatility", "res", res, "rse", rse, "idioVol", idioVol)
	}

	var beta float64
	for _, bv := range betaAndValues {
		beta += bv[0] * bv[1] / sumValues
	}
	slog.Info("calculated portfolio weighted beta", "beta", beta)
}

func read(dirName string) io.Reader {
	files, err := os.ReadDir(dirName)
	if err != nil {
		log.Fatal("Unable to read input file "+dirName, err)
	}
	for _, f := range files {
		if !f.IsDir() {
			fr, err := os.Open(filepath.Join(dirName, f.Name()))
			if err != nil {
				log.Fatal("Unable to read input file "+f.Name(), err)
			}
			return fr
		}
	}
	log.Fatal("no files found in ", dirName)
	return nil
}

// readHoldings reads a file from avanza with your holdings in a specific account
func readHoldings(f io.Reader, accountNumber string) []Holding {
	r := csv.NewReader(f)
	r.Comma = ';'

	// skip separator meta row
	if _, err := r.Read(); err != nil {
		log.Fatal("Unable to read header row ", err)
	}

	var holdings []Holding
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("failed reading data row ", err)
		}
		an := record[0]
		if an != accountNumber {
			continue
		}
		v, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			log.Fatal("failed parsing value ", err)
		}
		holdings = append(holdings, Holding{
			Name:  record[1],
			Value: v,
			ISIN:  record[5],
		})
	}
	return holdings
}

func parseQuotes(f io.Reader, dateCol, priceCol string, isUS bool) ([]Quote, error) {
	r := csv.NewReader(f)
	r.Comma = ';'

	// skip separator meta row
	r.FieldsPerRecord = -1
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("unable to read separator row %w", err)
	}
	//skip separator meta row
	h, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("unable to read header row %w", err)
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
		return nil, fmt.Errorf("unable to find column %s", dateCol)
	}
	if priceIndex == -1 {
		return nil, fmt.Errorf("unable to find column %s", priceCol)
	}

	var quotes []Quote
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed reading data row %w", err)
		}
		d, err := time.Parse("2006-01-02", record[dateIndex])
		if err != nil {
			return nil, fmt.Errorf("failed parsing date %w", err)
		}
		p, err := financial.ParseNumber(record[priceIndex], isUS)
		if err != nil {
			slog.Warn("failed parsing price", "value", record[priceIndex])
			continue
		}
		quotes = append(quotes, Quote{
			Date:  d,
			Close: p,
		})
	}
	return quotes, nil
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

func residual(x, y []float64, alpha, beta float64) float64 {
	var res float64
	for i, sample := range y {
		r := sample - (alpha + beta*x[i])
		res += math.Pow(r, 2)
	}
	return res
}
