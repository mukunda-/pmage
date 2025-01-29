
.PHONY: all clean sneskit_install

all:
	go build -o build/

clean:
	rm -rf build/

sneskit_install: all
ifeq ($(strip $(SNESKIT)),)
	$(error SNESKIT path not found, please add it to your environment)
endif
	cp build/pmage* $(SNESKIT)/bin/
