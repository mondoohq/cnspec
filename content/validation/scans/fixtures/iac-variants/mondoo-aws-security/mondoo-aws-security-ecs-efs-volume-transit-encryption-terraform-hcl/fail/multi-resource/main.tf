# Non-compliant: two task definitions; exactly one has a volume with transit
# encryption DISABLED. .all() over resources must still fail.
resource "aws_ecs_task_definition" "good" {
  family                = "good"
  container_definitions = jsonencode([{ name = "app", image = "nginx" }])

  volume {
    name = "efs-vol"

    efs_volume_configuration {
      file_system_id     = "fs-11111111"
      transit_encryption = "ENABLED"
    }
  }
}

resource "aws_ecs_task_definition" "bad" {
  family                = "bad"
  container_definitions = jsonencode([{ name = "app", image = "nginx" }])

  volume {
    name = "efs-vol"

    efs_volume_configuration {
      file_system_id     = "fs-22222222"
      transit_encryption = "DISABLED"
    }
  }
}
