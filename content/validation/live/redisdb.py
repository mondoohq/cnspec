# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Live fixtures for mondoo-redis-security.

Three configurations cover every check from both sides:

    defaults    the image as it ships — no password, bound to every interface
    hardened    ACL users, one bound address, memory capped, persistence on
    tls-only    a TLS listener with client certificates switched off

The split is not one fixture per check. `tls-only` also turns persistence off
and uncaps memory, because those are the two checks `hardened` can only be seen
passing, and a second container costs a container.

The bound-address check is why `hardened` sits on its own Docker network. A
container on the default bridge is assigned an address when it starts, so a
configuration file baked beforehand cannot name one, and `bind` would have to
stay a wildcard — leaving the check with no pass fixture at all.
"""

from common import Fixture, Scan, Suite, WaitFor, credential, mint_certs

_NETWORK = "cnspec-live-net"
_SUBNET = "172.28.0.0/16"
_BOUND_IP = "172.28.0.10"

# Loopback is bound alongside the fixed address, exactly as the check's own
# remediation recommends, so the readiness probe can reach the server from
# inside the container. `bindsAllInterfaces` is true only for a wildcard, so
# naming two addresses still leaves the check with something to pass on.
def _hardened_conf(password: str) -> str:
    return f"""\
bind {_BOUND_IP} 127.0.0.1 -::1
protected-mode yes
maxmemory 268435456
maxmemory-policy allkeys-lru
appendonly yes
user default off
user auditor on >{password} ~* &* +@all
"""

def _tls_conf(password: str) -> str:
    return f"""\
port 0
tls-port 6379
tls-cert-file /tls/server.crt
tls-key-file /tls/server.key
tls-ca-cert-file /tls/ca.crt
tls-auth-clients no
protected-mode yes
maxmemory 0
save ""
appendonly no
requirepass {password}
"""

_READY = WaitFor(["redis-cli", "ping"], timeout=60)


# The fixed address the hardened fixture binds to, and the network that hands it
# out. Declared here rather than in the entry point so this module stays the one
# place that knows what the Redis fixtures need.
NETWORKS = {_NETWORK: _SUBNET}


def build_suite(workdir):
    """The Redis suite. The TLS fixture needs a certificate chain, minted into
    `workdir` on the way in."""
    certs = mint_certs(workdir)
    # Generated per run; see common.credential.
    auditor_password = credential()
    tls_password = credential()
    return Suite(
        provider="redisdb",
        policy="mondoo-redis-security.mql.yaml",
        policy_uid="mondoo-redis-security",
        fixtures=[
            Fixture(
                name="redis-defaults",
                image="redis:7",
                container_port=6379,
                host_port=63790,
                setup=[_READY],
                scans=[
                    Scan(
                        name="defaults",
                        expect={
                            "mondoo-redis-security-authentication-required": "fail",
                            "mondoo-redis-security-default-user-disabled": "fail",
                            "mondoo-redis-security-memory-limit-set": "fail",
                            "mondoo-redis-security-not-bound-to-all-interfaces": "fail",
                            "mondoo-redis-security-persistence-enabled": "pass",
                            "mondoo-redis-security-plaintext-port-disabled": "fail",
                            "mondoo-redis-security-protected-mode-enabled": "fail",
                            "mondoo-redis-security-tls-client-certificate-auth": "pass",
                            "mondoo-redis-security-tls-enabled": "fail",
                        },
                    )
                ],
            ),
            Fixture(
                name="redis-hardened",
                image="redis:7",
                container_port=6379,
                host_port=63791,
                network=_NETWORK,
                ip=_BOUND_IP,
                mounts={"/etc/redis.conf": _hardened_conf(auditor_password)},
                command=["redis-server", "/etc/redis.conf"],
                setup=[
                    WaitFor(
                        ["redis-cli", "-u", f"redis://auditor:{auditor_password}@127.0.0.1:6379", "ping"],
                        timeout=90,
                    )
                ],
                scans=[
                    Scan(
                        name="hardened",
                        user="auditor",
                        password=auditor_password,
                        expect={
                            "mondoo-redis-security-authentication-required": "pass",
                            "mondoo-redis-security-default-user-disabled": "pass",
                            "mondoo-redis-security-memory-limit-set": "pass",
                            "mondoo-redis-security-not-bound-to-all-interfaces": "pass",
                            "mondoo-redis-security-persistence-enabled": "pass",
                            "mondoo-redis-security-plaintext-port-disabled": "fail",
                            "mondoo-redis-security-protected-mode-enabled": "pass",
                            "mondoo-redis-security-tls-client-certificate-auth": "pass",
                            "mondoo-redis-security-tls-enabled": "fail",
                        },
                    )
                ],
            ),
            Fixture(
                name="redis-tls",
                image="redis:7",
                container_port=6379,
                host_port=63792,
                mounts={
                    "/etc/redis.conf": _tls_conf(tls_password),
                    "/tls/ca.crt": certs["ca.crt"],
                    "/tls/server.crt": certs["server.crt"],
                    "/tls/server.key": certs["server.key"],
                },
                command=["redis-server", "/etc/redis.conf"],
                setup=[
                    WaitFor(
                        [
                            "redis-cli", "--tls",
                            "--cacert", "/tls/ca.crt",
                            "-a", tls_password, "--no-auth-warning",
                            "ping",
                        ],
                        timeout=60,
                    )
                ],
                scans=[
                    Scan(
                        name="tls-only",
                        password=tls_password,
                        tls=True,
                        expect={
                            "mondoo-redis-security-authentication-required": "pass",
                            "mondoo-redis-security-default-user-disabled": "fail",
                            "mondoo-redis-security-memory-limit-set": "fail",
                            "mondoo-redis-security-not-bound-to-all-interfaces": "fail",
                            "mondoo-redis-security-persistence-enabled": "fail",
                            # The fixture this check exists for: the plaintext
                            # listener is closed and only the TLS one is bound.
                            "mondoo-redis-security-plaintext-port-disabled": "pass",
                            "mondoo-redis-security-protected-mode-enabled": "pass",
                            "mondoo-redis-security-tls-client-certificate-auth": "fail",
                            "mondoo-redis-security-tls-enabled": "pass",
                        },
                    )
                ],
            ),
        ],
    )

# `plaintext-port-disabled` is the check this fixture set was worth building for.
# It reads `redisdb.instance.port == 0`, and it failed against `tls-only` — the
# exact configuration its own remediation produces — because the provider
# populated `port` from `INFO`, which reports whichever listener is active, so a
# server with `port 0` and a TLS listener read as 6379. `CONFIG GET port`
# returns 0 there. Fixed in mondoohq/mql#10066; this fixture is the regression
# test, and it needs a provider built from that fix or newer.
