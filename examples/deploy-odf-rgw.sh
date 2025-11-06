#!/bin/bash
# Quick deployment script for OpenShift ODF/RGW
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}S3 Workload - OpenShift ODF/RGW Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo

# Check if oc is installed
if ! command -v oc &> /dev/null; then
    echo -e "${RED}Error: oc CLI not found. Please install OpenShift CLI.${NC}"
    exit 1
fi

# Check if logged in
if ! oc whoami &> /dev/null; then
    echo -e "${RED}Error: Not logged into OpenShift. Please run 'oc login' first.${NC}"
    exit 1
fi

# Configuration
NAMESPACE=${NAMESPACE:-s3-workload}
RGW_ENDPOINT=${RGW_ENDPOINT:-https://s3.openshift-storage.svc.cluster.local}
RGW_ACCESS_KEY=${RGW_ACCESS_KEY:-}
RGW_SECRET_KEY=${RGW_SECRET_KEY:-}
BUCKET=${BUCKET:-odf-bench-bucket}

echo -e "${YELLOW}Configuration:${NC}"
echo "  Namespace: $NAMESPACE"
echo "  RGW Endpoint: $RGW_ENDPOINT"
echo "  Bucket: $BUCKET"
echo

# Check if credentials are provided
if [ -z "$RGW_ACCESS_KEY" ] || [ -z "$RGW_SECRET_KEY" ]; then
    echo -e "${YELLOW}RGW credentials not provided via environment variables.${NC}"
    echo -e "${YELLOW}Trying to retrieve from Noobaa admin secret...${NC}"
    
    if oc get secret noobaa-admin -n openshift-storage &> /dev/null; then
        RGW_ACCESS_KEY=$(oc get secret noobaa-admin -n openshift-storage -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
        RGW_SECRET_KEY=$(oc get secret noobaa-admin -n openshift-storage -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)
        echo -e "${GREEN}✓ Retrieved credentials from noobaa-admin secret${NC}"
    else
        echo -e "${RED}Error: Noobaa admin secret not found and credentials not provided.${NC}"
        echo
        echo "Please provide credentials via environment variables:"
        echo "  export RGW_ACCESS_KEY=your_access_key"
        echo "  export RGW_SECRET_KEY=your_secret_key"
        echo
        echo "Or create RGW user manually. See docs/ODF_RGW_SETUP.md for details."
        exit 1
    fi
fi

# Create namespace
echo -e "${BLUE}Step 1: Creating namespace...${NC}"
if oc get namespace $NAMESPACE &> /dev/null; then
    echo -e "${YELLOW}⚠ Namespace $NAMESPACE already exists${NC}"
else
    oc new-project $NAMESPACE
    echo -e "${GREEN}✓ Namespace created${NC}"
fi
echo

# Create secret
echo -e "${BLUE}Step 2: Creating secret with RGW credentials...${NC}"
if oc get secret s3-creds -n $NAMESPACE &> /dev/null; then
    echo -e "${YELLOW}⚠ Secret already exists, deleting...${NC}"
    oc delete secret s3-creds -n $NAMESPACE
fi

oc create secret generic s3-creds \
    --from-literal=accessKey="$RGW_ACCESS_KEY" \
    --from-literal=secretKey="$RGW_SECRET_KEY" \
    -n $NAMESPACE
echo -e "${GREEN}✓ Secret created${NC}"
echo

# Create service account
echo -e "${BLUE}Step 3: Creating service account...${NC}"
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: s3-workload
  namespace: $NAMESPACE
  labels:
    app: s3-workload
EOF
echo -e "${GREEN}✓ Service account created${NC}"
echo

# Create configmap
echo -e "${BLUE}Step 4: Creating configmap...${NC}"
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: s3-workload-config
  namespace: $NAMESPACE
  labels:
    app: s3-workload
    storage: odf-rgw
data:
  endpoint: "$RGW_ENDPOINT"
  region: "us-east-1"
  bucket: "$BUCKET"
  path-style: "true"
  skip-tls-verify: "false"
  concurrency: "64"
  duration: "30m"
  keys: "100000"
  prefix: "bench/"
  mix: "put=40,get=40,delete=10,copy=5,list=5"
  size: "dist:lognormal:mean=1MiB,std=0.6"
  pattern: "random:42"
  verify-rate: "0.1"
  log-level: "info"
  metrics-port: "9090"
EOF
echo -e "${GREEN}✓ ConfigMap created${NC}"
echo

# Get the script directory to find deployment files
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
DEPLOY_DIR="$SCRIPT_DIR/../deploy/kubernetes"

# Deploy workload
echo -e "${BLUE}Step 5: Deploying workload...${NC}"
if [ -f "$DEPLOY_DIR/deployment-odf-rgw.yaml" ]; then
    oc apply -f "$DEPLOY_DIR/deployment-odf-rgw.yaml"
else
    echo -e "${YELLOW}⚠ Using inline deployment manifest${NC}"
    cat <<EOF | oc apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: s3-workload
  namespace: $NAMESPACE
  labels:
    app: s3-workload
    storage: odf-rgw
spec:
  replicas: 1
  selector:
    matchLabels:
      app: s3-workload
  template:
    metadata:
      labels:
        app: s3-workload
        storage: odf-rgw
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: s3-workload
      securityContext:
        runAsNonRoot: true
        runAsUser: 10001
        fsGroup: 10001
      containers:
      - name: s3-workload
        image: ghcr.io/paragkamble/s3-workload:latest
        imagePullPolicy: Always
        args:
          - --endpoint=\$(S3_ENDPOINT)
          - --region=\$(AWS_REGION)
          - --bucket=\$(S3_BUCKET)
          - --path-style=\$(PATH_STYLE)
          - --skip-tls-verify=\$(SKIP_TLS_VERIFY)
          - --create-bucket
          - --concurrency=\$(CONCURRENCY)
          - --mix=\$(MIX)
          - --size=\$(SIZE)
          - --keys=\$(KEYS)
          - --prefix=\$(PREFIX)
          - --pattern=\$(PATTERN)
          - --verify-rate=\$(VERIFY_RATE)
          - --duration=\$(DURATION)
          - --metrics-port=\$(METRICS_PORT)
          - --log-level=\$(LOG_LEVEL)
          - --http-bind=0.0.0.0
        env:
        - name: S3_ENDPOINT
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: endpoint
        - name: AWS_REGION
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: region
        - name: S3_BUCKET
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: bucket
        - name: PATH_STYLE
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: path-style
        - name: SKIP_TLS_VERIFY
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: skip-tls-verify
        - name: CONCURRENCY
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: concurrency
        - name: DURATION
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: duration
        - name: KEYS
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: keys
        - name: PREFIX
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: prefix
        - name: MIX
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: mix
        - name: SIZE
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: size
        - name: PATTERN
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: pattern
        - name: VERIFY_RATE
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: verify-rate
        - name: LOG_LEVEL
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: log-level
        - name: METRICS_PORT
          valueFrom:
            configMapKeyRef:
              name: s3-workload-config
              key: metrics-port
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: s3-creds
              key: accessKey
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: s3-creds
              key: secretKey
        ports:
        - name: metrics
          containerPort: 9090
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /healthz
            port: metrics
          initialDelaySeconds: 10
          periodSeconds: 30
          timeoutSeconds: 5
        readinessProbe:
          httpGet:
            path: /readyz
            port: metrics
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "2000m"
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 10001
          capabilities:
            drop:
              - ALL
EOF
fi
echo -e "${GREEN}✓ Deployment created${NC}"
echo

# Create service
echo -e "${BLUE}Step 6: Creating service...${NC}"
cat <<EOF | oc apply -f -
apiVersion: v1
kind: Service
metadata:
  name: s3-workload-metrics
  namespace: $NAMESPACE
  labels:
    app: s3-workload
spec:
  type: ClusterIP
  ports:
  - port: 9090
    targetPort: metrics
    protocol: TCP
    name: http
  selector:
    app: s3-workload
EOF
echo -e "${GREEN}✓ Service created${NC}"
echo

# Wait for deployment
echo -e "${BLUE}Step 7: Waiting for deployment to be ready...${NC}"
oc rollout status deployment/s3-workload -n $NAMESPACE --timeout=120s
echo -e "${GREEN}✓ Deployment is ready${NC}"
echo

# Show status
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Deployment Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo
echo -e "${YELLOW}Useful commands:${NC}"
echo
echo "View logs:"
echo "  oc logs -n $NAMESPACE -l app=s3-workload -f"
echo
echo "View pods:"
echo "  oc get pods -n $NAMESPACE"
echo
echo "View metrics:"
echo "  oc port-forward -n $NAMESPACE svc/s3-workload-metrics 9090:9090"
echo "  Then open http://localhost:9090/metrics"
echo
echo "Delete deployment:"
echo "  oc delete project $NAMESPACE"
echo
echo "Cleanup test data:"
echo "  oc run cleanup --rm -it --restart=Never --image=ghcr.io/paragkamble/s3-workload:latest \\"
echo "    --env AWS_ACCESS_KEY_ID=$RGW_ACCESS_KEY \\"
echo "    --env AWS_SECRET_ACCESS_KEY=$RGW_SECRET_KEY \\"
echo "    -n $NAMESPACE -- \\"
echo "    --endpoint $RGW_ENDPOINT --bucket $BUCKET --prefix bench/ --path-style --cleanup"
echo
echo -e "${GREEN}For more information, see docs/ODF_RGW_SETUP.md${NC}"

