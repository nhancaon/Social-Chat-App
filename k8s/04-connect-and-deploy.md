# Nối EKS vào tầng quản lý + deploy app

Điều kiện trước khi làm bước này: đã xong `k8s/01-local-management.md`
(ArgoCD chạy local) **và** `terraform/02-rancher-host.md` (Rancher chạy
trên EC2, đã login được qua `https://<elastic-ip>.sslip.io`) **và**
`terraform/03-aws-eks.md` (cluster EKS đã `terraform apply` xong,
`kubectl get nodes` thấy node `Ready`).

## 0. Kết nối kubectl vào EKS

Chạy đúng lệnh in ra ở output `configure_kubectl` của `terraform apply`
(mục 2 trong `terraform/03-aws-eks.md`) — lệnh này thêm 1 context mới tên
`eks-lab` vào danh bạ context của `kubectl`, không đụng gì tới context
`docker-desktop` đang có:

```powershell
aws eks update-kubeconfig --region ap-southeast-1 --name social-chat-lab --alias eks-lab
kubectl config use-context eks-lab
kubectl get nodes
```

Kỳ vọng: 2 node `t3.medium`, `STATUS = Ready`. Thấy đúng vậy mới sang mục
0.5 — nếu `kubectl get nodes` lỗi `Unauthorized`, khả năng cao bạn đang
chạy lệnh bằng IAM user/profile khác với user đã `terraform apply` (xem
lại `enable_cluster_creator_admin_permissions` — chỉ cấp quyền cho đúng
danh tính đã tạo cluster).

## 0.5. Cài Metrics Server — bắt buộc cho HPA hoạt động

EKS **không tự cài `metrics-server`** (khác k3s trên Rancher host, có sẵn)
— thiếu nó, `backend-hpa` (mục 3) sẽ báo lỗi
`FailedGetResourceMetric: ... the server could not find the requested resource (get pods.metrics.k8s.io)`,
không đọc được CPU usage nên không tự scale được.

```powershell
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

Kiểm tra:

```powershell
kubectl get pods -n kube-system -l k8s-app=metrics-server
kubectl top pods -n chat-app
```

`kubectl top pods` ra được số liệu là đúng. Nếu Pod `metrics-server` bị
`CrashLoopBackOff` (đôi khi do chứng chỉ TLS tự ký của kubelet), thêm cờ
`--kubelet-insecure-tls` vào container args của Deployment
`metrics-server` (namespace `kube-system`).

## 1. Import EKS vào Rancher

Vào Rancher qua địa chỉ EC2 (`https://<elastic-ip>.sslip.io`, xem
`terraform/02-rancher-host.md`) → **Cluster Management** → **Import
Existing** → chọn **"Generic"** (mục "Import any Kubernetes cluster") —
**không chọn** "Amazon EKS" (mục "hosted Kubernetes provider" phía trên nó
tự tạo cluster EKS mới, không phải gắn vào cluster có sẵn, lại còn đòi
thêm AWS credentials cho Rancher, không cần thiết vì Terraform đã lo phần
đó).

Đặt tên `social-chat-lab` → **Create** → copy lệnh `kubectl apply` hiện
ra (lệnh này phải chứa đúng địa chỉ EC2, không phải `localhost` — vì
Rancher giờ có IP public thật ngay từ đầu, không cần cấu hình
`server-url` riêng như hồi còn dùng tunnel).

```powershell
kubectl config use-context eks-lab
curl.exe --insecure -sfL <url-rancher-vừa-đưa> | kubectl apply -f -
```

- **`curl.exe`** (không phải `curl` trơn — PowerShell alias `curl` thành
  `Invoke-WebRequest`, không hỗ trợ cờ dưới đây).
- **`--insecure`** — URL import vẫn mang chứng chỉ tự ký của Rancher lúc
  tải file YAML, bỏ qua bước kiểm tra chứng chỉ để tải được.

Đợi Pod agent ổn định:

```powershell
kubectl get pods -n cattle-system -w
```

