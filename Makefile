GOPATH := $(shell cd ../../../../; pwd)
run:
	go build . && ./go-tilesetter api web/ui
