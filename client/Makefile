.PHONY: test install build

node_modules: package.json package-lock.json
	npm ci
	@touch node_modules

install: node_modules

test: node_modules
	CI=1 npx vitest run

build: node_modules
	npm run build
