.PHONY: all format clean clean-all dep-init build-path

PROJECT_ROOT:=$(shell pwd)
export GOPATH:=$(PROJECT_ROOT)/build

BINARIES:=bin/linker-gate

all: format bin/linker-gate

format: 
	go fmt github.com/linker-im/server/main/linker-gate
	go fmt github.com/linker-im/server/gate

bin/linker-gate: build-path
	go install -v -gcflags='all=-N -l' github.com/linker-im/server/main/linker-gate

# Common rules
build-path:
	@ENSURE_DIRS="bin build";					\
	for dir in $$ENSURE_DIRS; do				\
		if [ -e "$$dir" ]; then 				\
			if ! [ -d "$$dir" ]; then			\
				echo $$dir occupied.; 			\
				exit 1;							\
			fi;									\
		else									\
			mkdir $$dir;						\
		fi;										\
	done
	@if [ -e "build/bin" ]; then				\
		if ! [ -h "build/bin" ]; then			\
			echo build/bin occupied.;			\
		else									\
			rm build/bin;						\
		fi;										\
	fi
	@ln -s $$(pwd)/bin build/bin

clean:
	@if [ -e "build" ] && [ -d "build" ]; then \
		rm build -rf;							\
	fi

clean-all: clean
	@for binary in ${BINARIES}; do 	\
		if [ -e $$binary ]; then 	\
			rm $$binary;			\
			echo Remove $$binary; 	\
		fi;							\
	 done

