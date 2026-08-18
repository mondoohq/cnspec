# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Live fixtures for mondoo-clickhousedb-security.

Three containers, one of which exists to hold a provider defect still.

`defaults` is the official image with CLICKHOUSE_PASSWORD set — the way almost
every self-hosted ClickHouse is deployed, and the configuration behind the
publicly exposed instances. The entrypoint writes that password into
`users.d/default-user.xml` in plaintext with `<ip>::/0</ip>`, and rewrites the
file on every start, so the account it produces is reachable from anywhere with
a credential stored in the clear. All five checks fail against it.

`lts` runs the 24.8 long-term-support line, which until recently could not be
assessed at all: `system.users.auth_type` is a scalar `Enum8` before 25.x and an
`Array(Enum8)` after, the provider decoded only the array, and every query
touching `clickhousedb.instance.users` errored. This fixture recorded that as
four `error` expectations, and when mondoohq/mql#10067 fixed it the fixture
failed and sent someone back here — which is what an `error` expectation is for.
It now expects the same verdicts as the 25.8 fixture, and holds the LTS line to
the same standard as any other.

`hardened` is built rather than seeded. Getting all five checks to pass at once
needs an account with access management to create the quota, and that account
would then be the one thing failing the least-privilege check — so it is written
in, used, and written back out before the scan.
"""

import hashlib

from common import Exec, Fixture, Restart, Scan, Suite, WaitFor, Write, credential

# Accounts are declared with sha256 hashes rather than <password> elements,
# because a plaintext element is exactly what the strong-authentication check
# exists to fail, and a fixture that trips its own check teaches nothing.
_ADMIN_BLOCK = """\
    <admin>
      <password_sha256_hex>{admin}</password_sha256_hex>
      <networks><ip>127.0.0.1</ip></networks>
      <profile>default</profile>
      <quota>default</quota>
      <access_management>1</access_management>
    </admin>
"""

_SCANNER_BLOCK = """\
    <scanner>
      <password_sha256_hex>{scanner}</password_sha256_hex>
      <networks>
        <ip>172.16.0.0/12</ip>
        <ip>192.168.0.0/16</ip>
        <ip>10.0.0.0/8</ip>
      </networks>
      <profile>default</profile>
      <quota>default</quota>
      <grants><query>GRANT SELECT ON system.*</query></grants>
    </scanner>
"""


def _users_xml(admin: bool, admin_sha: str, scanner_sha: str) -> str:
    """The hardened users.d file, with or without the bootstrap admin.

    `<default remove="remove"/>` deletes the built-in account outright. Leaving
    it in place and restricting it would work too, but removing it is the state
    the host-restriction check's remediation actually recommends.
    """
    body = _SCANNER_BLOCK.format(scanner=scanner_sha)
    if admin:
        body = _ADMIN_BLOCK.format(admin=admin_sha) + body
    return f'<clickhouse>\n  <users>\n    <default remove="remove"></default>\n{body}  </users>\n</clickhouse>\n'


# Files in users.d are merged in filename order, and the image writes its own
# default-user.xml there. The `zz-` prefix is load-bearing: a name sorting
# before that file lets the image's fragment re-add a `default` node carrying
# only <networks>, and ClickHouse then refuses to start because the account has
# no authentication method. Verified — it is a startup crash, not a warning.
_USERS_D = "/etc/clickhouse-server/users.d/zz-cnspec-live.xml"


# A bootstrap account for the permissive fixture. The account the image creates
# has no access management, so there is no way to add the passwordless user that
# fixture needs without one. It is itself unrestricted, which is fine there:
# every check in that fixture is expected to fail.
_BOOTSTRAP = """\
<clickhouse>
  <users>
    <bootstrap>
      <password_sha256_hex>{admin}</password_sha256_hex>
      <networks><ip>::/0</ip></networks>
      <profile>default</profile>
      <quota>default</quota>
      <access_management>1</access_management>
    </bootstrap>
  </users>
