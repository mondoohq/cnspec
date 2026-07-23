# Non-compliant: Azure connection does not use identity federation
# (no federated_application_client_id set).
resource "google_bigquery_connection" "azure_legacy" {
  connection_id = "my-azure-connection"
  location      = "US"

  azure {
    customer_tenant_id = "00000000-0000-0000-0000-000000000000"
  }
}
