# Non-compliant: redis (6379) open to the world via aws_vpc_security_group_ingress_rule.
resource "aws_security_group" "redis" {
  name = "redis"
}

resource "aws_vpc_security_group_ingress_rule" "redis" {
  security_group_id = aws_security_group.redis.id
  from_port         = 6379
  to_port           = 6379
  ip_protocol       = "tcp"
  cidr_ipv4         = "0.0.0.0/0"
}
