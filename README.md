# CSV-Statistics-CLI

Streaming CSV analyzer that reports numeric statistics (min, max, mean, median, stddev) and text summaries (unique count, top-N values) per column.

### ⚙️ How It Works

The tool wraps `encoding/csv` around a buffered reader and consumes the file one row at a time, so the file itself is never materialized in memory. Each column accumulates: an incrementing total and non-empty counter, a per-value frequency map, and — when parsing succeeds — a slice of parsed `float64` values (a leading `,` thousands separator is stripped before parsing). After reading, a column is classified numeric if at least 90 % of its non-empty values parsed as numbers; otherwise it is treated as text. Numeric columns then sort their values to compute the exact median plus min/max/mean/sample-stddev; text columns sort their frequency map descending (ties broken alphabetically) and print the top N.

## 📁 Setup

### Requirements
- Go 1.22 or newer
- A CSV file (or piped CSV on stdin)

### Installation
```bash
git clone https://github.com/fantasywastaken/CSV-Statistics-CLI.git
cd CSV-Statistics-CLI
go build -o csvstat .
```

### 🚀 Usage
```bash
csvstat data.csv
csvstat data.csv --column price
csvstat data.tsv --delim '	'
csvstat --no-header data.csv --top 20
cat data.csv | csvstat -
```

### ✨ Features
- Streaming CSV read via `encoding/csv` + 1 MiB buffered reader
- Auto-detects numeric vs. text columns (90 %-parse threshold)
- Numeric: min, max, mean, median (exact), sample stddev
- Text: unique count and top-N most frequent values with counts
- `--column` narrows analysis to a single column
- `--delim` supports tab or other single-character separators
- `--no-header` treats every row as data (columns named `col1`, `col2`, ...)
- `--top` overrides the default top-10 limit
- Reads from stdin when the path is `-`
- Tolerates ragged rows and lazy quoting
