pkgname=gcal-notify
pkgver=0.0.1
pkgrel=1
pkgdesc='Google Calendar notifier'
arch=('x86_64')
url="https://github.com/svenschwermer/$pkgname"
license=('GPL')
makedepends=('go')

prepare(){
  mkdir -p build/
}

build() {
  export CGO_CPPFLAGS="${CPPFLAGS}"
  export CGO_CFLAGS="${CFLAGS}"
  export CGO_CXXFLAGS="${CXXFLAGS}"
  export CGO_LDFLAGS="${LDFLAGS}"
  export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw"
  go build -o build ./cmd/$pkgname
}

check() {
  go test ./...
}

package() {
  install -Dm755 -t "$pkgdir/usr/bin/" build/$pkgname 
  install -Dm644 -t "$pkgdir/usr/lib/systemd/user/" $pkgname.service
}
