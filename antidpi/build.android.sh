#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
ndk="${1:?Android NDK path required}"
toolchain="$ndk/toolchains/llvm/prebuilt/linux-x86_64/bin"
abis=(armeabi-v7a arm64-v8a x86 x86_64)
triples=(armv7a-linux-androideabi aarch64-linux-android i686-linux-android x86_64-linux-android)
for i in "${!abis[@]}"; do
    out="$root/app/src/main/jniLibs/${abis[$i]}"
    mkdir -p "$out"
    sources=()
    for source in packets main conev proxy desync mpool extend; do
        sources+=("$root/antidpi/vendor/$source.c")
    done
    "$toolchain/${triples[$i]}26-clang" -D_DEFAULT_SOURCE -std=c99 -O2 -fPIE -pie \
        -I"$root/antidpi/vendor" "${sources[@]}" -o "$out/libbyedpi.so"
    "$toolchain/llvm-strip" "$out/libbyedpi.so"
done
