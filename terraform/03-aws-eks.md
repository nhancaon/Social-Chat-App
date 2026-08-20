# AWS EKS — dựng/xoá theo phiên lab

Khác với `k8s/01-local-management.md` (dựng 1 lần, giữ mãi) — cluster EKS ở
đây **dựng lên rồi xoá theo từng phiên lab** (~3 ngày) để tối ưu chi phí.
Rancher/ArgoCD ở tầng local sẽ **import/đăng ký** cluster này vào quản lý
sau khi nó đã chạy (xem `k8s/03-connect-and-deploy.md`).

## 0. Chuẩn bị tài khoản AWS & công cụ local (1 lần duy nhất)

### Cài công cụ

```powershell
winget install Amazon.AWSCLI --accept-source-agreements --accept-package-agreements
winget install Hashicorp.Terraform --accept-source-agreements --accept-package-agreements
```

Mở lại terminal, kiểm tra:

```powershell
aws --version
terraform -version
```

### Chuẩn bị trên AWS Console (làm tay)

1. **Bật MFA cho root** (Console &rarr; góc trên phải &rarr; Security credentials &rarr; MFA) &mdash; chỉ dùng root để làm các việc dưới đây, không dùng chạy lệnh hàng ngày.
2. **Tạo AWS Budget** (Billing Console &rarr; Budgets &rarr; đặt ngưỡng cảnh báo qua email, ví dụ $10) &mdash; lưới an toàn quan trọng nhất cho 1 lab hay quên xoá tài nguyên.
3. **Tạo IAM user riêng** cho Terraform (IAM &rarr; Users &rarr; Create user), gắn policy `AdministratorAccess`.
   > Chấp nhận được vì đây là account AWS cá nhân, chỉ mình bạn dùng cho lab &mdash; không phải khuyến nghị cho account công ty/nhiều người dùng chung.
4. **Tạo Access key** cho IAM user đó (Security credentials &rarr; Create access key &rarr; use case: Command Line Interface).

### `aws configure`

```powershell
aws configure
# AWS Access Key ID: <access-key-vừa-tạo>
# AWS Secret Access Key: <secret-vừa-tạo>
# Default region name: ap-southeast-1
```

Kiểm tra credentials hoạt động:

```powershell
aws sts get-caller-identity
```

Ra được `Account`/`UserId`/`Arn` là đã cấu hình đúng.

## 1. Bootstrap remote state (1 lần duy nhất)

Tạo bucket S3 lưu Terraform state &mdash; sống xuyên suốt mọi phiên lab sau
này, **không** bị xoá theo cluster.

```powershell
cd terraform/bootstrap
terraform init
terraform apply
# copy giá trị output "bucket_name"
```

Dán tên bucket đó vào `bucket = "..."` trong `terraform/backend.tf`.

## 2. Mỗi phiên lab — dựng cluster

```powershell
cd terraform
terraform init      # chỉ cần lại nếu backend.tf vừa đổi
terraform apply      # ~15-20 phút, phần lâu nhất là control plane EKS
```

Chạy đúng lệnh in ra ở output `configure_kubectl`, rồi kiểm tra:

```powershell
kubectl get nodes
```

Từ đây, sang `k8s/03-connect-and-deploy.md` để import vào Rancher, đăng ký
ArgoCD, deploy Kafka/Redis/backend.

## 3. Mỗi phiên lab — dọn dẹp

**Xoá mọi Service kiểu `type: LoadBalancer` (và Ingress nếu có) trước khi
destroy** &mdash; nếu không, cái ELB mà AWS tự tạo (Terraform không biết
tới) vẫn giữ VPC/subnet đang bận, khiến `terraform destroy` lỗi hoặc để sót
tài nguyên vẫn âm thầm tính phí:

```powershell
kubectl config use-context eks-lab
kubectl delete svc --all -n chat-app
kubectl delete svc --all -n monitoring
kubectl delete svc --all -n logging
kubectl delete pvc --all -n chat-app
# đợi ~1-2 phút cho AWS xoá xong ELB/EBS thật

cd terraform
terraform destroy
```

Bucket state ở bước 1 **không** bị xoá &mdash; dùng lại cho mọi lần
`terraform apply` sau này.
