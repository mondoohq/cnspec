# A four-hour idle window leaves an unattended session usable long after the
# user has walked away.
resource "snowflake_session_policy" "standard" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "STANDARD_SESSIONS"

  session_idle_timeout_mins    = 240
  session_ui_idle_timeout_mins = 15
}
