#!/bin/sh

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

run_genheap() {
    javac com/myco/GenHeap.java
    java com.myco.GenHeap &
    pid=$!
    sleep 2
    jmap -dump:format=b,file=genheap.hprof $pid
    kill $pid
}

warn() { echo "$*" >&2; }
die() { warn "$*"; exit 1; }

main "$@"

