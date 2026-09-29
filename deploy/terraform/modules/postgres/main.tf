resource "aws_security_group" "postgres" {
  name        = "${var.project}-postgres"
  description = "OmniFleet Postgres"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["10.20.0.0/16"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.project}-postgres"
  subnet_ids = var.private_subnet_ids
}

resource "aws_rds_cluster" "this" {
  cluster_identifier = "${var.project}-aurora"
  engine             = "aurora-postgresql"
  engine_mode        = "provisioned"
  engine_version     = "16.1"
  database_name      = "omnifleet"
  master_username    = "omnifleet"
  manage_master_user_password = true
  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.postgres.id]
  storage_encrypted      = true
  backup_retention_period = 7
}

resource "aws_rds_cluster_instance" "primary" {
  identifier         = "${var.project}-aurora-primary"
  cluster_identifier = aws_rds_cluster.this.id
  instance_class     = var.instance_class
  engine             = aws_rds_cluster.this.engine
  engine_version     = aws_rds_cluster.this.engine_version
}

resource "aws_rds_cluster_instance" "replica" {
  count              = var.read_replica_count
  identifier         = "${var.project}-aurora-replica-${count.index + 1}"
  cluster_identifier = aws_rds_cluster.this.id
  instance_class     = var.instance_class
  engine             = aws_rds_cluster.this.engine
  engine_version     = aws_rds_cluster.this.engine_version
}
