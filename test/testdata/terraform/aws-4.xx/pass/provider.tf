terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "~> 4.30"
    }
  }
}

provider "aws" {
  # Configuration options
}