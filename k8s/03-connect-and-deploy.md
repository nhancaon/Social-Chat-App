# Nối EKS vào tầng quản lý local + deploy app

Điều kiện trước khi làm bước này: đã xong `k8s/01-local-management.md`
(Rancher + ArgoCD đang chạy local) **và** `terraform/02-aws-eks.md`
(cluster EKS đã `terraform apply` xong, `kubectl get nodes` thấy node
`Ready`).

## 1. Import EKS vào Rancher

Rancher UI &rarr; **Cluster Management** &rarr; **Import Existing** &rarr;
đặt tên `social-chat-lab` &rarr; copy lệnh `kubectl apply` Rancher đưa ra,
chạy đúng lệnh đó với context đang trỏ vào EKS (không phải
`docker-desktop`):

```powershell
kubectl config use-context eks-lab
kubectl apply -f https://rancher.local/v3/import/<token-rancher-cap>.yaml
```

## 2. Đăng ký EKS với ArgoCD

_(sẽ điền chi tiết khi tới bước này)_

## 3. Deploy Redis / Kafka / backend qua ArgoCD

_(sẽ điền chi tiết khi tới bước này — xem thêm `k8s/manifests/`)_
