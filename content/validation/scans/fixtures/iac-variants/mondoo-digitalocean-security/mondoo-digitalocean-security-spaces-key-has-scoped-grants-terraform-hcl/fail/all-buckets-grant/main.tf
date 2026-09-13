resource "digitalocean_spaces_key" "app" {
  name = "app-key"

  grant {
    bucket     = ""
    permission = "readwrite"
  }
}
