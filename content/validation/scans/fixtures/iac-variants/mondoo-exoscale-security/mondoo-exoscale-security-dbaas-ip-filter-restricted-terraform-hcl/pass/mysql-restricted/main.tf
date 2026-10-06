resource "exoscale_dbaas" "db" {
  zone = "ch-gva-2"
  name = "app-db"
  type = "mysql"
  plan = "hobbyist-2"

  mysql = {
    ip_filter = ["10.0.0.0/8"]
  }
}
