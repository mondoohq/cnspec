# With this off, COPY INTO <location> can write to an arbitrary inline URL,
# bypassing the storage integrations that constrain where data may land.
resource "snowflake_account_parameter" "prevent_unload_to_inline_url" {
  key   = "PREVENT_UNLOAD_TO_INLINE_URL"
  value = "false"
}
