# Cluster quản lý local — ArgoCD

Ghi lại đúng thứ tự lệnh + giải thích để sau này cài lại (máy mới, hoặc set
up lại sau khi nghỉ 1 thời gian) không phải nhớ lại/dò lỗi từ đầu.

Cluster này **sống lâu dài, tách biệt hoàn toàn** với cluster AWS EKS —
EKS dựng/xoá theo từng phiên lab (xem `terraform/03-aws-eks.md`), còn
cluster local này chỉ dựng 1 lần và giữ nguyên để chạy **ArgoCD**.

> **Rancher không còn cài ở đây nữa.** Rancher cần EKS "gọi ngược về" được
> nó (agent phone-home) — 1 cluster chỉ sống trên máy bạn (đứng sau NAT,
> không có IP public) không đáp ứng được việc đó ổn định. Rancher giờ
> chạy trên 1 máy EC2 nhỏ, luôn bật — xem `terraform/02-rancher-host.md`.
> ArgoCD thì khác: nó tự gọi ra API public của EKS, không cần EKS gọi
> ngược lại, nên **vẫn chạy local bình thường** như dưới đây.

Sau khi cả 2 tầng đã lên (cluster local này + Rancher host trên EC2 + EKS),
xem `k8s/04-connect-and-deploy.md` để nối chúng lại và deploy app.

## 0. Công cụ cần cài trước

```powershell
winget install Kubernetes.kubectl --accept-source-agreements --accept-package-agreements
```

(Docker Desktop cũng tự mang theo 1 bản `kubectl` khi cài — lệnh trên chỉ
để chắc chắn có `kubectl` dù bạn dùng máy nào.) Mở lại terminal sau khi
cài. Helm **không cần cài ở đây** — chỉ cần trên EC2 Rancher host
(`terraform/02-rancher-host.md`), vì ArgoCD ở file này cài thẳng bằng
`kubectl apply`, không qua Helm.

## 1. Tạo cluster Kubernetes local

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

## 2. ArgoCD

**Vì sao tải file `install.yaml` về repo (`k8s/argocd-install.yaml`) thay
vì `kubectl apply -f <url>` thẳng:**

1. URL kiểu `.../stable/manifests/install.yaml` trỏ vào nhánh `stable` —
   nội dung đổi theo thời gian mỗi khi ArgoCD ra bản mới. Cài hôm nay và
   cài lại 6 tháng sau có thể ra 2 phiên bản khác nhau mà không biết.
   Ghim vào **1 version cụ thể** (thay vì `stable`) mới tái lập được y hệt
   mỗi lần cài lại.
2. Đúng tinh thần GitOps đang theo đuổi: trạng thái hạ tầng nằm trong Git,
   không rải rác trong lệnh gõ tay/URL ngoài.

> **Đặt file này ở `k8s/argocd-install.yaml` — ngoài `k8s/manifests/`,
> không phải bên trong.** `k8s/manifests/` là thư mục Application
> `chat-app` (mục 2.4 ở `k8s/04-connect-and-deploy.md`) tự động theo dõi
> và sync toàn bộ nội dung lên EKS — nếu để file cài đặt ArgoCD lẫn vào
> đó, ArgoCD sẽ cố cài lại **chính nó** vào cluster EKS, gây lỗi CRD quá
> lớn (`262144 bytes`) y hệt lỗi đã gặp lúc cài ArgoCD lần đầu.

```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/argoproj/argo-cd/v3.5.1/manifests/install.yaml" -OutFile "k8s\argocd-install.yaml"

kubectl create namespace argocd
kubectl apply -n argocd -f k8s/argocd-install.yaml --server-side --force-conflicts
```

`--server-side` cần thiết vì CRD của ArgoCD quá lớn, vượt giới hạn 256KB
của annotation mà `kubectl apply` thường dùng. `--force-conflicts` phòng
trường hợp bạn từng chạy `kubectl apply` (không có `--server-side`) trước
đó rồi mới đổi sang `--server-side` — lúc đó các resource đã có "chủ sở
hữu" kiểu cũ (`kubectl-client-side-apply`), server-side apply mặc định
không tự giành quyền quản lý field của người khác, cần xác nhận rõ ràng
bằng flag này.

Đợi server chính lên khoẻ:

```powershell
kubectl -n argocd rollout status deploy/argocd-server
```

Mở UI:

```powershell
kubectl -n argocd port-forward svc/argocd-server 8080:443
```

Vào `https://localhost:8080` (sẽ bị cảnh báo chứng chỉ tự ký — bỏ qua,
bình thường vì đây là cluster chạy local). Đăng nhập bằng user `admin`,
lấy mật khẩu khởi tạo (ArgoCD tự sinh):

```powershell
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

Lệnh này lấy giá trị `password` từ 1 Kubernetes Secret (dữ liệu nhạy cảm
K8s lưu ở dạng mã hoá base64, không phải văn bản thường), rồi giải mã
base64 ra để đọc được. Windows PowerShell không có sẵn lệnh `base64` như
Linux/macOS, nên dùng hàm .NET (`System.Convert`) thay thế.

## Nâng cấp version ArgoCD sau này

Đổi số version trong URL ở bước tải file, tải đè lên
`k8s/argocd-install.yaml`, rồi `kubectl apply` lại — `git diff`
sẽ cho thấy chính xác thứ gì thay đổi giữa 2 bản trước khi bạn áp dụng.

## Lỗi hay gặp

| Lỗi | Nguyên nhân | Cách sửa |
|---|---|---|
| `metadata.annotations: Too long: may not be more than 262144 bytes` | CRD của ArgoCD quá lớn, không nhét vừa vào annotation `last-applied-configuration` của `kubectl apply` thường | Thêm `--server-side` vào lệnh apply |
| `Apply failed with 1 conflict: conflict with "kubectl-client-side-apply"` | Đã lỡ `apply` không có `--server-side` trước đó, giờ đổi cách apply bị vướng "chủ sở hữu" cũ | Thêm `--force-conflicts` vào lệnh apply |
