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

Backend cần `MONGODB_URI` (có mật khẩu), `JWT_SECRET`, và (từ khi thêm
tính năng file storage) bộ credential S3 — đây đều là bí mật thật,
**không đặt trong file YAML commit lên Git** (đúng lý do đã giải thích
với `bootstrapPassword` của Rancher). `terraform/s3.tf` nằm chung root
module với EKS/VPC nên chỉ cần chạy lại đúng lệnh `terraform apply` ban
đầu (không phải lệnh mới) là bucket S3 + IAM user sẽ được tạo thêm cùng
lúc. Lấy giá trị ra bằng `terraform output`:

```powershell
cd terraform
terraform apply   # lệnh cũ — chạy lại để tạo thêm bucket S3 + IAM user
$bucketName = terraform output -raw files_bucket_name
$awsKeyId   = terraform output -raw backend_s3_access_key_id
$awsSecret  = terraform output -raw backend_s3_secret_access_key
cd ..
```

Rồi tạo Secret. Chạy trên máy Windows, context trỏ vào `eks-lab`:

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
  --from-literal=mongodb-uri="mongodb://admin:$mongoPass@mongodb.chat-app.svc.cluster.local:27017" `
  --from-literal=aws-s3-bucket=$bucketName `
  --from-literal=aws-access-key-id=$awsKeyId `
  --from-literal=aws-secret-access-key=$awsSecret
```

Namespace `chat-app` tạo tay trước ở đây luôn (dù `CreateNamespace=true`
đã khai báo ở Application) — vì Secret cần namespace **đã tồn tại** để
tạo vào, mà thời điểm này ArgoCD chưa chắc đã sync lần nào.

> **Nếu `backend-secrets` đã tồn tại sẵn từ trước** (đã deploy
> MongoDB/backend trong lần lab trước rồi) — **đừng** chạy lại
> `kubectl create secret` ở trên, nó sẽ báo lỗi "already exists". Chỉ cần
> **thêm** 3 key AWS mới vào Secret sẵn có, giữ nguyên `jwt-secret`/
> `mongodb-uri` cũ (đổi `jwt-secret` sẽ làm mọi user đang login bị logout
> hết):
>
> ```powershell
> $patch = @{ stringData = @{
>     "aws-s3-bucket"          = $bucketName
>     "aws-access-key-id"      = $awsKeyId
>     "aws-secret-access-key"  = $awsSecret
> } } | ConvertTo-Json -Compress
>
> # ghi ra file tạm rồi dùng --patch-file, KHÔNG dùng -p trực tiếp —
> # PowerShell gọi kubectl.exe (chương trình ngoài) thường làm hỏng dấu
> # ngoặc kép trong chuỗi JSON truyền qua tham số dòng lệnh
> $patchFile = New-TemporaryFile
> Set-Content -Path $patchFile -Value $patch -Encoding utf8 -NoNewline
> kubectl patch secret backend-secrets -n chat-app --type merge --patch-file $patchFile
> Remove-Item $patchFile
> ```
>
> `stringData` là trường ghi bằng plain text — Kubernetes tự base64-encode
> và gộp vào `data`, không cần tự mã hoá tay.

### 3.2 — Đẩy manifest lên Git, để ArgoCD tự sync

```powershell
git add k8s/manifests/mongodb.yaml k8s/manifests/redis.yaml k8s/manifests/kafka.yaml k8s/manifests/kafka-ui.yaml k8s/manifests/backend.yaml k8s/manifests/file-jobs.yaml
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

### 3.4 — Tính năng file storage (S3 + Glacier archival)

`k8s/manifests/file-jobs.yaml` khai báo 3 `CronJob`, dùng chung image
backend với `-job=<tên>` thay vì chạy HTTP server:

| CronJob | Lịch | Việc làm |
|---|---|---|
| `file-archive-scan` | 03:00 hằng ngày | file STANDARD không đụng tới quá `ARCHIVE_AFTER_HOURS` → chuyển Glacier |
| `file-trash-purge` | 03:30 hằng ngày | file trong trash quá `TRASH_RETENTION_HOURS` → xoá thật, hoàn quota |
| `file-restore-poll` | mỗi 5 phút | check file đang `RESTORING` đã xong chưa, xong thì báo qua Kafka `notifications` (dùng lại đúng đường dẫn WebSocket của chat/notification) |

API cho frontend: `POST /files/upload-url` → `POST /files/:id/confirm` →
(client upload thẳng S3, backend không proxy byte nào) → `GET /files`,
`GET /files/:id/download-url`, `POST /files/:id/restore` (nếu file đã
Glacier), `DELETE /files/:id` (soft-delete/trash).

Muốn demo archival ngay không đợi 90 ngày thật — **lưu ý**: sửa
`ARCHIVE_AFTER_HOURS` bằng `kubectl set env` (không commit lên Git) sẽ
bị ArgoCD tự revert lại gần như ngay lập tức, vì `chat-app` đang chạy
`Automated` + `selfHeal` (ArgoCD watch trực tiếp API, không phải đợi
poll 3 phút). Cách không bị giành lại: backdate `lastAccessedAt` của
đúng file demo trong Mongo, để nó tự đủ điều kiện theo chính sách thật
90 ngày — không đụng gì tới config nên không có gì để ArgoCD revert.

**Bước 1 — lấy `_id` của file muốn demo:**

```powershell
$mongoUriB64 = kubectl get secret backend-secrets -n chat-app -o jsonpath='{.data.mongodb-uri}'
$mongoUri = [System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String($mongoUriB64))

