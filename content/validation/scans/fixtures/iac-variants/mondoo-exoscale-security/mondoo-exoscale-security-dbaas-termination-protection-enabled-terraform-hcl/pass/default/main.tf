resource "exoscale_dbaas" "db" {
  zone = "ch-gva-2"
  name = "app-db"
  type = "pg"
  plan = "hobbyist-2"
}
