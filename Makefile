PREFIX = /usr/local
CFLAGS = $(shell . /etc/makepkg.conf; echo $${CFLAGS})
CPPFLAGS = $(shell . /etc/makepkg.conf; echo $${CPPFLAGS})
CXXFLAGS = $(shell . /etc/makepkg.conf; echo $${CXXFLAGS})
LDFLAGS = $(shell . /etc/makepkg.conf; echo $${LDFLAGS})

gcal-notify:
	CGO_CFLAGS="$(CFLAGS)" \
	CGO_CPPFLAGS="$(CPPFLAGS)" \
	CGO_CXXFLAGS="$(CXXFLAGS)" \
	CGO_LDFLAGS="$(LDFLAGS)" \
	GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw" \
	go build ./cmd/$@

clean:
	rm -f gcal-notify

install:
	install -Dm755 -t $(PREFIX)/bin gcal-notify
	install -Dm644 -t $(PREFIX)/lib/systemd/user contrib/systemd/gcal-notify.service
	sed -i 's|@PREFIX@|$(PREFIX)|g' $(PREFIX)/lib/systemd/user/gcal-notify.service

uninstall:
	rm -f $(PREFIX)/bin/gcal-notify
	rm -f $(PREFIX)/lib/systemd/user/gcal-notify.service

.PHONY: gcal-notify clean install uninstall