kubectl run mongo-shell --rm -i --restart=Never --image=mongo:7 -n chat-app -- `
  mongosh $mongoUri --quiet --eval `
  'db.getSiblingDB("social-chat").files.find({uploaded:true, isDeleted:false}, {filename:1, storageClass:1, lastAccessedAt:1}).toArray()'
```

Copy đúng `_id` của file cần demo (dạng `ObjectId('...')`, chỉ lấy phần
chuỗi hex bên trong).

**Bước 2 — backdate `lastAccessedAt` về 100 ngày trước:**

```powershell
$fileId = "<dán _id vừa copy, không kèm chữ ObjectId>"

kubectl run mongo-shell2 --rm -i --restart=Never --image=mongo:7 -n chat-app -- `
  mongosh $mongoUri --quiet --eval `
  "db.getSiblingDB('social-chat').files.updateOne({_id: ObjectId('$fileId')}, {`$set: {lastAccessedAt: new Date(Date.now() - 100*24*60*60*1000)}})"
```

**Bước 3 — chạy job archive tay:**

```powershell
kubectl create job --from=cronjob/file-archive-scan file-archive-scan-manual -n chat-app
kubectl logs -n chat-app job/file-archive-scan-manual -f
```

Kỳ vọng thấy `archive-scan: done — 1 archived, 0 failed, 1 eligible`.
Verify thật trên S3 (không chỉ tin ở log):

```powershell
aws s3api head-object --bucket <bucket-name> --key "uploads/<userId>/<fileId>-<filename>" --query "StorageClass" --output text
```
phải ra `GLACIER`.

Riêng biến `RESTORE_TIER=Expedited` (đẩy nhanh restore ~1-5 phút cho
demo, thay vì vài giờ) nằm trên **Deployment `backend`** đang chạy
24/7 — không có cách né như trên, nếu cần chỉnh phải tạm khoá auto-sync
trước:

```powershell
argocd app set chat-app --sync-policy none
kubectl set env deployment/backend -n chat-app RESTORE_TIER=Expedited
kubectl rollout status deployment backend -n chat-app
```

**Nhớ bật lại auto-sync ngay sau khi quay/test xong** — đây là bước dễ
quên nhất, quên thì mọi thay đổi tay sau đó không còn được ArgoCD tự
sync/self-heal bảo vệ nữa:

```powershell
argocd app set chat-app --sync-policy automated --self-heal --auto-prune
```

## 4. Dọn dẹp / Teardown — xoá sạch hạ tầng AWS

Xong lab/demo và muốn ngừng tính phí — xoá theo **đúng thứ tự** này,
sai thứ tự dễ để sót resource "mồ côi" (không bị Terraform quản lý
nữa nhưng vẫn âm thầm tính tiền), hoặc khiến `terraform destroy` báo
lỗi giữa chừng.

**Bước 1 — xoá Service LoadBalancer trước.** Đây là ELB do EKS in-tree
cloud provider tự tạo khi thấy `type: LoadBalancer` trong
`backend.yaml` — Terraform không hề biết tới nó (được tạo ra bởi
Kubernetes, không phải bởi `terraform apply`), nên `terraform destroy`
**không tự xoá** — nếu bỏ qua bước này, ELB sẽ nằm lại vĩnh viễn, tính
phí đều đều mà không nằm trong bất kỳ state Terraform nào để dọn sau.

```powershell
kubectl config use-context eks-lab
kubectl delete svc backend -n chat-app
```

**Bước 2 — xoá sạch object trong S3 bucket.** Bucket S3 do Terraform
tạo (`terraform/s3.tf`) không bật `force_destroy`, nên nếu còn file
bên trong, `terraform destroy` sẽ dừng lại báo lỗi "bucket not empty"
giữa chừng.

```powershell
aws s3 rm s3://<bucket-name> --recursive
```

**Bước 3 — destroy phần chính** (EKS, VPC, NAT Gateway, S3 bucket, IAM
user cho file storage):

```powershell
cd terraform
terraform destroy
```

**Bước 4 — destroy Rancher host** (EC2 riêng, module Terraform độc
lập, không phụ thuộc vào bước 3 nên có thể chạy song song):

```powershell
cd terraform/rancher-host
terraform destroy
```

**Không cần đụng tới** khi teardown (không tính phí AWS hoặc thuộc
dịch vụ/billing khác, không liên quan tới uptime cluster):
- ArgoCD + cluster local Docker Desktop — chạy trên máy bạn, không tốn phí AWS
- Domain `ncnhan.uk` (Cloudflare Registrar) — tính phí theo năm, không liên quan uptime
- Frontend trên Vercel — free tier, không tính theo uptime backend

Sau khi cả 2 lệnh `terraform destroy` chạy xong, kiểm tra lại
[AWS Billing Console](https://us-east-1.console.aws.amazon.com/costmanagement/home)
hoặc chạy `aws eks list-clusters` / `aws ec2 describe-instances` để
chắc chắn không còn resource nào sót lại.
