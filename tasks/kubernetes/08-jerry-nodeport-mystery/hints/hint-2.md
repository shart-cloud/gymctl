For Services, traffic flow is:

`client -> service.spec.ports[].port -> service.spec.ports[].targetPort -> containerPort`

One of those is mismatched.
