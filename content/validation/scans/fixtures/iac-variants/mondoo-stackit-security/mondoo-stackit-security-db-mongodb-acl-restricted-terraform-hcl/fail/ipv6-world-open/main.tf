resource "stackit_mongodbflex_instance" "app" {
  project_id      = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name            = "app"
  acl             = ["10.0.0.0/8", "::/0"]
  backup_schedule = "0 2 * * *"
  replicas        = 3
  flavor = {
    cpu = 1
    ram = 4
  }
  storage = {
    class = "premium-perf2-mongodb"
    size  = 10
  }
  version = "7.0"
  options = {
    type                       = "Replica"
    point_in_time_window_hours = 30
  }
}
