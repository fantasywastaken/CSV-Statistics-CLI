package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type columnData struct {
	name         string
	total        int
	nonEmpty     int
	numericCount int
	numbers      []float64
	freq         map[string]int
}

func newColumn(name string) *columnData {
	return &columnData{name: name, freq: map[string]int{}}
}

func (c *columnData) add(v string) {
	c.total++
	if v == "" {
		return
	}
	c.nonEmpty++
	if n, err := parseNumber(v); err == nil {
		c.numericCount++
		c.numbers = append(c.numbers, n)
	}
	c.freq[v]++
}

func parseNumber(v string) (float64, error) {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, ",", "")
	if v == "" {
		return 0, errors.New("empty")
	}
	return strconv.ParseFloat(v, 64)
}

func main() {
	col := flag.String("column", "", "analyze only this column name")
	delim := flag.String("delim", ",", "field delimiter (single character)")
	noHeader := flag.Bool("no-header", false, "treat the first row as data, columns named col1, col2, ...")
	topN := flag.Int("top", 10, "top-N values to list for text columns")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: csvstat [flags] <file>   (use - for stdin)")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if utf8.RuneCountInString(*delim) != 1 {
		fmt.Fprintln(os.Stderr, "error: --delim must be a single character")
		os.Exit(2)
	}

	src, closer, err := openSource(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open error: %v\n", err)
		os.Exit(1)
	}
	defer closer()

	reader := csv.NewReader(bufio.NewReaderSize(src, 1<<20))
	reader.Comma = []rune(*delim)[0]
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true

	columns, err := analyze(reader, *noHeader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}

	if *col != "" {
		found := false
		for _, c := range columns {
			if c.name == *col {
				printColumn(c, *topN)
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "column not found: %s\n", *col)
			os.Exit(1)
		}
		return
	}
	for _, c := range columns {
		printColumn(c, *topN)
		fmt.Println()
	}
}

func openSource(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func analyze(r *csv.Reader, noHeader bool) ([]*columnData, error) {
	var cols []*columnData
	first := true
	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				continue
			}
			return nil, err
		}

		if first {
			first = false
			if noHeader {
				cols = make([]*columnData, len(row))
				for i := range row {
					cols[i] = newColumn(fmt.Sprintf("col%d", i+1))
				}
				for i, v := range row {
					cols[i].add(v)
				}
				continue
			}
			cols = make([]*columnData, len(row))
			for i, name := range row {
				n := strings.TrimSpace(name)
				if n == "" {
					n = fmt.Sprintf("col%d", i+1)
				}
				cols[i] = newColumn(n)
			}
			continue
		}

		for i, v := range row {
			if i >= len(cols) {
				break
			}
			cols[i].add(v)
		}
	}
	return cols, nil
}

func printColumn(c *columnData, topN int) {
	fmt.Printf("== %s ==\n", c.name)
	fmt.Printf("rows:      %d\n", c.total)
	fmt.Printf("non-empty: %d\n", c.nonEmpty)
	if c.nonEmpty == 0 {
		fmt.Println("(no values)")
		return
	}
	numericRatio := float64(c.numericCount) / float64(c.nonEmpty)
	if numericRatio >= 0.9 {
		printNumeric(c)
	} else {
		printText(c, topN)
	}
}

func printNumeric(c *columnData) {
	nums := append([]float64(nil), c.numbers...)
	sort.Float64s(nums)
	n := len(nums)
	if n == 0 {
		fmt.Println("(no numeric values)")
		return
	}
	min := nums[0]
	max := nums[n-1]
	sum := 0.0
	for _, v := range nums {
		sum += v
	}
	mean := sum / float64(n)
	variance := 0.0
	for _, v := range nums {
		d := v - mean
		variance += d * d
	}
	stddev := 0.0
	if n > 1 {
		stddev = math.Sqrt(variance / float64(n-1))
	}
	median := 0.0
	if n%2 == 1 {
		median = nums[n/2]
	} else {
		median = (nums[n/2-1] + nums[n/2]) / 2
	}
	fmt.Println("type:      numeric")
	fmt.Printf("count:     %d\n", n)
	fmt.Printf("min:       %g\n", min)
	fmt.Printf("max:       %g\n", max)
	fmt.Printf("mean:      %g\n", mean)
	fmt.Printf("median:    %g\n", median)
	fmt.Printf("stddev:    %g\n", stddev)
}

func printText(c *columnData, topN int) {
	fmt.Println("type:      text")
	fmt.Printf("unique:    %d\n", len(c.freq))
	type kv struct {
		v string
		n int
	}
	list := make([]kv, 0, len(c.freq))
	for v, n := range c.freq {
		list = append(list, kv{v, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n != list[j].n {
			return list[i].n > list[j].n
		}
		return list[i].v < list[j].v
	})
	limit := topN
	if limit > len(list) {
		limit = len(list)
	}
	fmt.Printf("top %d:\n", limit)
	for i := 0; i < limit; i++ {
		fmt.Printf("  %6d  %s\n", list[i].n, truncate(list[i].v, 80))
	}
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "..."
}
