# Exposing EventListeners Externally

By default, `ClusterIP` services such as the EventListener sink are accessible 
within the cluster. There are a few ways of exposing it so that external services
can talk to it:


## Using an Ingress

You can use an Ingress resource to expose the EventListener. The [`create-ingress`](./create-ingress.yaml)
Tekton task can help setup an ingress resource using self-signed certs.

**Note**: If you are using a cloud hosted Kubernetes solution such as GKE, the built-in ingress will not work
with `ClusterIP` services. Instead, you can use the Nginx Ingress based approach below.


## Using Nginx Ingress

The following instructions have been tested on GKE cluster running version `1.13.7-gke.24`.
Instructions for installing nginx Ingress on other Kubernetes services can be found [here](https://kubernetes.github.io/ingress-nginx/deploy/).

1. First, install Nginx ingress controller:
    ```yaml
      kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/mandatory.yaml
      kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/master/deploy/static/provider/cloud-generic.yaml
    ```
1. Find the service name expose by the GitHub service:
    ```yaml
     kubectl get el -o=jsonpath='{.status.configuration.generatedName} <EVENTLISTENR_NAME> '
    ```
1. Create the Ingress resource. A sample Ingress is below. Check the docs [here](https://kubernetes.github.io/ingress-nginx/user-guide/nginx-configuration/)
for a full range of configuration options.
    ```YAML
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: ingress-resource
      namespace: getting-started
      annotations:
        kubernetes.io/ingress.class: nginx
        nginx.ingress.kubernetes.io/ssl-redirect: "false"
    spec:
      rules:
      - http:
          paths:
          - path: /
            backend:
              serviceName: getting-started-listener-b8rqz # REPLACE WITH YOUR SERVICE NAME FROM STEP 2
              servicePort: 8080
    ```

1. Try it out! Get the address of the Ingress by running `kubectl get ingress ingress-resource` 
and noting the address field. You can `curl` this IP or setup a GitHub webhook to send events to it.
