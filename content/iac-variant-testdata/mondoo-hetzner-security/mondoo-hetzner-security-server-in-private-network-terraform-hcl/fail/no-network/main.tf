resource "hcloud_server" "example" {
  name        = "example"
  image       = "ubuntu-24.04"
  server_type = "cx22"
  location    = "nbg1"
}
