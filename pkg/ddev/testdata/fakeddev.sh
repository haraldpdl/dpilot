#!/bin/sh
# Test double for the ddev binary: records argv and answers -j queries.
printf "%s\n" "$*" >> "$DPILOT_ARGS_FILE"
case "$1" in
  describe) printf "{\"level\":\"info\",\"msg\":\"\",\"raw\":{\"name\":\"x\",\"status\":\"running\"}}\n" ;;
  list) printf "{\"level\":\"info\",\"msg\":\"\",\"raw\":[]}\n" ;;
esac
