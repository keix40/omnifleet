terraform {
  required_version = ">= 1.6.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  # CI and local dev run `terraform plan` without cloud credentials (validate-only).
  access_key                  = var.terraform_plan_only ? "mock" : null
  secret_key                  = var.terraform_plan_only ? "mock" : null
  skip_credentials_validation = var.terraform_plan_only
  skip_requesting_account_id  = var.terraform_plan_only
  skip_metadata_api_check     = var.terraform_plan_only
}

module "network" {
  source               = "./modules/network"
  project              = var.project
  vpc_cidr             = var.vpc_cidr
  availability_zones   = var.availability_zones
}

module "eks" {
  source             = "./modules/eks"
  project            = var.project
  vpc_id             = module.network.vpc_id
  private_subnet_ids = module.network.private_subnet_ids
  cluster_version    = var.cluster_version
}

module "postgres" {
  source             = "./modules/postgres"
  project            = var.project
  vpc_id             = module.network.vpc_id
  private_subnet_ids = module.network.private_subnet_ids
  instance_class     = var.db_instance_class
  read_replica_count = var.db_read_replica_count
}

output "vpc_id" {
  value = module.network.vpc_id
}

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "postgres_endpoint" {
  value = module.postgres.primary_endpoint
}

output "postgres_read_replica_endpoints" {
  value = module.postgres.read_replica_endpoints
}
