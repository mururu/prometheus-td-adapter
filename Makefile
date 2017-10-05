.PHONY: all deps clean

TARGET=prometheus-td-adapter

all: $(TARGET)

$(TARGET):
	go build -o $@ main.go

deps:
	dep ensure

clean:
	rm -f $(TARGET)
