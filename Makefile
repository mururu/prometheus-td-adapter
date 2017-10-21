.PHONY: all deps clean

TARGET=prometheus-td-adapter

all: $(TARGET)

$(TARGET):
	go build -o $@ main.go

deps:
	dep ensure

test:
	go test -v ./td

clean:
	rm -f $(TARGET)
