# With session_idle_timeout_mins unset the policy does not constrain idle
# client sessions at all.
resource "snowflake_session_policy" "standard" {
  database = snowflake_database.security.name
  schema   = snowflake_schema.policies.name
  name     = "STANDARD_SESSIONS"

  session_ui_idle_timeout_mins = 15
}
