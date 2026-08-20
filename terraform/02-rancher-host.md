# Rancher host — persistent EC2, thay cho local + Cloudflare Tunnel

Trước đây Rancher chạy trên máy local (`k8s/01-local-management.md`) và
cần Cloudflare Tunnel để EKS gọi ngược về được (`k8s/04-connect-and-deploy.md`)
— nhược điểm: địa chỉ tunnel đổi mỗi lần chạy lại, Rancher chỉ sống khi máy
bạn bật + script tunnel đang chạy.

File này dựng **1 máy EC2 nhỏ, chạy 24/7, không phụ thuộc máy bạn** — cài
k3s + Rancher lên đó, có địa chỉ IP tĩnh thật, không cần tunnel nữa.

> **Chi phí thật, không miễn phí**: `t3.medium` (4GB RAM) chạy liên tục
> ~$0.0416/giờ ở `ap-southeast-1`, khoảng **$30/tháng** nếu để chạy xuyên
> suốt — trừ dần vào $100 credit của account, không phải free tier 12
> tháng kiểu cũ. Khác hẳn phần EKS (dựng/xoá theo phiên lab), máy này
> **không** thuộc quy trình teardown — cứ để chạy liên tục, không cần xoá
> đi dựng lại.
>
> Ban đầu thử `t3.micro` (1GB RAM) nhưng máy **quá tải, đơ hẳn** (mất kết
> nối SSM, không phản hồi) ngay khi vừa cài xong k3s + chuẩn bị cài Helm.
> Chọn thẳng `t3.medium` (4GB) thay vì thử dần `t3.small` (2GB) — khớp
> đúng mức tối thiểu Rancher tự khuyến nghị, tránh mất thêm 1 vòng debug
> nếu 2GB vẫn chưa đủ.

## 0. Đổi loại máy (chỉ cần nếu máy đã dựng rồi và muốn nâng cấp)

```powershell
cd terraform/rancher-host
terraform apply
```

Terraform tự dừng máy, đổi `instance_type`, khởi động lại — không mất dữ
liệu đã cài (k3s vẫn còn nguyên, tự khởi động lại qua systemd). Elastic
IP giữ nguyên không đổi vì gắn qua `aws_eip_association` riêng, không phụ
thuộc loại máy.

## 1. Dựng EC2

```powershell
cd terraform/rancher-host
terraform init
terraform apply
```

Lấy 2 giá trị từ output:

```powershell
terraform output public_ip
terraform output -raw rancher_bootstrap_password
terraform output ssm_connect_command
```

## 2. Kết nối vào máy — qua SSM, không cần SSH key

Cần cài thêm 1 plugin riêng cho `aws` CLI (không có sẵn trong bản cài gốc):

```powershell
winget install Amazon.SessionManagerPlugin --accept-source-agreements --accept-package-agreements
```

> Installer này **không tự thêm vào PATH** (khác đa số gói winget khác) —
> nếu mở terminal mới vẫn báo `SessionManagerPlugin is not found`, thêm
> tay:
> ```powershell
> $pluginPath = "C:\Program Files\Amazon\SessionManagerPlugin\bin"
> $p = [Environment]::GetEnvironmentVariable("PATH", "User")
> [Environment]::SetEnvironmentVariable("PATH", "$p;$pluginPath", "User")
> ```
> rồi mở lại terminal mới.

Sau khi cài (và thêm PATH nếu cần), kết nối:

```powershell
aws ssm start-session --target <instance-id> --region ap-southeast-1
```

(đúng câu lệnh in ra ở output `ssm_connect_command`). Máy này **không mở
port 22** — Security Group chỉ mở port 443 (cho Rancher UI + agent EKS gọi
vào), không có đường SSH nào để mở cả, giảm bề mặt tấn công cho 1 máy có
IP public thật 24/7.

## 3. Cài k3s

```bash
curl -sfL https://get.k3s.io | sh -
sudo k3s kubectl get nodes
```

`k3s` = bản Kubernetes 1-node siêu nhẹ (khác `kubeadm` dùng ở cluster local
— phù hợp hơn cho 1 máy nhỏ chỉ chạy đúng Rancher). Cài xong tự chạy luôn,
không cần thêm bước khởi động nào khác.

Đặt biến để không phải gõ `sudo k3s kubectl` mỗi lần:

```bash
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

# ghi vào ~/.bashrc thay vì chỉ export tay — export thường chỉ sống trong
# đúng session hiện tại, mất hết khi mở session SSM mới (VD sau khi
# reboot/resize máy) — dẫn tới lỗi "permission denied" vì kubectl quay về
# đọc /etc/rancher/k3s/k3s.yaml mặc định (chỉ root đọc được)
echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc
source ~/.bashrc

kubectl get nodes
```

## 4. Cài Helm

```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
helm version
```

## 5. cert-manager + Rancher

Các lệnh dưới đây chạy **trên chính EC2** (qua session SSM), máy đó không
có sẵn repo Git này — nên toàn bộ giá trị đều truyền tay qua `--set`,
không dùng file `-f` được. `crds.enabled=true` (cert-manager) không phải
bí mật, chỉ là gõ tay vì không có file để trỏ tới. `hostname`/
`bootstrapPassword` (Rancher) thì khác — đổi mỗi lần dựng máy mới (IP
tĩnh khác, mật khẩu Terraform sinh ngẫu nhiên) và là bí mật thật, nên dù
có repo trên máy cũng không nên đưa vào file commit.

```bash
kubectl create namespace cert-manager
helm repo add jetstack https://charts.jetstack.io
helm repo update
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --set crds.enabled=true

kubectl -n cert-manager rollout status deploy/cert-manager
kubectl -n cert-manager rollout status deploy/cert-manager-webhook
kubectl -n cert-manager rollout status deploy/cert-manager-cainjector

kubectl create namespace cattle-system
helm repo add rancher-latest https://releases.rancher.com/server-charts/latest
helm repo update
helm install rancher rancher-latest/rancher \
  --namespace cattle-system \
  --set hostname=<IP-tĩnh>.sslip.io \
  --set bootstrapPassword='<password-terraform-sinh-ra>' \
  --set replicas=1

kubectl -n cattle-system rollout status deploy/rancher
```

Kubernetes Ingress **không chấp nhận IP thô làm hostname** (bắt buộc phải
là DNS name) — dùng **sslip.io**, dịch vụ DNS miễn phí tự động phân giải
`<IP>.sslip.io` về đúng IP đó, không cần đăng ký domain thật. Rancher đã
tính sẵn trường hợp này (biến `CATTLE_INGRESS_IP_DOMAIN=sslip.io` mặc
định trong chart).

## 6. Mở Rancher

Mở thẳng trình duyệt vào `https://"IP".sslip.io` (không phải chỉ
IP trơn) — **không cần `port-forward`, không cần tunnel gì cả**, vì máy
này có IP public thật. Vẫn sẽ gặp cảnh báo chứng chỉ tự ký (giống trước)
— bỏ qua như đã quen.

Từ đây, quay lại `k8s/04-connect-and-deploy.md` để import cluster EKS vào
Rancher — không còn bước Cloudflare Tunnel nữa, `agent-tls-mode` cũng giữ
mặc định `strict` là chạy đúng (không có proxy nào đứng giữa thay chứng
chỉ như trường hợp Cloudflare trước đây).
