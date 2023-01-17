#!/bin/sh

# Run during uninstall
# Note: rpm runs this phase even during upgrades therefore we need to include a condition
if [ "$1" -eq 0 ] ; then
    if [ "$(cat /proc/1/comm)" = "init" ]
    then
        echo " -> Stop cnspec service (init)"
        stop cnspec || true
    elif [ "$(cat /proc/1/comm)" = "systemd" ]
    then
        echo " -> Stop cnspec service"
        systemctl stop cnspec
        systemctl disable cnspec
        systemctl daemon-reload
    fi
fi
