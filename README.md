# aws-secret-operator [![Build Status](https://travis-ci.com/)


This project is a Kubernetes operator that automatically creates and updates Kubernetes secrets according to what is stored in AWS Secrets Manager (SM).
A custom resource, named `AWSSecret`, maps an AWS SM entry to a K8S Secret resource.



# Usage

Let's assume we need to deploy a k8s application which needs to safely access a pair of credentials in order to call a remote REST server.
We could store these piece of information as, for instance, a JSON object identified by a SM's secret named `test/ac2019`. 

```json
{
  "userName": "my_username",
  "password": "123456"
}
```
 
We then use AWS CLI to create a new SM entry and store the secret credentials
```
$ aws secretsmanager get-secret-value --secret-id test/ac2019
$ aws secretsmanager create-secret --name test/ac2019
{
    "ARN": "arn:aws:secretsmanager:REGION:ACCOUNT:secret:test/ac2019-jeOSJZ",
    "Name": "test/ac2019"
}
```

```
$ aws secretsmanager put-secret-value\
    --secret-id test/ac2019 \
    --secret-string '{ "userName": "my_username", "password": "123456" }'
{
    "ARN": "arn:aws:secretsmanager:us-east-1:195056086334:secret:test/ac2019-jeOSJZ",
    "Name": "test/ac2019",
    "VersionId": "2b3ffed3-f4b7-4f5d-95fc-0294669f7f71",
    "VersionStages": [
        "AWSCURRENT"
    ]
}
```

Let's double check a new secret version has been properly created
```
$ aws secretsmanager describe-secret --secret-id prod/mysecret
{
    "ARN": "arn:aws:secretsmanager:us-east-1:195056086334:secret:test/ac2019-jeOSJZ",
    "Name": "test/ac2019",
    "LastChangedDate": 1556031994.027,
    "LastAccessedDate": 1555977600.0,
    "VersionIdsToStages": {
        "09b4e232-f5ad-4aa3-927c-db66815b8ebf": [
            "AWSPREVIOUS"
        ],
        "2b3ffed3-f4b7-4f5d-95fc-0294669f7f71": [
            "AWSCURRENT"
        ]
    }
}
```

Once the secret has been safely created in AWS we could destroy any local information about it.


To map the AWS SM secret into a k8s Secret resource we write a custom resource manifest (CR) file which references 
the SM secret's id and the SM's secret version obtained at the previous steps `2b3ffed3-f4b7-4f5d-95fc-0294669f7f71`.

Once the CR is deployed the operator creates a k8s Secret resource named `mysecret`.

```yaml
apiVersion: acamillo.github.com/v1alpha1
kind: AWSSecret
metadata:
  name: mysecret
spec:
    secretsManagerRef:
      secretId: test/ac2019
      versionId: 2b3ffed3-f4b7-4f5d-95fc-0294669f7f71
```
> Note that `aws-secret-operator` intentionally disallow omitting `versionId` as it makes impossible to trigger resource updates  in response to AWS secrets changes.

If the AWS secret id and version are correct, the operator creates a Kubernetes `secret` named `mysecret` that looks like:

```bash
$ kubectl get secrets mysecret -o yaml
```
```yaml
apiVersion: v1
data:
  password: MTIzNDU2
  userName: bXlfdXNlcm5hbWU=
kind: Secret
metadata:
  creationTimestamp: "2019-04-19T21:45:53Z"
  labels:
    app: aws-secret-operator
  name: mysecret
  namespace: default
  ownerReferences:
  - apiVersion: acamillo.github.com/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: AWSSecret
    name: mysecret
    uid: 80dc80d6-62ec-11e9-9798-8e0e1215e2a8
  resourceVersion: "47157"
  selfLink: /api/v1/namespaces/default/secrets/mysecret
  uid: 8163f034-62ec-11e9-9798-8e0e1215e2a8
type: Opaque
```
> Note the data is encrypted as expected

Now, the application can either mount the generated secret as a volume, or set an environment variable from the secret. In our example application we have a `deployment.yaml` as follow

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: aws-secret-consumer
  namespace: default
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: aws-secret-consumer
    spec:
      containers:
      - name: aws-secret-consumer
        image: gcr.io/google_containers/defaultbackend:1.0
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: service-key
          mountPath: /root/aws-secret
      volumes:
      - name: service-key
        secret:
          secretName: mysecret
```

The secret consumer application will see the secret as files. To inspect the value open a terminal into the application as:
```bash
$ kubectl exec -it aws-secret-consumer-76f4ccdf8c-jzkrc sh

/ # ls -l /root/aws-secret/
total 0
lrwxrwxrwx    1 root     root            15 Apr 19 21:56 password -> ..data/password
lrwxrwxrwx    1 root     root            15 Apr 19 21:56 userName -> ..data/userName

/ # cat /root/aws-secret/password
123456/ #

/ # cat /root/aws-secret/userName
my_username/ #