</clickhouse>
"""

_READY_DEFAULT = WaitFor(
    ["clickhouse-client", "--password", "testpass", "--query", "SELECT 1"], timeout=120
)
def _ready_as(user: str, password: str) -> WaitFor:
    return WaitFor(
        ["clickhouse-client", "--user", user, "--password", password, "--query", "SELECT 1"],
        timeout=120,
    )


def _client(statement: str, user: str, password: str) -> Exec:
    return Exec(["clickhouse-client", "--user", user, "--password", password, "--query", statement])


def build_suite(workdir):
    """The ClickHouse suite. Passwords are written here in the clear and hashed
    on the way into the XML, so an account's credential stays readable next to
    the account it opens."""
    # Generated per run; see common.credential. The hashes are derived here so
    # each account's credential stays readable next to the account it opens.
    admin_password = credential()
    scanner_password = credential()
    admin_sha = hashlib.sha256(admin_password.encode()).hexdigest()
    scanner_sha = hashlib.sha256(scanner_password.encode()).hexdigest()
    return Suite(
        provider="clickhousedb",
        policy="mondoo-clickhousedb-security.mql.yaml",
        policy_uid="mondoo-clickhousedb-security",
        fixtures=[
            Fixture(
                name="clickhouse-defaults",
                image="clickhouse/clickhouse-server:25.8",
                container_port=9000,
                host_port=19000,
                env={"CLICKHOUSE_PASSWORD": "testpass"},
                setup=[
                    _READY_DEFAULT,
                    Write(_USERS_D, _BOOTSTRAP.format(admin=admin_sha)),
                    Restart(),
                    WaitFor(
                        ["clickhouse-client", "--user", "bootstrap", "--password", admin_password,
                         "--query", "SELECT 1"],
                        timeout=120,
                    ),
                    # One account with no credential at all, which is what the
                    # has-a-password check needs to be seen failing. The image's
                    # own default account does have a password — a plaintext one,
                    # which is the weaker problem the next check catches.
                    _client("CREATE USER IF NOT EXISTS nopw IDENTIFIED WITH no_password",
                            user="bootstrap", password=admin_password),
                ],
                scans=[
                    Scan(
                        name="defaults",
                        user="default",
                        password="testpass",
                        expect={
                            "mondoo-clickhousedb-security-users-have-passwords": "fail",
                            "mondoo-clickhousedb-security-users-strong-authentication": "fail",
                            "mondoo-clickhousedb-security-users-host-restricted": "fail",
                            "mondoo-clickhousedb-security-users-least-privilege": "fail",
                            "mondoo-clickhousedb-security-quota-applies-to-all-users": "fail",
                        },
                    )
                ],
            ),
            Fixture(
                name="clickhouse-lts",
                image="clickhouse/clickhouse-server:24.8",
                container_port=9000,
                host_port=19001,
                env={"CLICKHOUSE_PASSWORD": "testpass"},
                setup=[_READY_DEFAULT],
                scans=[
                    Scan(
                        name="lts-24.8",
                        user="default",
                        password="testpass",
                        expect={
                            # These four returned "error" until mondoohq/mql#10067
                            # taught the provider to decode a scalar auth_type.
                            # They now match the 25.8 fixture, which is the point:
                            # the LTS line assesses the same as any other.
                            #
                            # `users-have-passwords` passing here is not the
                            # server being safe. The account has a password; it
                            # is stored in plaintext and reachable from ::/0,
                            # which is what the next two checks catch.
                            "mondoo-clickhousedb-security-users-have-passwords": "pass",
                            "mondoo-clickhousedb-security-users-strong-authentication": "fail",
                            "mondoo-clickhousedb-security-users-host-restricted": "fail",
                            "mondoo-clickhousedb-security-users-least-privilege": "fail",
                            "mondoo-clickhousedb-security-quota-applies-to-all-users": "fail",
                        },
                    )
                ],
            ),
            Fixture(
                name="clickhouse-hardened",
                image="clickhouse/clickhouse-server:25.8",
                container_port=9000,
                host_port=19002,
                # No CLICKHOUSE_PASSWORD: setting it makes the entrypoint write a
                # plaintext default account back in on every start, which would
                # undo this fixture at the restart below.
                setup=[
                    WaitFor(["clickhouse-client", "--query", "SELECT 1"], timeout=120),
                    Write(_USERS_D, _users_xml(True, admin_sha, scanner_sha)),
                    Restart(),
                    _ready_as("admin", admin_password),
                    _client(
                        "CREATE QUOTA IF NOT EXISTS org_default FOR INTERVAL 1 hour "
                        "MAX queries = 5000, read_rows = 5000000000 TO ALL",
                        "admin", admin_password,
                    ),
                    # The quota lives in the SQL-managed access store, so it
                    # survives the admin account being written back out.
                    Write(_USERS_D, _users_xml(False, admin_sha, scanner_sha)),
                    Restart(),
                ],
                scans=[
                    Scan(
                        name="hardened",
                        user="scanner",
                        password=scanner_password,
                        expect={
                            "mondoo-clickhousedb-security-users-have-passwords": "pass",
                            "mondoo-clickhousedb-security-users-strong-authentication": "pass",
                            "mondoo-clickhousedb-security-users-host-restricted": "pass",
                            "mondoo-clickhousedb-security-users-least-privilege": "pass",
                            "mondoo-clickhousedb-security-quota-applies-to-all-users": "pass",
                        },
                    )
                ],
            ),
        ],
    )
