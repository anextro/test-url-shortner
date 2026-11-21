.PHONY: up down test test-service test-client test-e2e test-clean

# Every test runs in Docker: only Docker is needed.
TEST = docker compose -f docker-compose.test.yml

up:
	docker compose up -d --build --wait

down:
	docker compose down -v

test-service:
	$(TEST) run --rm service-tests; s=$$?; $(TEST) down; exit $$s

test-client:
	$(TEST) run --rm client-tests; s=$$?; $(TEST) down; exit $$s

test-e2e:
	$(TEST) --profile e2e run --rm e2e; s=$$?; $(TEST) --profile e2e down; exit $$s

test: test-service test-client test-e2e

# Also removes the cached dependencies (Go modules, node_modules volumes).
test-clean:
	$(TEST) --profile e2e down -v
