GH_USER=hired-varied
PROJ_NAME=stupid-proxy
BINDIR=bin
VERSION=$(shell git describe --tags || echo "unknown version")
BUILDTIME=$(shell date -u)
GOBUILD=CGO_ENABLED=0 go build -ldflags '-X "github.com/$(GH_USER)/$(PROJ_NAME)/constant.Version=$(VERSION)" \
		-X "github.com/$(GH_USER)/$(PROJ_NAME)/constant.BuildTime=$(BUILDTIME)" \
		-w -s'

PLATFORM_LIST = \
	macos-amd64 \
	linux-386 \
	linux-amd64 \
	linux-armv5 \
	linux-armv6 \
	linux-armv7 \
	linux-armv8 \
	linux-mips-softfloat \
	linux-mips-hardfloat \
	linux-mipsle \
	linux-mipsle-softfloat \
	linux-mips64 \
	linux-mips64le \
	freebsd-386 \
	freebsd-amd64

WINDOWS_ARCH_LIST = \
	windows-386 \
	windows-amd64

define compile_both
	$(GOBUILD) -o $(BINDIR)/$(PROJ_NAME)-server-$(1) server/main.go
	$(GOBUILD) -o $(BINDIR)/$(PROJ_NAME)-client-$(1) client/main.go
endef

all: linux-amd64 # Most used

macos-amd64:
	GOARCH=amd64 GOOS=darwin $(call compile_both,$@)

linux-386:
	GOARCH=386 GOOS=linux $(call compile_both,$@)

linux-amd64:
	GOARCH=amd64 GOOS=linux $(call compile_both,$@)

linux-armv5:
	GOARCH=arm GOOS=linux GOARM=5 $(call compile_both,$@)

linux-armv6:
	GOARCH=arm GOOS=linux GOARM=6 $(call compile_both,$@)

linux-armv7:
	GOARCH=arm GOOS=linux GOARM=7 $(call compile_both,$@)

linux-armv8:
	GOARCH=arm64 GOOS=linux $(call compile_both,$@)

linux-mips-softfloat:
	GOARCH=mips GOMIPS=softfloat GOOS=linux $(call compile_both,$@)

linux-mips-hardfloat:
	GOARCH=mips GOMIPS=hardfloat GOOS=linux $(call compile_both,$@)

linux-mipsle:
	GOARCH=mipsle GOOS=linux $(call compile_both,$@)

linux-mipsle-softfloat:
	GOARCH=mipsle GOMIPS=softfloat GOOS=linux $(call compile_both,$@)

linux-mips64:
	GOARCH=mips64 GOOS=linux $(call compile_both,$@)

linux-mips64le:
	GOARCH=mips64le GOOS=linux $(call compile_both,$@)

freebsd-386:
	GOARCH=386 GOOS=freebsd $(call compile_both,$@)

freebsd-amd64:
	GOARCH=amd64 GOOS=freebsd $(call compile_both,$@)

windows-386:
	GOARCH=386 GOOS=windows $(call compile_both,$@.exe)

windows-amd64:
	GOARCH=amd64 GOOS=windows $(call compile_both,$@.exe)

gz_releases=$(addsuffix .gz, $(PLATFORM_LIST))
zip_releases=$(addsuffix .zip, $(WINDOWS_ARCH_LIST))

$(gz_releases): %.gz : %
	chmod +x $(BINDIR)/$(PROJ_NAME)-server-$(basename $@)
	chmod +x $(BINDIR)/$(PROJ_NAME)-client-$(basename $@)
	gzip -f -S -$(VERSION).gz $(BINDIR)/$(PROJ_NAME)-server-$(basename $@)
	gzip -f -S -$(VERSION).gz $(BINDIR)/$(PROJ_NAME)-client-$(basename $@)

$(zip_releases): %.zip : %
	zip -m -j $(BINDIR)/$(PROJ_NAME)-server-$(basename $@)-$(VERSION).zip $(BINDIR)/$(PROJ_NAME)-server-$(basename $@).exe
	zip -m -j $(BINDIR)/$(PROJ_NAME)-client-$(basename $@)-$(VERSION).zip $(BINDIR)/$(PROJ_NAME)-client-$(basename $@).exe

all-arch: $(PLATFORM_LIST) $(WINDOWS_ARCH_LIST)

releases: $(gz_releases) $(zip_releases)

clean:
	rm -f $(BINDIR)/*

