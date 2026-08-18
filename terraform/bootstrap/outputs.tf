output "bucket_name" {
  description = "Paste this into terraform/backend.tf's bucket field"
  value       = aws_s3_bucket.tfstate.bucket
}
