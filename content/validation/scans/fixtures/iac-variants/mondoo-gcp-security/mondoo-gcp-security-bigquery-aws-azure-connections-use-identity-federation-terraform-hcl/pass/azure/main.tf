# Compliant: Azure connection uses identity federation via a federated app client id.
resource "google_bigquery_connection" "azure" {
  connection_id = "my-azure-connection"
  location      = "US"

  azure {
    customer_tenant_id              = "00000000-0000-0000-0000-000000000000"
    federated_application_client_id = "11111111-1111-1111-1111-111111111111"
  }
}
