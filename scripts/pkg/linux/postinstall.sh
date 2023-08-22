#!/bin/sh
# Copyright (c) Mondoo, Inc.
# SPDX-License-Identifier: BUSL-1.1


# call the cnspec migrate command
echo " -> Migrate cnspec configuration"
cnspec --config /etc/opt/mondoo/mondoo.yml migrate

if [ "$(cat /proc/1/comm)" = "init" ]
then
  # Upstart best-effort to just stop mondoo service and
  # start cnspec service. Removing mondoo from upstart
  # will not be scripted here.
  # if mondoo service is running, stop it and start cnspec
  test -n "$(initctl status mondoo | grep -o running)" && \
    echo " -> Stop mondoo service; start cnspec service (init)" && \
    initctl stop mondoo && \
    initctl start cnspec

  test -n "$(initctl status cnspec | grep -o running)" && \
    echo " -> Restart cnspec service (init)" && \
    initctl restart cnspec

elif [ "$(cat /proc/1/comm)" = "systemd" ]
then
  # if Mondoo service is currently running, make sure
  # cnspec service is running to replace it
  systemctl is-active --quiet mondoo && \
    echo " -> Mondoo service running, replacing with cnspec (systemd)" && \
    systemctl stop mondoo && \
    systemctl daemon-reload && \
    systemctl restart cnspec

  # if Mondoo service is set up to run on boot
  # replace it with cnspec to start on boot
  systemctl is-enabled --quiet mondoo 2>/dev/null && \
    echo " -> Mondoo service enabled, replacing with cnspec (systemd)" && \
    systemctl disable mondoo &&
    systemctl enable cnspec

  systemctl is-active --quiet cnspec && \
    echo " -> Restart cnspec service (systemd)" && \
    systemctl daemon-reload && \
    systemctl restart cnspec
fi

exit 0
