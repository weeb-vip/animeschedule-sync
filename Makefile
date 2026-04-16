build:
	go build -o animeschedule-sync ./cmd/main.go

sync:
	go run cmd/main.go sync

serve:
	go run cmd/main.go serve

tidy:
	go mod tidy
