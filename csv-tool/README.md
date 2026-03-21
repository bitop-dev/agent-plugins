# csv-tool

Parse, filter, and query CSV/TSV data. Converts raw CSV into structured JSON
or markdown tables that LLMs can work with effectively.

## Tools

### `csv/parse`
Parse raw CSV data into JSON objects, markdown tables, or a preview summary.

### `csv/query`
Filter rows, select columns, and sort CSV data without processing the entire file.

## Build

```bash
cd cmd/csv-tool && go build -o csv-tool .
```
