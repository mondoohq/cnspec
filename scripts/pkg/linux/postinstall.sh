#!/bin/sh

# call the cnspec migrate command
echo " -> Migrate cnspec configuration"
cnspec --config /etc/opt/mondoo/mondoo.yml migrate

if [ "$(cat /proc/1/comm)" = "init" ]
then
  test -n "$(initctl status cnspec | grep -o running)" && \
    echo " -> Restart cnspec service (init)" && \
    initctl restart cnspec

  # if mondoo service is running, stop it and start cnspec
  test -n "$(initctl status mondoo | grep -o running)" && \
    echo " -> Replace mondoo service with cnspec service (init)" && \
    initctl stop mondoo && \
    initctl start cnspec

elif [ "$(cat /proc/1/comm)" = "systemd" ]
then
  systemctl is-active --quiet cnspec && \
    echo " -> Restart cnspec service (systemd)" && \
    systemctl daemon-reload && \
    systemctl restart cnspec

  systemctl is-active --quiet mondoo && \
      echo " -> Replace mondoo service with cnspec service (systemd)" && \
      systemctl daemon-reload && \
      systemctl stop mondoo && \
      systemctl disable mondoo && \
      systemctl start cnspec
fi

exit 0
