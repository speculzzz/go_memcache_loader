CURDIR := $(shell pwd)
PATH := $(CURDIR)/tools/go/bin:$(CURDIR)/tools/protoc/bin:$(PATH)

DATA_URLS := \
    https://cloud.mail.ru/public/KxgQ/UjzmvrKzN \
    https://cloud.mail.ru/public/2Gby/6RtFfLyKD
DATA_DIR := ./data/appsinstalled
DATA_FILE_MASK = $(DATA_DIR)/*.tsv.gz

check-data:
	@echo "Checking for .tsv.gz files in $(DATA_DIR)..."
	@if ls $(DATA_FILE_MASK) 1> /dev/null 2>&1; then \
		echo "Found these data files:"; \
		ls -lh $(DATA_FILE_MASK); \
	else \
		echo "\nERROR: No .tsv.gz files found in $(DATA_DIR)"; \
		echo "Please download these files manually:"; \
		for url in $(DATA_URLS); do \
			echo "  $$url"; \
		done; \
		echo "\nPlace the downloaded .tsv.gz files into $(DATA_DIR) directory"; \
		echo "Expected at least one file matching: $(DATA_FILE_MASK)"; \
		exit 1; \
	fi

generate:
	protoc --plugin=protoc-gen-go=./tools/protoc-gen-go \
		--go_out=. --go_opt=paths=source_relative \
		internal/proto/*.proto

build: generate
	go build -o bin/app main.go

run: generate check-data
	scripts/memcached-ctl restart
	go run main.go

tidy:
	go mod tidy

selftest:
	go run main.go --test

test:
	go run main.go --dry --pattern sample.tsv.gz
	@if [ -f ".sample.tsv.gz" ]; then \
		mv .sample.tsv.gz sample.tsv.gz; \
	fi;

.PHONY: check-data generate build run tidy test seftest
