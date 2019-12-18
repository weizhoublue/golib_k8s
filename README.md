# golib_k8s

#========================

test in pod:

step 0 
kubectl create clusterrolebinding default-view --clusterrole=view --serviceaccount=default:default


step 1

cd DIR

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-golang
  labels:
    app: test-golang
spec:
  selector:
    matchLabels:
      app: test-golang
  template:
    metadata:
      name: test-golang
      labels:
        app: test-golang
    spec:
      hostNetwork: true
      nodeName: `hostname`
      containers:
      - name: test-golang
        image: golang:1.13.3
        command: ["sleep" , "100d"]
        env:
        - name: GO111MODULE
          value: "on"
        - name: GOPROXY
          value: "https://goproxy.io,direct"
        resources:
          limits:
              cpu: 0.2
              memory: 500Mi
          requests:
              cpu: 0.2
              memory: 500Mi
        volumeMounts:
          - mountPath: /src
            name: mysrc
      volumes:
        # Used by calico-node.
        - name: mysrc
          hostPath:
            path: `pwd`
EOF

step 2 : enter

kubectl get pod | grep test-golang

kubectl exec -it `kubectl get pod | grep test-golang | awk '{print $1}'`  bash

go build && run executable

step 3: delete
kubectl get deployment | grep test-golang | awk '{print $1}' | xargs -I {} -n 1 kubectl delete deployment {}



//================================

export GO111MODULE=on 
export GOPROXY="https://goproxy.io,direct"













