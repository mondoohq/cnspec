#!/bin/sh
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1


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
        # remove the opt-in hardening settings, but no other file in the directory
        rm -f /etc/systemd/system/cnspec.service.d/50-cnspec-hardening.conf
        rmdir /etc/systemd/system/cnspec.service.d 2>/dev/null || true
        systemctl daemon-reload
    fi
fi
