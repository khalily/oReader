.PHONY: install test lint fmt

install:
	npm install

test:
	npm test -- --run

lint:
	npm run lint

fmt:
	npx prettier --write "src/**/*.{ts,tsx}"
