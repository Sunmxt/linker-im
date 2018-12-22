.PHONY: all format clean clean-all

export GOPATH:=$(shell pwd)

all: format bin/linker-gate

src:
	@ln -s ./ src

bin/linker-gate: src
	go install -v -gcflags='all=-N -l' server/main/linker-gate

format: src
	go fmt server/main/linker-gate
	go fmt server/gate

clean:
	@if [ -h ./src ]; then rm ./src; fi 

clean-all: clean
	@if [ -e bin/linker-gate ]; then rm bin/linker-gate; fi
