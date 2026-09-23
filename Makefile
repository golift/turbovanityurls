# This is the same logic found in settings.sh.
DATE="$(shell date -u +%Y-%m-%dT%H:%M:00Z)"
COMMIT="$(shell git rev-parse --short HEAD 2>/dev/null || echo 0)"
BRANCH="$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo ${GITHUB_REF_NAME})"
MAINT=David Newhall II <captain at golift dot io>
DESC=HTTP Server providing vanity go import paths.
LICENSE=Apache-2.0
SOURCE_URL=https://github.com/golift/turbovanityurls
VENDOR=Go Lift <code@golift.io>

# Some builds pass the version in. Local builds get it from the current git tag.
ifeq ($(VERSION),)
	VERSION=$(shell git describe --abbrev=0 --tags $(shell git rev-list --tags --max-count=1) | tr -d v)
endif
ifeq ($(VERSION),)
	VERSION=development
endif
ifeq ($(ITERATION),)
	ITERATION=$(shell git rev-list --count --all 2>/dev/null || echo 0)
endif

# If you put a signing key into GPG, set it here.
SIGNING_KEY=$(shell gpg --list-keys 2>/dev/null | grep -o B93DD66EF98E54E2EAE025BA0166AD34ABC5A57C)

define PACKAGE_ARGS
--after-install init/systemd/after-install.sh \
--before-remove init/systemd/before-remove.sh \
--name turbovanityurls \
--deb-no-default-config-files \
--rpm-os linux \
--deb-user turbovanityurls \
--rpm-user turbovanityurls \
--pacman-user turbovanityurls \
--iteration $(ITERATION) \
--license $(LICENSE) \
--url $(SOURCE_URL) \
--maintainer "$(MAINT)" \
--vendor "$(VENDOR)" \
--description "$(DESC)" \
--config-files "/etc/turbovanityurls/config.yaml" \
--freebsd-origin "$(SOURCE_URL)"
endef

# rpm is weird and changes - to _ in versions.
RPMVERSION:=$(shell echo $(VERSION) | tr -- - _)

VERSION_LDFLAGS:= -X \"main.Branch=$(BRANCH) ($(COMMIT))\" \
	-X \"main.BuildDate=$(DATE)\" \
	-X \"main.BuildUser=$(shell whoami)\" \
	-X \"main.Revision=$(ITERATION)\" \
	-X \"main.Version=$(VERSION)\"

# Makefile targets follow.

.PHONY: all clean build release man readme rsrc
all: clean build

####################
##### Releases #####
####################

