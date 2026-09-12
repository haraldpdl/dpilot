#!/bin/sh
# Test double for the ddev binary: records argv and answers -j queries.
# With DPILOT_HANG set it blocks like a long ddev start and records how it
# was interrupted: "INT" when it received SIGINT and could clean up.
printf "%s\n" "$*" >> "$DPILOT_ARGS_FILE"
if [ -n "${DPILOT_HANG:-}" ]; then
  sleep 30 &
  child=$!
  if [ "$DPILOT_HANG" = "ignore" ]; then
    trap '' INT # a ddev that does not react to SIGINT; only a kill ends it
  else
    trap 'kill $child 2>/dev/null; printf "INT\n" >> "$DPILOT_ARGS_FILE"; exit 130' INT
  fi
  wait $child
  exit 0
fi
case "$1" in
  describe) printf "{\"level\":\"info\",\"msg\":\"\",\"raw\":{\"name\":\"x\",\"status\":\"running\"}}\n" ;;
  list) printf "{\"level\":\"info\",\"msg\":\"\",\"raw\":[]}\n" ;;
esac
