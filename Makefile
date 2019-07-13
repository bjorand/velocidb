test:
	go test -v ./...

dep:
	go get -u github.com/golang/dep/cmd/dep
	dep ensure -v

build:
	go build -v

install:
	go install -v

test_cluster_rebuild:
	docker-compose build

test_cluster_start:
	docker-compose up
