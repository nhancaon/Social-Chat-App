terraform {
  backend "s3" {
    bucket       = "social-chat-app-tfstate-43a40f5b" # output of `terraform apply` in terraform/bootstrap
    key          = "social-chat-app/eks/terraform.tfstate"
    region       = "ap-southeast-1"
    encrypt      = true
    use_lockfile = true # native S3 locking, no DynamoDB table needed
  }
}
