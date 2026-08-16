resource "aws_ecs_task_definition" "pass" {
  family = "service"
  container_definitions = jsonencode([
    {
      name            = "app"
      image           = "nginx"
      linuxParameters = { initProcessEnabled = true }
    }
  ])
}
