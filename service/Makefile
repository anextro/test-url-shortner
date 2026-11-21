SHELL := /bin/bash
.PHONY: test build docker-build

PG_IMAGE ?= postgres:16-alpine

build:
	go build -o /dev/null .

docker-build:
	docker build -t url-shortener-service .

# Starts a throwaway Postgres on a random free host port (Docker picks it),
# runs all tests, and always removes the container afterwards.
test:
	@set -eo pipefail; \
	CID=$$(docker run -d --rm -e POSTGRES_PASSWORD=test -e POSTGRES_DB=links -p 127.0.0.1::5432 $(PG_IMAGE)); \
	trap 'docker stop -t 1 '"$$CID"' >/dev/null 2>&1 || true' EXIT INT TERM; \
	PORT=$$(docker port $$CID 5432/tcp | head -n1 | sed 's/.*://'); \
	echo "postgres container $${CID:0:12} on 127.0.0.1:$$PORT"; \
	for i in $$(seq 1 60); do \
	  docker exec $$CID pg_isready -U postgres -d links -h 127.0.0.1 >/dev/null 2>&1 && break; sleep 0.5; \
	done; \
	docker exec $$CID pg_isready -U postgres -d links -h 127.0.0.1 >/dev/null; \
	go vet ./...; \
	TEST_DATABASE_URL="postgres://postgres:test@127.0.0.1:$$PORT/links?sslmode=disable" \
	  go test -count=1 -p 1 -v ./... 2>&1 | grep -Ev '^=== (RUN|PAUSE|CONT)'
