resource "google_vertex_ai_tensorboard" "fail" {
  display_name = "terraform-tensorboard"
  region       = "us-central1"
}
