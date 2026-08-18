terraform {
  backend "s3" {
    bucket       = "REPLACE_WITH_BOOTSTRAP_OUTPUT" # output of `terraform apply` in terraform/bootstrap
    key          = "social-chat-app/eks/terraform.tfstate"
    region       = "ap-southeast-1"
    encrypt      = true
    use_lockfile = true # native S3 locking, no DynamoDB table needed
  }
}
