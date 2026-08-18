# Cluster quản lý local — Rancher + ArgoCD

Ghi lại đúng thứ tự lệnh + giải thích để sau này cài lại (máy mới, hoặc set
up lại sau khi nghỉ 1 thời gian) không phải nhớ lại/dò lỗi từ đầu.

Cluster này **sống lâu dài, tách biệt hoàn toàn** với cluster AWS EKS —
EKS dựng/xoá theo từng phiên lab (xem `terraform/02-aws-eks.md`), còn
cluster này chỉ dựng 1 lần và giữ nguyên để chạy Rancher/ArgoCD quản lý.
Sau khi cả 2 tầng đã lên, xem `k8s/03-connect-and-deploy.md` để nối chúng
lại và deploy app.

## 0. Tạo cluster Kubernetes local

Dùng Kubernetes tích hợp sẵn trong Docker Desktop (Settings → Kubernetes →
Enable), chọn kiểu `kubeadm`, 1 node. Không cần cài thêm công cụ nào khác
cho bước này (không cần `k3d`/`kind` riêng).

Kiểm tra `kubectl` đã trỏ đúng cluster:

```powershell
kubectl config get-contexts     # liệt kê mọi cluster kubectl biết (như danh bạ điện thoại)
kubectl config use-context docker-desktop   # chọn "số" đang gọi là cluster docker-desktop
kubectl get nodes               # hỏi cluster: node có đang Ready không
```

Kỳ vọng: 1 node, `STATUS = Ready`.

## 1. cert-manager — bắt buộc cài trước Rancher

**Vì sao cần:** Rancher mặc định tự tạo chứng chỉ HTTPS cho chính nó bằng
cách tạo ra 1 resource loại `Issuer` — nhưng `Issuer` không phải loại
resource có sẵn trong Kubernetes, nó do `cert-manager` định nghĩa thêm vào
(gọi là CRD — Custom Resource Definition). Không cài `cert-manager` trước
thì Rancher cài sẽ lỗi ngay: `no matches for kind "Issuer"`.

```powershell
kubectl create namespace cert-manager

helm repo add jetstack https://charts.jetstack.io
helm repo update

helm install cert-manager jetstack/cert-manager `
  --namespace cert-manager `
  -f k8s/helm-values/cert-manager.yaml
```

Đợi cả 3 phần đều lên khoẻ (lệnh dưới sẽ tự đứng chờ tới khi xong, không
phải bị treo):

```powershell
kubectl -n cert-manager rollout status deploy/cert-manager
kubectl -n cert-manager rollout status deploy/cert-manager-webhook
kubectl -n cert-manager rollout status deploy/cert-manager-cainjector
```

Cả 3 dòng đều ra `successfully rolled out` mới sang bước 2.

## 2. Rancher

**Vì sao cần từng thông số trong `k8s/helm-values/rancher.yaml`:**

| Thông số | Ý nghĩa |
|---|---|
| `hostname: rancher.local` | Tên miền Rancher dùng để tạo cert/URL nội bộ — chỉ mang tính hình thức vì bạn truy cập qua `port-forward`, không qua DNS thật |
| `bootstrapPassword: admin` | Mật khẩu đăng nhập lần đầu — set cứng để khỏi phải đào `kubectl logs` tìm mật khẩu ngẫu nhiên. Chỉ chấp nhận được vì cluster này chạy local, không expose ra internet. Rancher bắt đổi mật khẩu thật ngay lần đăng nhập đầu |
| `replicas: 1` | Chạy 1 bản sao Rancher — cluster chỉ có 1 node nên chạy nhiều bản không tăng độ tin cậy, chỉ tốn tài nguyên |

```powershell
kubectl create namespace cattle-system
helm repo add rancher-latest https://releases.rancher.com/server-charts/latest
helm repo update

helm install rancher rancher-latest/rancher `
  --namespace cattle-system `
  -f k8s/helm-values/rancher.yaml
```

**Vì sao namespace tên `cattle-system`:** không phải mình tự đặt — đây là
namespace bắt buộc theo tài liệu cài đặt chính thức của Rancher (tên
"cattle" là di sản từ engine điều phối nội bộ đời Rancher 1.x, giữ lại làm
quy ước đặt tên tới tận bây giờ).

Đợi Rancher lên khoẻ:

```powershell
kubectl -n cattle-system rollout status deploy/rancher
```

