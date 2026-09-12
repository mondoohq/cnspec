variable "k" {}
provider "aws" {
  region = "us-east-1"
  access_key = var.k
  secret_key = var.k
}
