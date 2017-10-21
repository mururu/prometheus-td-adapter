TARGET := prometheus-td-adapter
VERSION := 0.1.0

.PHONY: all deps clean

all: $(TARGET)

$(TARGET):
	go build -ldflags "-X main.version=$(VERSION)" -o $@ main.go

deps:
	dep ensure

test:
	go test -v ./td

clean:
	rm -f $(TARGET)
