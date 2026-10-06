resource "exoscale_dbaas" "db" {
  zone = "ch-gva-2"
  name = "app-db"
  type = "valkey"
  plan = "hobbyist-2"

  valkey = {
    ip_filter = ["10.0.0.0/8", "::/0"]
  }
}
