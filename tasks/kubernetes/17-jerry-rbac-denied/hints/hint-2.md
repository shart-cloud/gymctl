`jerry-dev` is bound to the right RoleBinding. The issue is in Role rules.

The ServiceAccount must be able to:

1. `list` pods
2. read `pods/log`

But it should still not be able to delete pods.

Check the `rules` section in the Role and adjust verbs/resources accordingly.
