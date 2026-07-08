resource "google_workbench_instance" "pass" {
  name     = "workbench-instance"
  location = "us-central1-a"

  gce_setup {
    machine_type     = "e2-standard-4"
    disable_public_ip = true
  }
}
