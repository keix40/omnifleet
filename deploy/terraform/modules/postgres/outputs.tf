output "primary_endpoint" {
  value = aws_rds_cluster.this.endpoint
}

output "read_replica_endpoints" {
  value = aws_rds_cluster_instance.replica[*].endpoint
}
