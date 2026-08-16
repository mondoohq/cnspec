# Compliant: route destination is inside the 10.0.0.0/8 private range.
resource "google_datastream_route" "compliant" {
  display_name          = "route-to-source"
  location              = "us-central1"
  private_connection    = "projects/my-project/locations/us-central1/privateConnections/datastream-pc"
  route_id              = "route-to-source"
  destination_address   = "10.20.30.40"
  destination_port      = 3306
}
