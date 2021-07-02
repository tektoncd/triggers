# EventListener TLS Connection

Triggers now support both `HTTP` and `HTTPS` connection by adding some configurations to eventlistener.

## Prerequisites

### Certificates with Key and Cert

##### 1. Steps to generate root key, cert
1. Create Root Key
   ```text
   openssl genrsa -des3 -out rootCA.key 4096
   ```
2. Create and self sign the Root Certificate
   ```text
   openssl req -x509 -new -nodes -key rootCA.key -sha256 -days 1024 -out rootCA.crt
   ```
##### 2. Steps to generate certificate (for each server)
1. Create the certificate key
   ```text
   openssl genrsa -out tls.key 2048
   ```
2. Create the signing (csr)

* The CSR is where you specify the details for the certificate you want to generate.
  This request will be processed by the owner of the root key to generate the certificate.

* **Important:** While creating the csr it is important to specify the `Common Name` providing the IP address or domain name for the service, otherwise the certificate cannot be verified.
   ```text
   openssl req -new -key tls.key -out tls.csr
   ```
3. Generate the certificate using the tls csr and key along with the CA Root key
   ```text
   openssl x509 -req -in tls.csr -CA rootCA.crt -CAkey rootCA.key -CAcreateserial -out tls.crt -days 500 -sha256
   ```
##### 3. Follow same steps from 2 to generate certificates for client also


### Secret which includes those certificates
Once you have the certs created following the steps above, you can create a Kubernetes secret that includes those
certificates:

 ```text
   kubectl create secret generic tls-secret-key --from-file=tls.crt --from-file=tls.key
 ```



## Try it out locally:

Once you have the certs, and secrets configured by following the steps in the prerequisite section, you can configure 
the EventListener to listen for TLS connections

1. To create the TLS connection for EventListener and all related resources, run:

   ```bash
   kubectl apply -f .
   ```

1. Test by sending the sample payload.

   ```bash
   curl -v \
   -H 'X-GitHub-Event: pull_request' \
   -H 'X-Hub-Signature: sha1=ba0cdc263b3492a74b601d240c27efe81c4720cb' \
   -H 'Content-Type: application/json' \
   -d '{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}' \
   https://<el-address> --cacert rootCA.crt --key client.key --cert client.crt
   ```

   The response status code should be `202 Accepted`
   
   [`HMAC`](https://www.freeformatter.com/hmac-generator.html) tool used to create X-Hub-Signature.
   
   In [`HMAC`](https://www.freeformatter.com/hmac-generator.html) `string` is the *body payload ex:* `{"action": "opened", "pull_request":{"head":{"sha": "28911bbb5a3e2ea034daf1f6be0a822d50e31e73"}},"repository":{"clone_url": "https://github.com/tektoncd/triggers.git"}}`
   and `secretKey` is the *given secretToken ex:* `1234567`.

1. You should see a new TaskRun that got created:

   ```bash
   kubectl get taskruns | grep tls-run-
   ```
