resource "snowflake_session_policy" "standard" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "STANDARD_SESSIONS"

  session_idle_timeout_mins    = 30
  session_ui_idle_timeout_mins = 15
}
