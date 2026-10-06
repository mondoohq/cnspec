resource "exoscale_sks_cluster" "prod" {
  zone = "ch-gva-2"
  name = "prod"
  auto_upgrade = true
}
