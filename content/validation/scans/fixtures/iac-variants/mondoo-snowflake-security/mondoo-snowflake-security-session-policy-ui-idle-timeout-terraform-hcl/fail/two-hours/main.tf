# Snowsight sessions are browser sessions; two hours of idle tolerance is long
# enough for an unlocked laptop to be used by someone else.
resource "snowflake_session_policy" "standard" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "STANDARD_SESSIONS"

  session_idle_timeout_mins    = 30
  session_ui_idle_timeout_mins = 120
}