## 3. Mở giao diện Rancher

```powershell
kubectl -n cattle-system port-forward svc/rancher 8443:443
```

Mở trình duyệt vào `https://localhost:8443`, đăng nhập bằng
`bootstrapPassword` đã set ở bước 2 (`admin`), sau đó Rancher sẽ bắt đặt
mật khẩu thật.

`port-forward` = mở 1 đường ống tạm nối cổng `8443` trên máy bạn thẳng vào
cổng `443` của Service `rancher` trong cluster — chỉ có tác dụng trong lúc
lệnh này còn chạy (đóng terminal là mất kết nối, chạy lại lệnh là nối lại
được ngay, không cần cài lại gì).

## Lỗi hay gặp

| Lỗi | Nguyên nhân | Cách sửa |
|---|---|---|
| `repo rancher-latest not found` | Chưa `helm repo add` trước khi `helm install` | Chạy `helm repo add` + `helm repo update` trước |
| `no matches for kind "Issuer"` | Cài Rancher trước khi cài `cert-manager` | Cài `cert-manager` (mục 1) trước, rồi mới cài Rancher |
| `metadata.annotations: Too long: may not be more than 262144 bytes` | CRD của ArgoCD quá lớn, không nhét vừa vào annotation `last-applied-configuration` của `kubectl apply` thường | Thêm `--server-side` vào lệnh apply |
| `Apply failed with 1 conflict: conflict with "kubectl-client-side-apply"` | Đã lỡ `apply` không có `--server-side` trước đó, giờ đổi cách apply bị vướng "chủ sở hữu" cũ | Thêm `--force-conflicts` vào lệnh apply |

## 4. ArgoCD

**Vì sao tải file `install.yaml` về repo (`k8s/manifests/argocd-install.yaml`)
thay vì `kubectl apply -f <url>` thẳng:**

1. URL kiểu `.../stable/manifests/install.yaml` trỏ vào nhánh `stable` —
   nội dung đổi theo thời gian mỗi khi ArgoCD ra bản mới. Cài hôm nay và
   cài lại 6 tháng sau có thể ra 2 phiên bản khác nhau mà không biết.
   Ghim vào **1 version cụ thể** (thay vì `stable`) mới tái lập được y hệt
   mỗi lần cài lại.
2. Đúng tinh thần GitOps đang theo đuổi: trạng thái hạ tầng nằm trong Git,
   không rải rác trong lệnh gõ tay/URL ngoài.

```powershell
mkdir k8s\manifests -Force
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/argoproj/argo-cd/v3.5.1/manifests/install.yaml" -OutFile "k8s\manifests\argocd-install.yaml"

kubectl create namespace argocd
kubectl apply -n argocd -f k8s/manifests/argocd-install.yaml --server-side 
```

`--force-conflicts` phòng trường hợp bạn từng chạy `kubectl apply` (không có
`--server-side`) trước đó rồi mới đổi sang `--server-side` — lúc đó các
resource đã có "chủ sở hữu" kiểu cũ (`kubectl-client-side-apply`), server-side
apply mặc định không tự giành quyền quản lý field của người khác, cần xác
nhận rõ ràng bằng flag này.

Đợi server chính lên khoẻ:

```powershell
kubectl -n argocd rollout status deploy/argocd-server
```

Mở UI (đường ống riêng, cổng khác Rancher để chạy song song được):

```powershell
kubectl -n argocd port-forward svc/argocd-server 8080:443
```

Vào `https://localhost:8080` (cũng sẽ bị cảnh báo chứng chỉ tự ký giống
Rancher — bỏ qua như đã làm ở bước Rancher). Đăng nhập bằng user `admin`,
lấy mật khẩu khởi tạo (ArgoCD tự sinh, không đặt cứng như Rancher):

```powershell
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

Lệnh này lấy giá trị `password` từ 1 Kubernetes Secret (dữ liệu nhạy cảm
K8s lưu ở dạng mã hoá base64, không phải văn bản thường), rồi giải mã
base64 ra để đọc được.

## Nâng cấp version ArgoCD sau này

Đổi số version trong URL ở bước tải file, tải đè lên
`k8s/manifests/argocd-install.yaml`, rồi `kubectl apply` lại — `git diff`
sẽ cho thấy chính xác thứ gì thay đổi giữa 2 bản trước khi bạn áp dụng.
