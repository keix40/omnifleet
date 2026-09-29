variable "project" {
  type    = string
  default = "omnifleet"
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "vpc_cidr" {
  type    = string
  default = "10.20.0.0/16"
}

variable "availability_zones" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b", "us-east-1c"]
}

variable "cluster_version" {
  type    = string
  default = "1.29"
}

variable "db_instance_class" {
  type    = string
  default = "db.r6g.large"
}

variable "db_read_replica_count" {
  type    = number
  default = 1
}

variable "terraform_plan_only" {
  type        = bool
  default     = true
  description = "When true, provider skips credential checks so terraform plan succeeds offline/CI."
}
