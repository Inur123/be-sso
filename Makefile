# Makefile untuk SSO Backend — pelajarnumagetan.or.id

.PHONY: run build dev tidy migrate

# Jalankan server (development)
run:
	go run ./cmd/server

# Build binary
build:
	go build -o bin/sso ./cmd/server

# Tidy dependencies
tidy:
	go mod tidy

# Bersihkan binary
clean:
	rm -rf bin/

# Cek semua package
check:
	go build ./...

# Format kode
fmt:
	go fmt ./...

# Vet
vet:
	go vet ./...
