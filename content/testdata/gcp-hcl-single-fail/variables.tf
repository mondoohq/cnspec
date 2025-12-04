variable "gcp_project_id" {
  description = "GCP Project ID"
  type        = string
  #default     = "" // add your project name here
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

variable "database_version" {
  description = "MySQL database version"
  type        = string
  default     = "MYSQL_8_0"
}

variable "tier" {
  description = "Instance tier"
  type        = string
  default     = "db-n1-standard-1"
}

variable "authorized_network_name" {
  description = "Name of the authorized network"
  type        = string
  default     = "office-network"
}

variable "authorized_network_cidr" {
  description = "CIDR block for authorized network"
  type        = string
  default     = "151.251.171.0/24"
}

variable "database_name" {
  description = "Name of the database"
  type        = string
  default     = "example-database"
}

variable "user_name" {
  description = "Database user name"
  type        = string
  default     = "example-user"
}

variable "user_password" {
  description = "Database user password"
  type        = string
  default     = "changeme"
  sensitive   = true
}

variable "deletion_protection" {
  description = "Enable deletion protection"
  type        = bool
  default     = false
}