# Non-compliant: MQ broker has general logging disabled.
resource "aws_mq_broker" "fail_example" {
  broker_name = "fail-example"
  engine_type = "ActiveMQ"

  logs {
    audit   = true
    general = false
  }
}
