#! /bin/bash

TARGET=/usr/local/bin/dl
MESSAGE_START="Removing dl"
MESSAGE_END="dl removed"

echo "$MESSAGE_START"
rm $TARGET
echo "$MESSAGE_END"
