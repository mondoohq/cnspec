resource "stackit_sqlserverflex_instance" "app" {
  project_id = "8e1e0b09-2f5a-4c0d-9f04-1b6b2a3c4d5e"
  name       = "app"
  acl        = ["10.0.0.0/8"]

  flavor = {
    cpu = 4
    ram = 16
  }
  storage = {
    class = "premium-perf2-stackit"
    size  = 5
  }
  version = 2022
}
