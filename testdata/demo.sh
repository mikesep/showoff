#!/usr/bin/env bash

echo "Hello, world!"

sleep 1

: "a comment for set -x mode" # script comment

# two statements in a single line
echo "before semicolon" ; echo "after semicolon"

# multi-line statements
echo "multiline args:" \
  "first" \
  "second" \
  "third"

echo "multiline string: \
  first \
  second \
  third"

cat <<EOF
==============================
     This is a heredoc!
==============================
EOF

for i in $(seq 3)
do
  echo "$i"
done

function bashStyleFunc {
  echo "bashStyleFunc"
}

posixStyleFunc()
{
  echo "posixStyleFunc"
}

read -r -e -p "Who are you? " name
cat << EOF
Hi $name! It's nice to meet you!
EOF

(
  echo "[start of a subcommand]"
  for i in $(seq 3 1) ; do
    echo $i
    sleep 1
  done
  echo "BLASTOFF!"
  echo "[end of subcommand]"
)
