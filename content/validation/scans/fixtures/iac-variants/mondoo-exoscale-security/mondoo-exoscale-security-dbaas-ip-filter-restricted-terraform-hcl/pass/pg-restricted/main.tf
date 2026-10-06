resource "exoscale_dbaas" "db" {
  zone = "ch-gva-2"
  name = "app-db"
  type = "pg"
  plan = "hobbyist-2"

  pg = {
    version   = "16"
    ip_filter = ["203.0.113.0/24"]
  }
}
