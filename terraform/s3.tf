# S3 bucket names are globally unique across all AWS accounts — a random suffix
# avoids collisions without the user having to pick a name themselves.
resource "random_id" "files_bucket_suffix" {
  byte_length = 4
}

resource "aws_s3_bucket" "chat_app_files" {
  bucket = "${var.cluster_name}-files-${random_id.files_bucket_suffix.hex}"
  tags   = var.tags
}

resource "aws_s3_bucket_public_access_block" "chat_app_files" {
  bucket = aws_s3_bucket.chat_app_files.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Browsers PUT/GET directly against S3 using the backend's presigned URLs, so the
# bucket itself — not the backend — has to answer the CORS preflight.
resource "aws_s3_bucket_cors_configuration" "chat_app_files" {
  bucket = aws_s3_bucket.chat_app_files.id

  cors_rule {
    allowed_methods = ["GET", "PUT"]
    allowed_origins = var.file_upload_cors_origins
    allowed_headers = ["*"]
    max_age_seconds = 3000
  }
}

# Production-grade fallback for anything the demo archival CronJob didn't already
# reclassify by hand, plus cleanup for incomplete multipart uploads left behind by
# interrupted client-side S3 PUTs. The CronJob (not this rule) is what makes the
# archival flow demoable on a schedule you control instead of AWS's ~daily cadence.
resource "aws_s3_bucket_lifecycle_configuration" "chat_app_files" {
  bucket = aws_s3_bucket.chat_app_files.id

  rule {
    id     = "archive-cold-files"
    status = "Enabled"

    filter {
      prefix = "uploads/"
    }

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

# Static IAM user + access key for the backend, matching how Mongo/Redis credentials
# are already handled in this lab (manual K8s Secret) instead of introducing IRSA as
# a second, inconsistent credentials pattern just for this one feature. A real
# production setup on EKS would use IRSA (pod-level IAM role via OIDC) instead of a
# long-lived static key.
resource "aws_iam_user" "chat_app_backend" {
  name = "${var.cluster_name}-backend-s3"
  tags = var.tags
}

resource "aws_iam_access_key" "chat_app_backend" {
  user = aws_iam_user.chat_app_backend.name
}

resource "aws_iam_user_policy" "chat_app_backend_s3" {
  name = "${var.cluster_name}-backend-s3-access"
  user = aws_iam_user.chat_app_backend.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ObjectAccess"
        Effect = "Allow"
        # CopyObject (used by the archival job) is authorized as GetObject on the
        # source + PutObject on the destination — there is no separate IAM action
        # for it.
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:DeleteObject",
          "s3:PutObjectTagging",
          "s3:RestoreObject",
        ]
        Resource = "${aws_s3_bucket.chat_app_files.arn}/*"
      },
      {
        Sid      = "BucketList"
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.chat_app_files.arn
      }
    ]
  })
}
