VERSION = $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
OSARCH=$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)

KMFDDM=\
	kmfddm-darwin-arm64 \
	kmfddm-darwin-amd64 \
	kmfddm-linux-amd64 \
	kmfddm-windows-amd64.exe

SUPPLEMENTAL=\
	tools/*.sh \
	tools/ideclr.py

my: kmfddm-$(OSARCH)

docker: kmfddm-linux-amd64

$(KMFDDM): cmd/kmfddm
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

kmfddm-%-$(VERSION).zip: kmfddm-% $(SUPPLEMENTAL)
	rm -rf $@ $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	echo $^ | xargs -n 1 | cpio -pdmu $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

kmfddm-%-$(VERSION).zip: kmfddm-%.exe $(SUPPLEMENTAL)
	rm -rf $@ $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	echo $^ | xargs -n 1 | cpio -pdmu $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

clean:
	rm -f kmfddm-* kmfddm-*.zip

release: \
	kmfddm-darwin-amd64-$(VERSION).zip \
	kmfddm-darwin-arm64-$(VERSION).zip \
	kmfddm-linux-amd64-$(VERSION).zip

test:
	go test -v -cover -race ./...

.PHONY: my docker $(KMFDDM) clean release test
