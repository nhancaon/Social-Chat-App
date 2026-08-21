output "cluster_name" {
  value = module.eks.cluster_name
}

output "cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "configure_kubectl" {
  description = "Run this after apply to point kubectl at the new cluster"
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name} --alias eks-lab"
}

output "files_bucket_name" {
  description = "S3 bucket for user-uploaded files — set as AWS_S3_BUCKET on the backend"
  value       = aws_s3_bucket.chat_app_files.bucket
}

output "backend_s3_access_key_id" {
  description = "AWS_ACCESS_KEY_ID for the backend — store in the backend-secrets K8s Secret, never commit it"
  value       = aws_iam_access_key.chat_app_backend.id
  sensitive   = true
}

output "backend_s3_secret_access_key" {
  description = "AWS_SECRET_ACCESS_KEY for the backend — store in the backend-secrets K8s Secret, never commit it"
  value       = aws_iam_access_key.chat_app_backend.secret
  sensitive   = true
}
