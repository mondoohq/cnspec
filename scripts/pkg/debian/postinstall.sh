#!/bin/sh

# call the cnspec migrate command
echo " -> Migrate cnspec configuration"
cnspec --config /etc/opt/mondoo/mondoo.yml migrate

if [ "$(cat /proc/1/comm)" = "init" ]
then
  test -n "$(initctl status cnspec | grep -o running)" && \
    echo " -> Restart cnspec service (init)" && \
    initctl restart cnspec
elif [ "$(cat /proc/1/comm)" = "systemd" ]
then
  systemctl is-active --quiet cnspec && \
    echo " -> Restart cnspec service (systemd)" && \
    systemctl daemon-reload && \
    systemctl restart cnspec
fi

exit 0
