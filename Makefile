NAME=mini-ss
VERSION=1.0.0
BINDIR=bin
GOBUILD=CGO_ENABLED=0 go build -ldflags '-X github.com/josexy/mini-ss/cmd.Version=$(VERSION) -w -s -buildid='
PACKAGE=cmd/miniss/miniss.go

all: linux-amd64 linux-arm64 macos-amd64 macos-arm64 win64

linux-amd64:
	GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(PACKAGE)

linux-arm64:
	GOARCH=arm64 GOOS=linux $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(PACKAGE)

macos-amd64:
	GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(PACKAGE)

macos-arm64:
	GOARCH=arm64 GOOS=darwin $(GOBUILD) -o $(BINDIR)/$(NAME)-$@ $(PACKAGE)

win64:
	GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(BINDIR)/$(NAME)-$@.exe $(PACKAGE)

releases: linux-amd64 linux-arm64 macos-amd64 macos-arm64 win64
	chmod +x $(BINDIR)/$(NAME)-*
	tar czf $(BINDIR)/$(NAME)-linux-amd64.tgz -C $(BINDIR) $(NAME)-linux-amd64
	tar czf $(BINDIR)/$(NAME)-linux-arm64.tgz -C $(BINDIR) $(NAME)-linux-arm64
	gzip $(BINDIR)/$(NAME)-linux-amd64
	gzip $(BINDIR)/$(NAME)-linux-arm64
	gzip $(BINDIR)/$(NAME)-macos-amd64
	gzip $(BINDIR)/$(NAME)-macos-arm64
	zip -m -j $(BINDIR)/$(NAME)-win64.zip $(BINDIR)/$(NAME)-win64.exe

clean:
	rm $(BINDIR)/*
