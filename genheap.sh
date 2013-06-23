#!/bin/sh

set -e

tmp=/tmp/genheap.$$
trap "rm -f $tmp" 0 ERR

main() {
    verify_java
    run_genheap
}

verify_java() {
    if [ -z "$JAVA_HOME" ]; then
        die "JAVA_HOME is not set" 
    fi
    export PATH="${JAVA_HOME}/bin:${PATH}"
    case "$(java -version 2>&1)" in
        *version\ \"1.[67]*) ;;
        *) die "Need JAVA_HOME to be JVM version 1.6 or 1.7" ;;
    esac
}

Huge=50000000 
Small=10000
Size=$Small

run_genheap() {
    javac com/myco/GenHeap.java
    rm -f genheap.hprof
    java -Xmx10g -verbose:gc -XX:+UseConcMarkSweepGC com.myco.GenHeap $Size 2>$tmp &
    pid=$!
    while true; do
        if grep ready $tmp; then
            jmap -dump:format=b,file=genheap.hprof $pid
            kill $pid
            return
        fi
    done
}

warn() { echo "$*" >&2; }
die() { warn "$*"; exit 1; }

main "$@"