```

# Building The Operator 

## Prerequisites

You need the following software: 

* This project is being developed in [Go](https://golang.org/doc/install) 
* [dep](https://github.com/golang/dep) is used for dependency management.
* This project depends on the [operator-sdk v0.0.6](https://github.com/operator-framework/operator-sdk/tree/v0.0.6) - you will need the [sdk cli](https://github.com/operator-framework/operator-sdk#quick-start) for building.

**Getting The Source**:

    $ mkdir -d $GOPATH/src/github.com/acamillo
    $ cd $GOPATH/src/github.com/acamillo
    $ git clone https://github.com/acamillo/aws-secret-operator.git


**Fetching Dependencies**:

    $ cd $GOPATH/src/github.com/acamillo/aws-secret-operator
    $ dep ensure

**Build**:

    $ make docker-build tag=SOME-VERSION


**Build and Publish on Docker Registry**:

    $ make docker-push tag=SOME-VERSION

**To regenerate just the custom resources types ([pkg/apis/acamillo/v1alpha1/awssecret_types.go](`pkg/apis/acamillo§/v1alpha1/awssecret_types.go`)):**   

    $ operator-sdk generate k8s

## Deployment

Modify the files in [deploy/](deploy/), setting appropriate variables for your environment. 

Then, to deploy the operator into a kubernetes cluster run:

```å
# Setup Service Account
$ kubectl create -f deploy/service_account.yaml

# Setup RBAC
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml

# Setup the CRD
$ kubectl create -f deploy/crds/acamillo_v1alpha1_awssecret_crd.yaml

# Deploy the aws-secret-operator
$ kubectl create -f deploy/operator.yaml

# Create an AWSSecret CR
# The controller will watch for AWSSecret objects and create a new Secret for each CR
$ kubectl create -f deploy/crds/acamillo_v1alpha1_awssecret_cr.yaml


# Test the new Resource Type
$ kubectl describe secrets mysecret
Name:         mysecret
Namespace:    default
Labels:       app=aws-secret-operator
Annotations:  <none>

Type:  Opaque

Data
====
password:  6 bytes
userName:  11 bytes

```

## Cleanup
```bash
# Cleanup
kubectl delete -f deploy/crds/acamillo_v1alpha1_awssecret_cr.yaml
kubectl delete -f deploy/operator.yaml
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/role_binding.yaml
kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/crds/acamillo_v1alpha1_awssecret_crd.yaml
```


# Run locally outside the cluster
This method is preferred during development cycle to deploy and test faster.

Set the name of the operator in an environment variable and AWS environment setup:

```bash
export OPERATOR_NAME=aws-secret-operator
export AWS_SDK_LOAD_CONFIG=1
export AWS_DEFAULT_PROFILE=default

```
Run the operator locally with the default kubernetes config file present at $KUBECONFIG:

```
$ operator-sdk up local --namespace=default
INFO[0000] Running the operator locally.
INFO[0000] Using namespace default.
{"level":"info","ts":1555680165.8881862,"logger":"cmd","msg":"Go Version: go1.12.2"}
{"level":"info","ts":1555680165.888619,"logger":"cmd","msg":"Go OS/Arch: darwin/amd64"}
{"level":"info","ts":1555680165.8887522,"logger":"cmd","msg":"Version of operator-sdk: v0.6.0"}
{"level":"info","ts":1555680165.89208,"logger":"leader","msg":"Trying to become the leader."}
{"level":"info","ts":1555680165.8921359,"logger":"leader","msg":"Skipping leader election; not running in a cluster."}
{"level":"info","ts":1555680165.934182,"logger":"cmd","msg":"Registering Components."}
{"level":"info","ts":1555680165.934331,"logger":"kubebuilder.controller","msg":"Starting EventSource","controller":"awssecret-controller","source":"kind source: /, Kind="}
{"level":"info","ts":1555680165.962528,"logger":"metrics","msg":"Skipping metrics Service creation; not running in a cluster."}
{"level":"info","ts":1555680165.962552,"logger":"cmd","msg":"Starting the Cmd."}
{"level":"info","ts":1555680166.063024,"logger":"kubebuilder.controller","msg":"Starting Controller","controller":"awssecret-controller"}
{"level":"info","ts":1555680166.166064,"logger":"kubebuilder.controller","msg":"Starting workers","controller":"awssecret-controller","worker count":1}
```

# Run locally inside Minikube

See time synch issue [here](https://github.com/kubernetes/minikube/issues/1378) 

The steps to follow for running the operator as a dweployment in a local Minikube cluster are:
1. Start Minikube
2. log in AWS as usual
3. mount the AWS configuration and credentials file as a minikube file system. 
4. synchronize the minikube VM date and time with the host 
5. deploy the operator with a [minikube/](minikube/) customized deployment manifest.

```bash
$ minikube start --vm-driver hyperkit --memory 8192 --cpus 4 --extra-config=apiserver.authorization-mode=RBAC
$ minikube mount ~/.aws/:/aws
$ minikube ssh -- docker run -i --rm --privileged --pid=host debian nsenter -t 1 -m -u -n -i date -u $(date -u +%m%d%H%M%Y)
$ kubectl apply -f deploy/minikube/operator.yaml
```
