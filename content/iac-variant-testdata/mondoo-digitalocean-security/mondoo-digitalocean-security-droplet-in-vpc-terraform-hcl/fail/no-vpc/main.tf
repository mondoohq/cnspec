resource "digitalocean_droplet" "web" {
  name   = "web-1"
  region = "nyc1"
  size   = "s-1vcpu-1gb"
  image  = "ubuntu-22-04-x64"
}
