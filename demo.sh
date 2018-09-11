#!/bin/bash

echo "HostName:$(hostname)"
echo "I am a test Shell script running on the remote server!"
echo "Script Args \$1: $1"
echo "Script Args \$2: $2"
echo "What happens if an exception occurs during script execution?"
ls ThisFileIsNotExist