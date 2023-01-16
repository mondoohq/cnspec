#!/bin/sh

if [ "$1" = "remove" ]; then
    if [ "$(cat /proc/1/comm)" = "init" ]
    then
      echo " -> Stop cnspec service (init)"
      if [ -f "$FILE" ]; then
        stop cnspec || true
      fi
    elif [ "$(cat /proc/1/comm)" = "systemd" ]
    then
        echo " -> Stop cnspec service"
        systemctl stop cnspec
        systemctl disable cnspec
        systemctl daemon-reload
    fi
fi
