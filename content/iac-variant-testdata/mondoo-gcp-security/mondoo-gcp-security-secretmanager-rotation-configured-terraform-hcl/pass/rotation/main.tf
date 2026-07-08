# Compliant: secret has a rotation block with a rotation_period set.
resource "google_secret_manager_secret" "pass_example" {
  secret_id = "my-secret"

  replication {
    auto {}
  }

  topics {
    name = "projects/my-project/topics/secret-rotations"
  }

  rotation {
    rotation_period    = "2592000s"
    next_rotation_time = "2026-08-01T00:00:00Z"
  }
}
