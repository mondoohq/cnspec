# Non-compliant: route destination is a public IP address.
resource "google_datastream_route" "public" {
  display_name          = "route-to-public"
  location              = "us-central1"
  private_connection    = "projects/my-project/locations/us-central1/privateConnections/datastream-pc"
  route_id              = "route-to-public"
  destination_address   = "203.0.113.25"
  destination_port      = 3306
}
