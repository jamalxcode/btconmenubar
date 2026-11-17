#!/bin/sh
# Builds dist/BTCMenuBar.app (universal: Apple Silicon + Intel) and dist/BTCMenuBar.zip. Run on macOS.
#   scripts/build-app.sh v1.2.3
set -eu
VERSION="${1:-dev}"
APP=dist/BTCMenuBar.app
rm -rf dist
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
export CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=11.0
for arch in arm64 amd64; do
  carch=$arch; [ "$arch" = amd64 ] && carch=x86_64
  GOOS=darwin GOARCH=$arch CC="clang -arch $carch" \
    go build -trimpath -ldflags "-s -w -X main.version=$VERSION" -o "dist/btconmenubar-$arch" .
done
lipo -create -output "$APP/Contents/MacOS/btconmenubar" dist/btconmenubar-arm64 dist/btconmenubar-amd64
rm dist/btconmenubar-arm64 dist/btconmenubar-amd64
sed "s/__VERSION__/${VERSION#v}/g" packaging/Info.plist > "$APP/Contents/Info.plist"
codesign --force --deep --sign - "$APP"     # ad-hoc signature (Apple Silicon refuses unsigned code)
( cd dist && ditto -c -k --keepParent BTCMenuBar.app BTCMenuBar.zip )
lipo -info "$APP/Contents/MacOS/btconmenubar"
echo "Built $APP ($VERSION)"
