#!/bin/bash

[ -z "$GOLANG_VERSION" ] && GOLANG_VERSION="1.25.0"
[ -z "$GOST_VERSION" ] && GOST_VERSION="3.2.4"
[ -z "$ANDROID_NDK_ROOT" ] && NDK_VERSION="r26c"

set -e

pushd $( cd "$( dirname "$0"  )" && pwd ) > /dev/null

if [ ! -d ".go_build" ]
then
  mkdir ".go_build"
fi
pushd ".go_build" > /dev/null

export GOCACHE="${GOCACHE:-$PWD/cache}"
export GOPATH="${GOPATH:-$PWD/gopath}"

if [ ! -e go ]
then
  GOLANG_RELEASE="go${GOLANG_VERSION}.linux-amd64.tar.gz"
  # GOLANG_DOWNLOAD="Go.v${GOLANG_VERSION}.Linux.x64_p30download.com.tar.gz"
  GOLANG_DOWNLOAD=$GOLANG_RELEASE
  # GOLANG_URL="https://pdn.sharezilla.ir/d/software/${GOLANG_DOWNLOAD}"
  GOLANG_URL="https://go.dev/dl/${GOLANG_RELEASE}"
  echo "GO was not detected"
  [ -f $GOLANG_DOWNLOAD ] || echo "Downloading '$GOLANG_RELEASE' ($GOLANG_URL) ..."
  [ -f $GOLANG_DOWNLOAD ] || curl $GOLANG_URL -LO
  [ -f $GOLANG_DOWNLOAD ] || mv -n $GOLANG_DOWNLOAD $GOLANG_RELEASE
  echo "Extracting $GOLANG_DOWNLOAD ..."
  tar -zxf $GOLANG_RELEASE || exit $?
  pushd go > /dev/null
  patch -p1 -r . < ../../gost/go.patch
  popd > /dev/null
fi

export PATH=$PWD/go/bin:$PATH
export GOROOT=$PWD/go
echo "Current GO version: $(go version|grep -oP "\d.*")"

if [ ! -e gost ]
then
  GOST_RELEASE="gost_v${GOST_VERSION}.tar.gz"
  GOST_URL="https://github.com/go-gost/gost/archive/refs/tags/v${GOST_VERSION}.tar.gz"
  echo "GOST was not detected"
  [ -f $GOST_RELEASE ] || echo "Downloading '$GOST_RELEASE' ($GOST_URL) ..."
  [ -f $GOST_RELEASE ] || curl $GOST_URL -Lo $GOST_RELEASE
  echo "Extracting $GOST_RELEASE ..."
  tar -zxf $GOST_RELEASE || exit $?
  mv gost-* gost
fi

if [ ! -d "gost/gost_helper" ]; then
  patch -d gost --no-backup-if-mismatch -tlNp1 < ../gost/gost.patch
fi
mkdir -p gost/gost_helper
cp ../gost/gost_helper/* gost/gost_helper/
echo "GOST version: v$GOST_VERSION"

IS_NDK_MISSING=true
if [ ! -z "$ANDROID_NDK_ROOT" ]
then
  IS_NDK_MISSING=$([[ ! -d $ANDROID_NDK_ROOT || -z "$(find $ANDROID_NDK_ROOT -iname "*clang" -print -quit)" ]] && echo true || echo false);
fi

if $IS_NDK_MISSING && [ -d ~+/ndk ]
then
  ANDROID_NDK_ROOT=$(find ~+/ndk -maxdepth 1 -mindepth 1 -type d -print -quit);
  IS_NDK_MISSING=$([[ ! -d $ANDROID_NDK_ROOT || -z "$(find $ANDROID_NDK_ROOT -iname "*clang" -print -quit)" ]] && echo true || echo false);
fi

if $IS_NDK_MISSING
then
  NDK_BASE="android-ndk-${NDK_VERSION}"
  NDK_RELEASE="${NDK_BASE}-linux.zip"
  NDK_URL="https://dl.google.com/android/repository/$NDK_RELEASE"
  echo "No NDK could be detected, downloading '$NDK_RELEASE' ($NDK_URL) ..."
  curl $NDK_URL -L -O
  unzip $NDK_RELEASE > /dev/null || exit $?
  pushd $NDK_BASE > /dev/null
  export ANDROID_NDK_ROOT=$PWD
  popd > /dev/null
fi
echo "Android NDK root: $ANDROID_NDK_ROOT"

bash ../antidpi/build.android.sh "$ANDROID_NDK_ROOT"

pushd gost > /dev/null

src="./cmd/gost"
start=$SECONDS
abis=(armeabi-v7a arm64-v8a x86 x86_64)
arches=(arm arm64 386 amd64)
triples=(armv7a-linux-androideabi aarch64-linux-android i686-linux-android x86_64-linux-android)
for i in "${!abis[@]}"; do
  output="../../app/src/main/jniLibs/${abis[$i]}/libgost-plugin.so"
  mkdir -p "$(dirname "$output")"
  compiler="$ANDROID_NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/bin/${triples[$i]}26-clang"
  CC="$compiler" GOOS=android GOARCH="${arches[$i]}" GOARM=7 CGO_ENABLED=1 \
    go build -buildvcs=false -ldflags "-s -w" -o "$output" "$src"
done

duration=$(( $SECONDS - $start ))
echo " + Finished: $(date) [Took $duration seconds]"

exit 0;
