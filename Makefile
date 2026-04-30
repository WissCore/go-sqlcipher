.PHONY: all test update-sqlcipher update-modules

all:
	go build -v ./...

test:
	go test -race -count=1 ./...

# Refresh the vendored SQLCipher amalgamation. Usage:
#   make update-sqlcipher VERSION=4.15.0
update-sqlcipher:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make update-sqlcipher VERSION=<sqlcipher-tag>" >&2; \
		exit 1; \
	fi
	scripts/update-vendored.sh $(VERSION)

update-modules:
	go get -u
	go mod tidy -v
