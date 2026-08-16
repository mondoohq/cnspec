# Non-compliant: EFS volume with transit encryption DISABLED, expressed via a
# dynamic block. Real task definitions frequently generate volumes dynamically
# from a variable.
variable "efs_volumes" {
  type = map(string)
  default = {
    "efs-vol" = "fs-12345678"
  }
}

resource "aws_ecs_task_definition" "fail" {
  family                = "service"
  container_definitions = jsonencode([{ name = "app", image = "nginx" }])

  dynamic "volume" {
    for_each = var.efs_volumes
    content {
      name = volume.key

      efs_volume_configuration {
        file_system_id     = volume.value
        transit_encryption = "DISABLED"
      }
    }
  }
}
