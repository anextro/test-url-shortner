.PHONY: up down test test-service test-client test-e2e

up:
	docker compose up -d --build --wait

down:
	docker compose down -v

test-service:
	$(MAKE) -C service test

test-client:
	$(MAKE) -C client test

test-e2e:
	$(MAKE) up
	node e2e/e2e.test.js; status=$$?; $(MAKE) down; exit $$status

test: test-service test-client test-e2e