Kỳ vọng `cattle-cluster-agent` chuyển `1/1 Running`, không rơi vào
`CrashLoopBackOff`. Quay lại Rancher UI, cluster chuyển `Active` sau
khoảng 1 phút — lần này không cần đụng tới `agent-tls-mode`, vì không có
proxy nào đứng giữa thay chứng chỉ như hồi dùng Cloudflare Tunnel nữa,
CA tự ký của Rancher pin theo mặc định (`strict`) là khớp đúng luôn.

## 2. Đăng ký EKS với ArgoCD

Chạy trên **máy Windows** (giống mục 1 — ArgoCD CLI và context `eks-lab`
đều chỉ có ở đây).

### 2.1 — Đảm bảo có ArgoCD CLI

Nếu chưa cài (khác `kubectl`/`helm`, không có trên winget): tải
`argocd-windows-amd64.exe` từ
[github.com/argoproj/argo-cd/releases](https://github.com/argoproj/argo-cd/releases),
đổi tên thành `argocd.exe`, bỏ vào 1 thư mục đã có trong PATH.

### 2.2 — Đăng nhập ArgoCD

Cần port-forward tới ArgoCD đang chạy (mở ở tab riêng, giữ chạy xuyên
suốt — giống cách làm với Rancher port-forward trước đây):

```powershell
kubectl config use-context docker-desktop
kubectl -n argocd port-forward svc/argocd-server 8080:443
```

Ở tab khác, đăng nhập CLI (dùng mật khẩu đã lấy ở `k8s/01-local-management.md` mục 2):

```powershell
argocd login localhost:8080
```

### 2.3 — Đăng ký EKS làm cluster đích

```powershell
kubectl config use-context eks-lab
argocd cluster add eks-lab --name eks-lab
```

`argocd cluster add` đọc context `eks-lab` từ kubeconfig **local** của
bạn (không phải chạy trên EKS) — nó tạo 1 Service Account bên trong EKS,
cấp quyền cho ArgoCD (đang chạy trên cluster local) gọi vào quản lý EKS
từ xa. `--name eks-lab` đặt tên gọi cố định, dùng để tham chiếu trong
Application bên dưới (nếu bỏ qua, ArgoCD tự đặt tên theo địa chỉ API
server, dài và khó nhớ hơn).

Kiểm tra đã đăng ký đúng:

```powershell
argocd cluster list
```

### 2.4 — Tạo Application trỏ vào Git repo

File `k8s/argocd-app.yaml` (đã có sẵn trong repo — đặt **ngoài**
`k8s/manifests/`, không phải bên trong, để tránh ArgoCD tự đưa luôn
chính Application này vào diện nó quản lý):

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: chat-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/nhancaon/Social-Chat-App.git
    targetRevision: main
    path: k8s/manifests
  destination:
    name: eks-lab
    namespace: chat-app
  syncPolicy:
    automated:
      selfHeal: true
      prune: true
    syncOptions:
      - CreateNamespace=true
```

Áp dụng (vẫn trên máy Windows, context nào cũng được vì Application là
resource của cluster **local** — nơi ArgoCD đang chạy):

```powershell
kubectl config use-context docker-desktop
kubectl apply -f k8s/argocd-app.yaml
```

`selfHeal: true` — nếu ai đó `kubectl edit` sửa tay trực tiếp trên EKS,
ArgoCD tự phát hiện lệch khỏi Git rồi sửa lại đúng theo Git. `prune: true`
— xoá file khỏi `k8s/manifests/` trong Git thì ArgoCD tự xoá resource
tương ứng trên EKS, không cần `kubectl delete` tay. `CreateNamespace=true`
— ArgoCD mặc định **không** tự tạo namespace đích, phải khai báo rõ mới
tự tạo `chat-app`, nếu không sẽ báo lỗi "namespace not found" khi sync
lần đầu.

Từ giờ, `git push` vào `k8s/manifests/` là ArgoCD tự đồng bộ lên EKS —
không cần `kubectl apply` tay nữa. `k8s/manifests/` hiện đang trống, sẽ
có nội dung thật ở mục 3 (Redis/Kafka/backend) — Application đã đăng ký
sẵn từ bây giờ, cứ để `Synced` với trạng thái rỗng cho tới khi có file.

## 3. Deploy MongoDB / Redis / Kafka / backend qua ArgoCD

5 file manifest đã có sẵn trong `k8s/manifests/` (`mongodb.yaml`,
`redis.yaml`, `kafka.yaml`, `kafka-ui.yaml`, `backend.yaml`) — tự host cả
5 trong cluster `chat-app`, không dùng dịch vụ ngoài (đơn giản cho lab,
không cần tạo thêm account Upstash/Atlas nào).

`kafka-ui` (y hệt tool đã dùng ở `backend/docker-compose.yml` cho dev
local) chỉ tạo `ClusterIP` — không public ra ngoài, xem qua
`port-forward` khi cần:

```powershell
kubectl config use-context eks-lab
kubectl -n chat-app port-forward svc/kafka-ui 8080:8080
```

Mở `http://localhost:8080` xem topic/message trực tiếp trên Kafka đang
chạy ở EKS.

### 3.1 — Tạo Secret trước (làm tay, không đưa vào Git)

Backend cần `MONGODB_URI` (có mật khẩu) và `JWT_SECRET` — đây là bí mật
thật, **không đặt trong file YAML commit lên Git** (đúng lý do đã giải
thích với `bootstrapPassword` của Rancher). Chạy trên máy Windows, context
trỏ vào `eks-lab`:

```powershell
kubectl config use-context eks-lab
kubectl create namespace chat-app

$mongoPass = -join ((48..57)+(65..90)+(97..122) | Get-Random -Count 24 | % {[char]$_})
$jwtSecret = -join ((48..57)+(65..90)+(97..122) | Get-Random -Count 32 | % {[char]$_})

kubectl create secret generic mongodb-secrets -n chat-app `
  --from-literal=MONGO_INITDB_ROOT_USERNAME=admin `
  --from-literal=MONGO_INITDB_ROOT_PASSWORD=$mongoPass

kubectl create secret generic backend-secrets -n chat-app `
  --from-literal=jwt-secret=$jwtSecret `
  --from-literal=mongodb-uri="mongodb://admin:$mongoPass@mongodb.chat-app.svc.cluster.local:27017"
```

Namespace `chat-app` tạo tay trước ở đây luôn (dù `CreateNamespace=true`
đã khai báo ở Application) — vì Secret cần namespace **đã tồn tại** để
tạo vào, mà thời điểm này ArgoCD chưa chắc đã sync lần nào.

### 3.2 — Đẩy manifest lên Git, để ArgoCD tự sync

```powershell
git add k8s/manifests/mongodb.yaml k8s/manifests/redis.yaml k8s/manifests/kafka.yaml k8s/manifests/kafka-ui.yaml k8s/manifests/backend.yaml
git commit -m "add app manifests for ArgoCD"
git push
```

ArgoCD tự phát hiện thay đổi trong `k8s/manifests/` (mặc định poll mỗi 3
phút, hoặc bấm **Refresh** trên UI để không phải đợi) → tự `sync` lên
`eks-lab`. Theo dõi qua UI (`https://localhost:8080`) hoặc CLI:

```powershell
argocd app get chat-app
```

### 3.3 — Lấy địa chỉ backend cho frontend Vercel

```powershell
kubectl config use-context eks-lab
kubectl get svc backend -n chat-app -w
```

Đợi cột `EXTERNAL-IP` có giá trị (~2 phút, AWS tự cấp Network Load
Balancer do `type: LoadBalancer` trong `backend.yaml`) — dùng đúng địa
chỉ đó làm `VUE_APP_API_URL` khi build frontend trên Vercel.

> **Chi phí**: NLB do `type: LoadBalancer` tự tạo cũng tính phí riêng
> (~$0.0225/giờ + data) — nhớ `kubectl delete svc backend -n chat-app`
> trước khi `terraform destroy` phần EKS, không thì NLB có thể bị "mồ
> côi", không tự xoá theo cluster.
