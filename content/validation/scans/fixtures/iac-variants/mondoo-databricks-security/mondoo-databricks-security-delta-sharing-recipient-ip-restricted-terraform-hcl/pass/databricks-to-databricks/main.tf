resource "databricks_recipient" "internal" {
  name                                 = "internal"
  authentication_type                  = "DATABRICKS"
  data_recipient_global_metastore_id   = "aws:us-east-1:abc"
}
