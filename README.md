## Installing Components on Kubernetes

**Install cAdvisor:**
```bash
kubectl apply -f Metric_Collector/cAdvisor-deploy.yaml
```

**Install Node Exporter:**
```bash
kubectl apply -f Metric_Collector/NodeExporter-deploy.yaml
```

**Install Kube State Metrics:**

**Get Repository Info**

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
```

**Install Chart**

```bash
helm install kube-state-metrics prometheus-community/kube-state-metrics \
--set service.type=NodePort \
--set service.nodePort=30808 \
-n monitoring <namespace>
```
## Init Blockchain Node 
```bash
cd StreamPay/streampay-socone
make reset
```

## Run Docker Stack

adjust the parameters accordingly,
e.g. RUNTIME_RPC_ADDRESS ( dexplorer ), provider-address ( payment-engine) 

```bash
docker compose up -d
```


