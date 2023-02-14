COMMIT-ID := $(shell git rev-parse HEAD)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H-%M-%SZ')

GOLDFLAGS += -X main.CommitID=$(COMMIT-ID)
GOLDFLAGS += -X main.BuildTime=$(BUILDTIME)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

go-pages: git.go  main.go  tools.go  wiki.go
	go build -o go-pages $(GOFLAGS) .

all:
	go build -o go-pages $(GOFLAGS) .

strip: go-pages
	strip go-pages

run:
	./go-pages

README.html: README.md
	pandoc -o README.html README.md

clean:
	rm -f go-pages README.html
