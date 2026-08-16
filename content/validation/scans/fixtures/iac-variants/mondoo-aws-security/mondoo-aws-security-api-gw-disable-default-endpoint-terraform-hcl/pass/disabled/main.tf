resource "aws_api_gateway_rest_api" "orders" {
  name                         = "orders-api"
  description                  = "Order service, fronted by a custom domain"
  disable_execute_api_endpoint = true

  endpoint_configuration {
    types = ["REGIONAL"]
  }
}
