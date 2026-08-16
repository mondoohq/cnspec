# Compliant: route destination is inside the 172.16.0.0/12 private range.
resource "google_datastream_route" "compliant" {
  display_name          = "route-to-source"
  location              = "us-central1"
  private_connection    = "projects/my-project/locations/us-central1/privateConnections/datastream-pc"
  route_id              = "route-to-source"
  destination_address   = "172.16.8.5"
  destination_port      = 5432
}
