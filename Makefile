BINARY=s3sync

all:
	go build ${LDFLAGS} .

dev:
	go fmt .
	go vet .

install: dev
	go install .

release: dev
	GOOS=linux GOARCH=amd64 go build -o ${BINARY}_linux_amd64 .
	GOOS=linux GOARCH=arm GOARM=5 go build -o ${BINARY}_linux_arm5 .
	GOOS=windows GOARCH=amd64 go build -o ${BINARY}_windows_amd64 .
	GOOS=darwin GOARCH=amd64 go build -o ${BINARY}_darwin_amd64 .

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	if [ -f ${BINARY}_linux_amd64 ] ; then rm ${BINARY}_linux ; fi
	if [ -f ${BINARY}_linux_arm5] ; then rm ${BINARY}_darwin ; fi
	if [ -f ${BINARY}_windows_amd64 ] ; then rm ${BINARY}_windows ; fi
	if [ -f ${BINARY}_darwin_amd64 ] ; then rm ${BINARY}_darwin ; fi

test: dev
	go test -v -race .
# install errcheck with `go get -u github.com/kisielk/errcheck`
	errcheck -ignoretests .
# install golint with `go get -u github.com/golang/lint/golint`
	golint .
# install varcheck with `go get -u github.com/opennota/check/cmd/varcheck`
	varcheck .
# install with `go get -u honnef.co/go/tools/cmd/gosimple`
	gosimple .

cover: dev
	go test -covermode=count -coverprofile=cover.out .
	go tool cover -html=cover.out


