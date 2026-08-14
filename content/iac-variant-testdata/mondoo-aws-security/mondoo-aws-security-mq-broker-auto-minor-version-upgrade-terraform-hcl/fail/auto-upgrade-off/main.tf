# Minor versions carry the broker's security patches; pinning them off means
# known CVEs stay unpatched until someone upgrades by hand.
resource "aws_mq_broker" "orders" {
  broker_name                = "orders"
  engine_type                = "ActiveMQ"
  engine_version             = "5.18.4"
  host_instance_type         = "mq.m5.large"
  auto_minor_version_upgrade = false
  publicly_accessible        = false
  subnet_ids                 = [aws_subnet.private_a.id]

  user {
    username = "admin"
    password = var.mq_password
  }
}
