# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Live fixtures for mondoo-cassandra-security.

Two containers. The first is the image as it ships, which is what a check
written for a fresh cluster has to fail against. The second is configured for
security and then deliberately walked backwards, so the role and keyspace checks
are seen from both sides without paying for a third Cassandra — a cold JVM plus
the system_auth bootstrap is a minute and a half before anything can connect.

Read the vacuous passes in the `defaults` fixture as what they are. With
`AllowAllAuthenticator` there are no rows in `system_auth.roles` and no user
keyspaces, so every check shaped `roles.where(...).all(...)` passes over an
empty list. Those passes prove the query runs; they do not prove it discriminates.
The `secured` fixture is what proves that, which is why its expectations are the
ones to read first when a change here starts failing.
"""

from common import Exec, Fixture, Restart, Scan, Suite, WaitFor, credential

# The image writes cassandra.yaml from a template on every start, so the
# settings that have no environment variable are edited in place and the node is
# restarted into them. Editing before the first start is not an option: the file
# does not exist until the entrypoint has run.
_SECURE_YAML = [
    "sed", "-i",
    "-e", "s/^authenticator:.*/authenticator: PasswordAuthenticator/",
    "-e", "s/^authorizer:.*/authorizer: CassandraAuthorizer/",
    "-e", "s/^role_manager:.*/role_manager: CassandraRoleManager/",
    "-e", "s/^network_authorizer:.*/network_authorizer: CassandraNetworkAuthorizer/",
    # Audit logging lives in a nested block, so the substitution is scoped to
    # the lines between `audit_logging_options:` and the next top-level key —
    # `enabled: false` appears thirty times in this file. `nodetool
    # enableauditlog` looks like the tidier route and is not: it changes the
    # running state without touching system_views.settings, which is where the
    # provider reads from, so the check stays failing.
    "-e", "/^audit_logging_options:/,/^[a-z_]*:/ s/^\\(  *\\)enabled: false/\\1enabled: true/",
    "/etc/cassandra/cassandra.yaml",
]

# Readiness is a CQL round trip, not an open port. Cassandra binds 9042 well
# before system_auth holds the default superuser, and a scan in that window
# fails to authenticate for reasons that have nothing to do with the policy.
_READY_NOAUTH = WaitFor(["cqlsh", "-e", "SELECT now() FROM system.local"])
_READY_AUTH = WaitFor(["cqlsh", "-u", "cassandra", "-p", "cassandra", "-e", "SELECT now() FROM system.local"])


def _cql(statement: str, user: str = "cassandra", password: str = "cassandra") -> Exec:
    return Exec(["cqlsh", "-u", user, "-p", password, "-e", statement])


# Bringing the cluster to the state every check should pass in: a named
# superuser that is not the built-in one, a keyspace replicated the way the
# checks ask for, and a service role scoped to that keyspace rather than to the
# cluster.
#
# The passwords arrive as arguments rather than as literals here; see
# common.credential. Only the statements that carry one are f-strings — CQL
# replication maps are full of braces, and an f-string would read them as
# interpolation.
def _harden(admin_password: str, svc_password: str) -> list[Exec]:
    return [
        _cql(
            "CREATE ROLE IF NOT EXISTS dba_admin WITH LOGIN = true "
            f"AND PASSWORD = '{admin_password}' AND SUPERUSER = true"
        ),
        _cql(
            "CREATE KEYSPACE IF NOT EXISTS app WITH replication = "
            "{'class':'NetworkTopologyStrategy','datacenter1':1} AND durable_writes = true"
        ),
        _cql(f"CREATE ROLE IF NOT EXISTS svc WITH LOGIN = true AND PASSWORD = '{svc_password}'"),
        _cql("GRANT SELECT ON KEYSPACE app TO svc"),
        # Last, because it is the statement that takes the account we are using
        # out of service. Everything after this authenticates as dba_admin.
        _cql("ALTER ROLE cassandra WITH LOGIN = false"),
    ]


# Each statement below undoes exactly one check, so a failure names the check
# rather than the fixture.
def _degrade(admin_password: str) -> list[Exec]:
    admin = ("dba_admin", admin_password)
    return [
        _cql("CREATE ROLE IF NOT EXISTS nopw WITH LOGIN = true", *admin),
        _cql("GRANT ALL PERMISSIONS ON ALL KEYSPACES TO svc", *admin),
        _cql(
            "CREATE KEYSPACE IF NOT EXISTS legacy WITH replication = "
            "{'class':'SimpleStrategy','replication_factor':1} AND durable_writes = false",
            *admin,
        ),
        _cql("ALTER ROLE cassandra WITH LOGIN = true", *admin),
    ]


def build_suite(workdir):
    """The Cassandra suite. Fixture account passwords are generated per run; the
    fixtures otherwise reach their configuration through commands in the
    container."""
    admin_password = credential()
    svc_password = credential()
    return Suite(
        provider="cassandra",
        policy="mondoo-cassandra-security.mql.yaml",
        policy_uid="mondoo-cassandra-security",
        no_pass_fixture={
            "mondoo-cassandra-security-client-encryption-enabled":
                "needs a JKS keystore and truststore built into the image; a single-container "
                "fixture can show the setting off but not on",
            "mondoo-cassandra-security-internode-encryption-enabled":
                "same keystores, and internode traffic needs a second node before the setting "
                "does anything observable",
            "mondoo-cassandra-security-system-auth-replication-factor":
                "a replication factor of three needs three nodes. Setting it higher than the node "
                "count makes the LOCAL_QUORUM read for authentication unsatisfiable, which locks "
                "every account out of the cluster — verified the hard way while writing this suite",
        },
        fixtures=[
            Fixture(
                name="cassandra-defaults",
                image="cassandra:5.0",
                container_port=9042,
                host_port=19042,
                memory="3g",
                env={"MAX_HEAP_SIZE": "1500M", "HEAP_NEWSIZE": "300M"},
                setup=[_READY_NOAUTH],
                scans=[
                    Scan(
                        name="defaults",
                        expect={
                            "mondoo-cassandra-security-authentication-enabled": "fail",
                            "mondoo-cassandra-security-authorization-enabled": "fail",
                            "mondoo-cassandra-security-audit-logging-enabled": "fail",
                            "mondoo-cassandra-security-client-encryption-enabled": "fail",
                            "mondoo-cassandra-security-internode-encryption-enabled": "fail",
                            "mondoo-cassandra-security-network-authorizer-enabled": "fail",
                            "mondoo-cassandra-security-cluster-name-not-default": "fail",
                            "mondoo-cassandra-security-system-auth-replication-factor": "fail",
                            # Vacuous: no roles and no user keyspaces exist yet.
                            "mondoo-cassandra-security-default-superuser-login-disabled": "pass",
                            "mondoo-cassandra-security-login-roles-require-passwords": "pass",
                            "mondoo-cassandra-security-roles-least-privilege": "pass",
                            "mondoo-cassandra-security-keyspaces-durable-writes": "pass",
                            "mondoo-cassandra-security-keyspaces-network-topology-strategy": "pass",
                        },
                    )
                ],
            ),
            Fixture(
                name="cassandra-secured",
                image="cassandra:5.0",
                container_port=9042,
                host_port=19043,
                memory="3g",
                env={"MAX_HEAP_SIZE": "1500M", "HEAP_NEWSIZE": "300M", "CASSANDRA_CLUSTER_NAME": "prod-eu-1"},
                setup=[
                    _READY_NOAUTH,
                    Exec(_SECURE_YAML),
                    Restart(),
                    _READY_AUTH,
                ],
                scans=[
                    Scan(
                        name="hardened",
                        user="dba_admin",
                        password=admin_password,
                        seed=_harden(admin_password, svc_password),
                        expect={
                            "mondoo-cassandra-security-authentication-enabled": "pass",
                            "mondoo-cassandra-security-authorization-enabled": "pass",
                            "mondoo-cassandra-security-audit-logging-enabled": "pass",
                            "mondoo-cassandra-security-network-authorizer-enabled": "pass",
                            "mondoo-cassandra-security-cluster-name-not-default": "pass",
                            "mondoo-cassandra-security-default-superuser-login-disabled": "pass",
                            "mondoo-cassandra-security-login-roles-require-passwords": "pass",
                            "mondoo-cassandra-security-roles-least-privilege": "pass",
                            "mondoo-cassandra-security-keyspaces-durable-writes": "pass",
                            "mondoo-cassandra-security-keyspaces-network-topology-strategy": "pass",
                            "mondoo-cassandra-security-client-encryption-enabled": "fail",
                            "mondoo-cassandra-security-internode-encryption-enabled": "fail",
                            "mondoo-cassandra-security-system-auth-replication-factor": "fail",
                        },
                    ),
                    Scan(
                        name="degraded",
                        user="dba_admin",
                        password=admin_password,
                        seed=_degrade(admin_password),
                        expect={
                            "mondoo-cassandra-security-authentication-enabled": "pass",
                            "mondoo-cassandra-security-authorization-enabled": "pass",
                            "mondoo-cassandra-security-network-authorizer-enabled": "pass",
                            "mondoo-cassandra-security-cluster-name-not-default": "pass",
                            # Set in cassandra.yaml, so it survives the degrade —
                            # the fail side comes from the defaults fixture.
                            "mondoo-cassandra-security-audit-logging-enabled": "pass",
                            "mondoo-cassandra-security-default-superuser-login-disabled": "fail",
                            "mondoo-cassandra-security-login-roles-require-passwords": "fail",
                            "mondoo-cassandra-security-roles-least-privilege": "fail",
                            "mondoo-cassandra-security-keyspaces-durable-writes": "fail",
                            "mondoo-cassandra-security-keyspaces-network-topology-strategy": "fail",
                            "mondoo-cassandra-security-client-encryption-enabled": "fail",
                            "mondoo-cassandra-security-internode-encryption-enabled": "fail",
                            "mondoo-cassandra-security-system-auth-replication-factor": "fail",
                        },
                    ),
                ],
            ),
        ],
    )

