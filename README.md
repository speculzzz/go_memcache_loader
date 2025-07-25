# Go Memcached Data Loader

A high-performance concurrent loader for Memcached with Protocol Buffers support.
It is a part of Homework 17 for the OTUS course and is intended for educational purposes.

## Key Features

- Concurrent batch processing
- Protocol Buffers serialization
- Configurable batch parameters:
  - Batch size (default: 100 items)
  - Worker count (default: 4 threads)
  - Connection timeout (default: 3s)
- Dry-run mode for validation
- Graceful error handling

## Quick Start

```bash
# Clone repository
git clone https://github.com/speculzzz/go_memcache_loader.git
cd go_memcache_loader

# Install dependencies
go mod download

# Build and run
go build -o loader main.go
./loader --pattern sample.tsv.gz
```

## Configuration Options

```text
--idfa       Memcached address for IDFA devices (default: 127.0.0.1:33013)
--gaid       Memcached address for GAID devices (default: 127.0.0.1:33014)
--batch-size Batch size for processing (default: 100)
--workers    Number of concurrent workers (default: 4)
--dry        Enable dry-run mode (no writes to Memcached)
```

## Data Format

Input files should be gzipped TSV with format:
```text
<device_type>\t<device_id>\t<lat>\t<lon>\t<app1,app2,...>
```

## License

MIT License. See [LICENSE](LICENSE) for details.

## Author

- **speculzzz** (speculzzz@gmail.com)

---

Feel free to use it!
