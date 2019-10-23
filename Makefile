default:
	@export GOPATH=$$GOPATH:$$(pwd) && go install aws-utils
edit:
	@export GOPATH=$$GOPATH:$$(pwd) && atom .
edit2:
	@export GOPATH=$$GOPATH:$$(pwd) && code .
edit3:
	@export GOPATH=$$GOPATH:$$(pwd) && /home/tal/programs/idea/bin/idea.sh .
run: default
	@bin/aws-utils
	@echo ""
clean:
	@rm -rf bin
test:
	@export GOPATH=$$GOPATH:$$(pwd) && go test ./...
test_ver:
	@export GOPATH=$$GOPATH:$$(pwd) && go test -v ./...
setup:
	go get gopkg.in/mgo.v2
	go get -u github.com/aws/aws-sdk-go/...
	go get github.com/mxk/go-sqlite/sqlite3
	go get golang.org/x/crypto/ssh
	go get github.com/bsphere/le_go
merge:
	git checkout master && git merge dev && git checkout dev && git push origin --all
