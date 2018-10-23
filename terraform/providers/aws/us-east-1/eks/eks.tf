provider "random" {
  version = "=1.3.1"
}
data "aws_availability_zones" "available" {}
locals {
  cluster_name = "test-george"
  worker_groups = [
    {
      instance_type = "t2.small"
      subnets       = "${join(",", module.vpc.private_subnets)}"
    },
    {
      name                          = "management" 
      instance_type                 = "t2.small"
      subnets                       = "${join(",", module.vpc.private_subnets)}"
      additional_security_group_ids = "${aws_security_group.worker_group_mgmt_one.id}"
    },
    #   {
    #     asg_desired_capacity = 1
    #     asg_max_size = 5
    #     asg_min_size = 1
    #     instance_type = "m4.2xlarge"
    #     name = "worker_group_b"
    #     additional_userdata = "echo foo bar"
    #     subnets = "${join(",", module.vpc.private_subnets)}"
    #   },
  ]
  tags = {
    Environment = "test"
    GithubRepo  = "terraform-aws-eks"
    GithubOrg   = "terraform-aws-modules"
    Workspace   = "${terraform.workspace}"
  }
}
resource "aws_security_group" "worker_group_mgmt_one" {
  name_prefix = "worker_group_mgmt_one"
  description = "SG to be applied to all *nix machines"
  vpc_id      = "${module.vpc.vpc_id}"
  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"
    cidr_blocks = [
      "10.1.0.0/16",
    ]
  }
}
resource "aws_security_group" "all_worker_mgmt" {
  name_prefix = "all_worker_management"
  vpc_id      = "${module.vpc.vpc_id}"
  ingress {
    from_port = 22
    to_port   = 22
    protocol  = "tcp"
    cidr_blocks = [
      "10.1.0.0/16"
    ]
  }
}
module "vpc" {
  source             = "terraform-aws-modules/vpc/aws"
  version            = "1.45.0"
  name               = "test-eks-vpc"
  cidr               = "10.1.0.0/16"
  azs                = ["${data.aws_availability_zones.available.names[0]}", "${data.aws_availability_zones.available.names[1]}", "${data.aws_availability_zones.available.names[2]}"]
  private_subnets    = ["10.1.1.0/24", "10.1.2.0/24", "10.1.3.0/24"]
  public_subnets     = ["10.1.4.0/24", "10.1.5.0/24", "10.1.6.0/24"]
  enable_nat_gateway = true
  single_nat_gateway = true
  tags               = "${merge(local.tags, map("kubernetes.io/cluster/${local.cluster_name}", "shared"))}"
}
module "eks" {
  source                               = "terraform-aws-modules/eks/aws"
  version                              = "1.6.0"
  cluster_name                         = "${local.cluster_name}"
  subnets                              = ["${module.vpc.private_subnets}"]
  tags                                 = "${local.tags}"
  vpc_id                               = "${module.vpc.vpc_id}"
  worker_groups                        = "${local.worker_groups}"
  worker_group_count                   = "2"
  worker_additional_security_group_ids = ["${aws_security_group.all_worker_mgmt.id}"]
}
