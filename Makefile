build:
	go build -o pitank_client

build-arm:
	GOOS=linux GOARCH=arm GOARM=5 go build -o pitank_client_arm

fmt:
	gofmt -w *.go
