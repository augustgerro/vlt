BINARY_NAME=vlt
INSTALL_PATH=${HOME}/go/bin

build:
	go build -o ${BINARY_NAME} main.go

install: build
	mv ${BINARY_NAME} ${INSTALL_PATH}/

clean:
	rm -f ${BINARY_NAME}

test:
	./${BINARY_NAME} "ls -la"
