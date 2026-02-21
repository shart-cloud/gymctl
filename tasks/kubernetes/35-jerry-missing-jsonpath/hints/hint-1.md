# Hint 1: Run the Script and See What Each Variable Contains

The broken script is at `/tmp/jerry-jsonpath-broken.sh`. Source it and print each
variable to see which queries return empty output:

```bash
source /tmp/jerry-jsonpath-broken.sh
echo "Q1: $QUERY1"
echo "Q2: $QUERY2"
echo "Q3: $QUERY3"
echo "Q4: $QUERY4"
```

Compare against what `kubectl get pods -l app=jerry-api` shows. Which queries
produce nothing, and which produce something unexpected?
