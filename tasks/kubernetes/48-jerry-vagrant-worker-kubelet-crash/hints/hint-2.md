If kubelet is inactive, restart and enable it:

```bash
sudo systemctl daemon-reload
sudo systemctl restart kubelet
sudo systemctl enable kubelet
```

Then verify from control plane:

```bash
kubectl get nodes
```
