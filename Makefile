release-build: build
	docker build -t timsimmons/rtl_433_exporter:0.0.1 .

release-push:
	docker push timsimmons/rtl_433_exporter:0.0.1

docker-build:
	docker build .

build: clean
	go build -o rtl_433_exporter cmd/rtl_433_exporter/main.go

clean:
	rm rtl_433_exporter || true
