resource "databricks_token" "ci" {
  comment          = "ci pipeline"
  lifetime_seconds = 31536000
}
