resource "stackit_postgresflex_instance" "app" {
  project_id      = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name            = "app"
  backup_schedule = "00 02 * * *"
  replicas        = 3
  flavor = {
    cpu = 2
    ram = 4
  }
  network = {
    access_scope = "PUBLIC"
    acl          = ["10.0.0.0/8"]
  }
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  version = 14
}
