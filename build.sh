#!/bin/bash

get_script_dir() {
    local SOURCE_PATH="${BASH_SOURCE[0]}"
    local SYMLINK_DIR
    local SCRIPT_DIR
    # Resolve symlinks recursively
    while [ -L "$SOURCE_PATH" ]; do
        # Get symlink directory
        SYMLINK_DIR="$( cd -P "$( dirname "$SOURCE_PATH" )" >/dev/null 2>&1 && pwd )"
        # Resolve symlink target (relative or absolute)
        SOURCE_PATH="$(readlink "$SOURCE_PATH")"
        # Check if candidate path is relative or absolute
        if [[ $SOURCE_PATH != /* ]]; then
            # Candidate path is relative, resolve to full path
            SOURCE_PATH=$SYMLINK_DIR/$SOURCE_PATH
        fi
    done
    # Get final script directory path from fully resolved source path
    SCRIPT_DIR="$(cd -P "$( dirname "$SOURCE_PATH" )" >/dev/null 2>&1 && pwd)"
    echo "$SCRIPT_DIR"
}

SCRIPT_DIR=$(get_script_dir)

mkdir -p $SCRIPT_DIR/bin

cd $SCRIPT_DIR
go build -o bin/mp
cp mpc.py bin/mpc

cat <<EOF
Binary has been written to $SCRIPT_DIR/bin/mp.

To save to a directory already in the PATH, run the following command:
	sudo mv $SCRIPT_DIR/bin/* /usr/local/bin # or any other directory already in the PATH

To add the binary to the PATH as is, run:
	echo 'export PATH=\$PATH:$SCRIPT_DIR/bin' >> ~/.bash_profile # or other configuration script
EOF
