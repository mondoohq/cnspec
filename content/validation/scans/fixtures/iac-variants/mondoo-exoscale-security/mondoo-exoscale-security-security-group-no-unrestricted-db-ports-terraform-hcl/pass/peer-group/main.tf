resource "exoscale_security_group" "web" {
  name = "web"
}
resource "exoscale_security_group" "app" {
  name = "app"
}

resource "exoscale_security_group_rule" "r" {
  security_group_id      = exoscale_security_group.web.id
  type                   = "INGRESS"
  protocol               = "TCP"
  user_security_group_id = exoscale_security_group.app.id
  start_port             = 5432
  end_port               = 5432
}
