# Hint 1: Pod Admission Failures

Jerry's deployment is being rejected at the admission control level before pods are even created.

Look for:
- Deployment status and events
- Admission controller error messages
- The namespace's Pod Security labels

The error message will tell you exactly which security standards are being violated.