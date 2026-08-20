terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "aws" {
  region = var.region
}

# Reuse the account's default VPC — this box must survive every EKS
# lab teardown/rebuild cycle, so it can't live in terraform/main.tf's
# VPC (destroyed each cycle). A single instance doesn't warrant its own
# VPC/NAT/route tables.
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "random_password" "rancher_bootstrap" {
  length  = 20
  special = false
}

# --- Access: SSM Session Manager, no SSH key pair, no port 22 ---
resource "aws_iam_role" "rancher_host" {
  name = "${var.name}-ssm-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })

  tags = var.tags
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.rancher_host.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "rancher_host" {
  name = "${var.name}-ssm-profile"
  role = aws_iam_role.rancher_host.name
}

# --- Networking ---
resource "aws_security_group" "rancher_host" {
  name        = "${var.name}-sg"
  description = "Rancher host - HTTPS in, no SSH (access via SSM instead)"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "Rancher UI + cattle-cluster-agent registration"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = var.tags
}

# --- The box itself ---
resource "aws_instance" "rancher_host" {
  ami                         = data.aws_ami.ubuntu.id
  instance_type               = var.instance_type
  subnet_id                   = data.aws_subnets.default.ids[0]
  vpc_security_group_ids      = [aws_security_group.rancher_host.id]
  iam_instance_profile        = aws_iam_instance_profile.rancher_host.name
  associate_public_ip_address = true

  root_block_device {
    volume_size = 20 # default 8GB is tight for k3s + Rancher + cert-manager images
    volume_type = "gp3"
  }

  tags = merge(var.tags, { Name = var.name })
}

resource "aws_eip" "rancher_host" {
  domain = "vpc"
  tags   = var.tags
}

resource "aws_eip_association" "rancher_host" {
  instance_id   = aws_instance.rancher_host.id
  allocation_id = aws_eip.rancher_host.id
}
