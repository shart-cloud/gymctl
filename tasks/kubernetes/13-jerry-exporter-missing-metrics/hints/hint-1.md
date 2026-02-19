Check what the exporter currently returns:

```bash
kubectl run metrics-probe --rm -it --restart=Never --image=busybox:1.36 -- wget -qO- http://jerry-exporter:8080/metrics
```
