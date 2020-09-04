#!/bin/bash

BASE=$(dirname $(readlink -f $0))
DIR="${BASE}/doctest"
OH="${BASE}/../oh"

extract() {
    grep "^#[+-]     " "$1" | sed -e "s/^#[+-]     //g"
}

prefix() {
    COUNT=1
    while read LINE; do
        echo "$1: ${COUNT}: ${LINE}"
        COUNT=$(expr "$COUNT" + 1)
    done
}

find "$DIR" -name "[0-9]*.oh" | grep -Fv unused | sort |
while read FILE; do
    NAME=$(basename "$FILE")
    echo running "$NAME" >&2
    cd $(dirname "$FILE")
    diff <(extract "$NAME" | prefix "$NAME") <("$OH" "$NAME" 2>&1 | prefix "$NAME")
done
