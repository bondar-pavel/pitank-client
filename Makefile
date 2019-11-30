build:
	go build -tags camera -o pitank-client

build-arm:
	GOOS=linux GOARCH=arm GOARM=5 go build -tags camera -o pitank-client-arm

fmt:
	gofmt -w *.go

install:
	cp ./pitank-client-arm /usr/bin/pitank-client-arm
	cp ./pitank-client.service /usr/lib/systemd/system
	/usr/bin/systemctl daemon-reload

uninstall:
	rm /usr/bin/pitank-client-arm
	rm /usr/lib/systemd/system/pitank-client.service