# Prepare a release. Called in Travis CI.
release: clean linux_packages freebsd_packages windows
	# Prepareing a release!
	mkdir -p $@
	mv turbovanityurls.*.linux turbovanityurls.*.freebsd $@/
	gzip -9r $@/
	for i in turbovanityurls*.exe ; do zip -9qj $@/$$i.zip $$i examples/*.example *.html; rm -f $$i;done
	mv *.rpm *.deb *.txz $@/
	# Generating File Hashes
	openssl dgst -r -sha256 $@/* | sed 's#release/##' | tee $@/checksums.sha256.txt

# Delete all build assets.
clean:
	rm -f turbovanityurls turbovanityurls.*.{macos,freebsd,linux,exe}{,.gz,.zip} turbovanityurls.1{,.gz} turbovanityurls.rb
	rm -f turbovanityurls{_,-}*.{deb,rpm,txz} v*.tar.gz.sha256 examples/MANUAL .metadata.make rsrc_*.syso
	rm -f cmd/turbovanityurls/README{,.html} README{,.html} ./turbovanityurls_manual.html rsrc.syso
	rm -f pkg/bindata/bindata.go
	rm -rf package_build_* release

####################
##### Sidecars #####
####################

# Build a man page from a markdown file using md2roff.
# This also turns the repo readme into an html file.
# md2roff is needed to build the man file and html pages from the READMEs.
man: turbovanityurls.1.gz
turbovanityurls.1.gz:
	# Building man page. Build dependency first: md2roff
	$(shell go env GOPATH)/bin/md2roff --manual turbovanityurls --version $(VERSION) --date "$(DATE)" examples/MANUAL.md
	gzip -9nc examples/MANUAL > $@
	mv examples/MANUAL.html turbovanityurls_manual.html

# TODO: provide a template that adds the date to the built html file.
readme: README.html
README.html:
	# This turns README.md into README.html
	$(shell go env GOPATH)/bin/md2roff --manual turbovanityurls --version $(VERSION) --date "$(DATE)" README.md

rsrc: rsrc.syso
rsrc.syso: init/windows/application.ico init/windows/manifest.xml
	$(shell go env GOPATH)/bin/rsrc -arch amd64 -ico init/windows/application.ico -manifest init/windows/manifest.xml

####################
##### Binaries #####
####################

.PHONY: build linux linux386 arm arm64 armhf macos freebsd freebsd386 freebsdarm exe windows
build: turbovanityurls
turbovanityurls: main.go
	go build $(BUILD_FLAGS) -o turbovanityurls -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

linux: turbovanityurls.amd64.linux
turbovanityurls.amd64.linux: main.go
	# Building linux 64-bit x86 binary.
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

linux386: turbovanityurls.386.linux
turbovanityurls.386.linux: main.go
	# Building linux 32-bit x86 binary.
	GOOS=linux GOARCH=386 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

arm: arm64 armhf

arm64: turbovanityurls.arm64.linux
turbovanityurls.arm64.linux: main.go
	# Building linux 64-bit ARM binary.
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

armhf: turbovanityurls.arm.linux
turbovanityurls.arm.linux: main.go
	# Building linux 32-bit ARM binary.
	GOOS=linux GOARCH=arm GOARM=6 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

macos: turbovanityurls.universal.macos
turbovanityurls.universal.macos: turbovanityurls.amd64.macos turbovanityurls.arm64.macos
	# Building darwin 64-bit universal binary.
	lipo -create -output $@ turbovanityurls.amd64.macos turbovanityurls.arm64.macos
turbovanityurls.amd64.macos: main.go
	# Building darwin 64-bit x86 binary.
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 CGO_LDFLAGS=-mmacosx-version-min=10.8 CGO_CFLAGS=-mmacosx-version-min=10.8 go build -o $@ -ldflags "-v -w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "
turbovanityurls.arm64.macos: main.go
	# Building darwin 64-bit arm binary.
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 CGO_LDFLAGS=-mmacosx-version-min=10.8 CGO_CFLAGS=-mmacosx-version-min=10.8 go build -o $@ -ldflags "-v -w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "


freebsd: turbovanityurls.amd64.freebsd
turbovanityurls.amd64.freebsd: main.go
	GOOS=freebsd GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

freebsd386: turbovanityurls.i386.freebsd
turbovanityurls.i386.freebsd: main.go
	GOOS=freebsd GOARCH=386 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

freebsdarm: turbovanityurls.armhf.freebsd
turbovanityurls.armhf.freebsd: main.go
	GOOS=freebsd GOARCH=arm go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS) "

exe: turbovanityurls.amd64.exe
windows: turbovanityurls.amd64.exe
turbovanityurls.amd64.exe: rsrc.syso main.go
	# Building windows 64-bit x86 binary.
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ -ldflags "-w -s $(VERSION_LDFLAGS) $(EXTRA_LDFLAGS)"

####################
##### Packages #####
####################

.PHONY: linux_packages freebsd_packages rpm deb rpm386 deb386 rpmarm debarm rpmarmhf debarmhf freebsd_pkg freebsd386_pkg freebsdarm_pkg
linux_packages: rpm deb rpm386 deb386 debarm rpmarm debarmhf rpmarmhf

freebsd_packages: freebsd_pkg freebsd386_pkg freebsdarm_pkg

rpm: turbovanityurls-$(RPMVERSION)-$(ITERATION).x86_64.rpm
turbovanityurls-$(RPMVERSION)-$(ITERATION).x86_64.rpm: package_build_linux_rpm check_fpm
	@echo "Building 'rpm' package for turbovanityurls version '$(RPMVERSION)-$(ITERATION)'."
	fpm -s dir -t rpm $(PACKAGE_ARGS) -a x86_64 -v $(RPMVERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || rpmsign --key-id=$(SIGNING_KEY) --resign turbovanityurls-$(RPMVERSION)-$(ITERATION).x86_64.rpm

deb: turbovanityurls_$(VERSION)-$(ITERATION)_amd64.deb
turbovanityurls_$(VERSION)-$(ITERATION)_amd64.deb: package_build_linux_deb check_fpm
	@echo "Building 'deb' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t deb $(PACKAGE_ARGS) -a amd64 -v $(VERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || debsigs --default-key="$(SIGNING_KEY)" --sign=origin turbovanityurls_$(VERSION)-$(ITERATION)_amd64.deb

rpm386: turbovanityurls-$(RPMVERSION)-$(ITERATION).i386.rpm
turbovanityurls-$(RPMVERSION)-$(ITERATION).i386.rpm: package_build_linux_386_rpm check_fpm
	@echo "Building 32-bit 'rpm' package for turbovanityurls version '$(RPMVERSION)-$(ITERATION)'."
	fpm -s dir -t rpm $(PACKAGE_ARGS) -a i386 -v $(RPMVERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || rpmsign --key-id=$(SIGNING_KEY) --resign turbovanityurls-$(RPMVERSION)-$(ITERATION).i386.rpm

deb386: turbovanityurls_$(VERSION)-$(ITERATION)_i386.deb
turbovanityurls_$(VERSION)-$(ITERATION)_i386.deb: package_build_linux_386_deb check_fpm
	@echo "Building 32-bit 'deb' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t deb $(PACKAGE_ARGS) -a i386 -v $(VERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || debsigs --default-key="$(SIGNING_KEY)" --sign=origin turbovanityurls_$(VERSION)-$(ITERATION)_i386.deb

rpmarm: turbovanityurls-$(RPMVERSION)-$(ITERATION).aarch64.rpm
turbovanityurls-$(RPMVERSION)-$(ITERATION).aarch64.rpm: package_build_linux_arm64_rpm check_fpm
	@echo "Building 64-bit ARM8 'rpm' package for turbovanityurls version '$(RPMVERSION)-$(ITERATION)'."
	fpm -s dir -t rpm $(PACKAGE_ARGS) -a aarch64 -v $(RPMVERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || rpmsign --key-id=$(SIGNING_KEY) --resign turbovanityurls-$(RPMVERSION)-$(ITERATION).aarch64.rpm

debarm: turbovanityurls_$(VERSION)-$(ITERATION)_arm64.deb
turbovanityurls_$(VERSION)-$(ITERATION)_arm64.deb: package_build_linux_arm64_deb check_fpm
	@echo "Building 64-bit ARM8 'deb' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t deb $(PACKAGE_ARGS) -a arm64 -v $(VERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || debsigs --default-key="$(SIGNING_KEY)" --sign=origin turbovanityurls_$(VERSION)-$(ITERATION)_arm64.deb

rpmarmhf: turbovanityurls-$(RPMVERSION)-$(ITERATION).armhf.rpm
turbovanityurls-$(RPMVERSION)-$(ITERATION).armhf.rpm: package_build_linux_armhf_rpm check_fpm
	@echo "Building 32-bit ARM6/7 HF 'rpm' package for turbovanityurls version '$(RPMVERSION)-$(ITERATION)'."
	fpm -s dir -t rpm $(PACKAGE_ARGS) -a armhf -v $(RPMVERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || rpmsign --key-id=$(SIGNING_KEY) --resign turbovanityurls-$(RPMVERSION)-$(ITERATION).armhf.rpm

debarmhf: turbovanityurls_$(VERSION)-$(ITERATION)_armhf.deb
turbovanityurls_$(VERSION)-$(ITERATION)_armhf.deb: package_build_linux_armhf_deb check_fpm
	@echo "Building 32-bit ARM6/7 HF 'deb' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t deb $(PACKAGE_ARGS) -a armhf -v $(VERSION) -C $<
	[ "$(SIGNING_KEY)" = "" ] || debsigs --default-key="$(SIGNING_KEY)" --sign=origin turbovanityurls_$(VERSION)-$(ITERATION)_armhf.deb

freebsd_pkg: turbovanityurls-$(VERSION)_$(ITERATION).amd64.txz
turbovanityurls-$(VERSION)_$(ITERATION).amd64.txz: package_build_freebsd check_fpm
	@echo "Building 'freebsd pkg' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t freebsd $(PACKAGE_ARGS) -a amd64 -v $(VERSION) -p turbovanityurls-$(VERSION)_$(ITERATION).amd64.txz -C $<

freebsd386_pkg: turbovanityurls-$(VERSION)_$(ITERATION).i386.txz
turbovanityurls-$(VERSION)_$(ITERATION).i386.txz: package_build_freebsd_386 check_fpm
	@echo "Building 32-bit 'freebsd pkg' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t freebsd $(PACKAGE_ARGS) -a 386 -v $(VERSION) -p turbovanityurls-$(VERSION)_$(ITERATION).i386.txz -C $<

freebsdarm_pkg: turbovanityurls-$(VERSION)_$(ITERATION).armhf.txz
turbovanityurls-$(VERSION)_$(ITERATION).armhf.txz: package_build_freebsd_arm check_fpm
	@echo "Building 32-bit ARM6/7 HF 'freebsd pkg' package for turbovanityurls version '$(VERSION)-$(ITERATION)'."
	fpm -s dir -t freebsd $(PACKAGE_ARGS) -a arm -v $(VERSION) -p turbovanityurls-$(VERSION)_$(ITERATION).armhf.txz -C $<

# Build an environment that can be packaged for linux.
package_build_linux_rpm: readme man linux
	# Building package environment for linux.
	mkdir -p $@/usr/bin $@/etc/turbovanityurls $@/usr/share/man/man1 $@/usr/share/doc/turbovanityurls $@/usr/lib/turbovanityurls $@/var/log/turbovanityurls
	# Copying the binary, config file, unit file, and man page into the env.
	cp turbovanityurls.amd64.linux $@/usr/bin/turbovanityurls
	cp *.1.gz $@/usr/share/man/man1
	rm -f $@/usr/lib/turbovanityurls/*.so
	[ ! -f *amd64.so ] || cp *amd64.so $@/usr/lib/turbovanityurls/
	cp examples/config.yaml.example $@/etc/turbovanityurls/
	cp examples/config.yaml.example $@/etc/turbovanityurls/config.yaml
	cp LICENSE *.html examples/*?.?* $@/usr/share/doc/turbovanityurls/
	 mkdir -p $@/lib/systemd/system
	cp init/systemd/turbovanityurls.service $@/lib/systemd/system/
	[ ! -d "init/linux/rpm" ] || cp -r init/linux/rpm/* $@

# Build an environment that can be packaged for linux.
package_build_linux_deb: readme man linux
	# Building package environment for linux.
	mkdir -p $@/usr/bin $@/etc/turbovanityurls $@/usr/share/man/man1 $@/usr/share/doc/turbovanityurls $@/usr/lib/turbovanityurls $@/var/log/turbovanityurls
	# Copying the binary, config file, unit file, and man page into the env.
	cp turbovanityurls.amd64.linux $@/usr/bin/turbovanityurls
	cp *.1.gz $@/usr/share/man/man1
	rm -f $@/usr/lib/turbovanityurls/*.so
	[ ! -f *amd64.so ] || cp *amd64.so $@/usr/lib/turbovanityurls/
	cp examples/config.yaml.example $@/etc/turbovanityurls/
	cp examples/config.yaml.example $@/etc/turbovanityurls/config.yaml
	cp LICENSE *.html examples/*?.?* $@/usr/share/doc/turbovanityurls/
	mkdir -p $@/lib/systemd/system
	cp init/systemd/turbovanityurls.service $@/lib/systemd/system/
	[ ! -d "init/linux/deb" ] || cp -r init/linux/deb/* $@

package_build_linux_386_deb: package_build_linux_deb linux386
	mkdir -p $@
	cp -r $</* $@/
	[ ! -f *386.so ] || cp *386.so $@/usr/lib/turbovanityurls/
	cp turbovanityurls.386.linux $@/usr/bin/turbovanityurls

package_build_linux_arm64_deb: package_build_linux_deb arm64
	mkdir -p $@
	cp -r $</* $@/
	[ ! -f *arm64.so ] || cp *arm64.so $@/usr/lib/turbovanityurls/
	cp turbovanityurls.arm64.linux $@/usr/bin/turbovanityurls

package_build_linux_armhf_deb: package_build_linux_deb armhf
	mkdir -p $@
	cp -r $</* $@/
	[ ! -f *armhf.so ] || cp *armhf.so $@/usr/lib/turbovanityurls/
	cp turbovanityurls.arm.linux $@/usr/bin/turbovanityurls
package_build_linux_386_rpm: package_build_linux_rpm linux386
	mkdir -p $@
	cp -r $</* $@/
	[ ! -f *386.so ] || cp *386.so $@/usr/lib/turbovanityurls/
	cp turbovanityurls.386.linux $@/usr/bin/turbovanityurls

package_build_linux_arm64_rpm: package_build_linux_rpm arm64
	mkdir -p $@
	cp -r $</* $@/
	[ ! -f *arm64.so ] || cp *arm64.so $@/usr/lib/turbovanityurls/
	cp turbovanityurls.arm64.linux $@/usr/bin/turbovanityurls

package_build_linux_armhf_rpm: package_build_linux_rpm armhf
	mkdir -p $@
	cp -r $</* $@/
	[ ! -f *armhf.so ] || cp *armhf.so $@/usr/lib/turbovanityurls/
	cp turbovanityurls.arm.linux $@/usr/bin/turbovanityurls

# Build an environment that can be packaged for freebsd.
package_build_freebsd: readme man freebsd
	mkdir -p $@/usr/local/bin $@/usr/local/etc/turbovanityurls $@/usr/local/share/man/man1 $@/usr/local/share/doc/turbovanityurls $@/usr/local/var/log/turbovanityurls
	cp turbovanityurls.amd64.freebsd $@/usr/local/bin/turbovanityurls
	cp *.1.gz $@/usr/local/share/man/man1
	cp examples/config.yaml.example $@/usr/local/etc/turbovanityurls/
	cp examples/config.yaml.example $@/usr/local/etc/turbovanityurls/config.yaml
	cp LICENSE *.html examples/*?.?* $@/usr/local/share/doc/turbovanityurls/
	mkdir -p $@/usr/local/etc/rc.d
	cp init/bsd/freebsd.rc.d $@/usr/local/etc/rc.d/turbovanityurls
	chmod +x $@/usr/local/etc/rc.d/turbovanityurls

package_build_freebsd_386: package_build_freebsd freebsd386
	mkdir -p $@
	cp -r $</* $@/
	cp turbovanityurls.i386.freebsd $@/usr/local/bin/turbovanityurls

package_build_freebsd_arm: package_build_freebsd freebsdarm
	mkdir -p $@
	cp -r $</* $@/
	cp turbovanityurls.armhf.freebsd $@/usr/local/bin/turbovanityurls

.PHONY: check_fpm test lint docker install

check_fpm:
	@fpm --version > /dev/null || (echo "FPM missing. Install FPM: https://fpm.readthedocs.io/en/latest/installing.html" && false)

# Run code tests and lint.
test: lint
	# Testing.
	go test -race -covermode=atomic ./...
lint:
	# Checking lint.
	$(shell go env GOPATH)/bin/golangci-lint version
	GOOS=linux $(shell go env GOPATH)/bin/golangci-lint run
	GOOS=freebsd $(shell go env GOPATH)/bin/golangci-lint run

##################
##### Docker #####
##################

docker:
	docker buildx build --no-cache --load --pull --tag turbovanityurls \
		--build-arg "BUILD_DATE=$(DATE)" \
		--build-arg "COMMIT=$(COMMIT)" \
		--build-arg "VERSION=$(VERSION)-$(ITERATION)" \
		--build-arg "LICENSE=$(LICENSE)" \
		--build-arg "DESC=$(DESC)" \
		--build-arg "VENDOR=$(VENDOR)" \
		--build-arg "AUTHOR=$(MAINT)" \
		--build-arg "BINARY=turbovanityurls" \
		--build-arg "CONFIG_FILE=config.yaml" \
		--build-arg "BUILD_FLAGS=$(BUILD_FLAGS)" \
		--file init/docker/Dockerfile .

# Used for Homebrew only. Other distros can create packages.
install: man readme turbovanityurls
	@echo -  Done Building  -
	@echo -  Local installation with the Makefile is only supported on macOS.
	@echo -  Otherwise, build and install a package: make rpm -or- make deb
	@[ "$(shell uname)" = "Darwin" ] || (echo "Unable to continue, not a Mac." && false)
	@[ "$(PREFIX)" != "" ] || (echo "Unable to continue, PREFIX not set. Use: make install PREFIX=/usr/local ETC=/usr/local/etc" && false)
	@[ "$(ETC)" != "" ] || (echo "Unable to continue, ETC not set. Use: make install PREFIX=/usr/local ETC=/usr/local/etc" && false)
	# Copying the binary, config file, unit file, and man page into the env.
	/usr/bin/install -m 0755 -d $(PREFIX)/bin $(PREFIX)/share/man/man1 $(ETC)/turbovanityurls $(PREFIX)/share/doc/turbovanityurls $(PREFIX)/lib/turbovanityurls
	/usr/bin/install -m 0755 -cp turbovanityurls $(PREFIX)/bin/turbovanityurls
	/usr/bin/install -m 0644 -cp turbovanityurls.1.gz $(PREFIX)/share/man/man1
	/usr/bin/install -m 0644 -cp examples/config.yaml.example $(ETC)/turbovanityurls/
	[ -f $(ETC)/turbovanityurls/config.yaml ] || /usr/bin/install -m 0644 -cp  examples/config.yaml.example $(ETC)/turbovanityurls/config.yaml
	/usr/bin/install -m 0644 -cp LICENSE *.html examples/* $(PREFIX)/share/doc/turbovanityurls/
