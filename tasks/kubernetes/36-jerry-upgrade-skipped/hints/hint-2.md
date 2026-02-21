# Hint 2: The Correct Upgrade Sequence (Control Plane)

A kubeadm upgrade from v1.29 to v1.31 requires two separate passes:
v1.29 → v1.30, then v1.30 → v1.31.

For each pass, the control-plane steps must be in this order:

1. `apt-mark unhold kubeadm`
2. `apt-get install kubeadm=<target-version>`
3. `kubeadm upgrade plan`
4. `kubeadm upgrade apply <target-version>`
5. `apt-mark unhold kubelet kubectl` then install new versions

Count how many of these five things are missing or wrong in the current script.
