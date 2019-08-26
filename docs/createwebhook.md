# Create Webhook Tekton Task
The create webhook task configures necessary componetns for the github webhook sending event to the event listener.
It configures following compoments:

1. The github webhook
1. Tekton event listener
1. Ingress for the event listener
1. Selfsigned certificate for the ingress

There are options to enable / disable the configuration of the components.
This task requires the following permissions to execute.  The clusterrole with these permissions must be bound to the service account used to run this task (taskrun).

```
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - get
  - list
  - create
  - update
  - delete
- apiGroups:
  - tekton.dev
  resources:
  - eventlisteners
  verbs:
  - get
  - list
  - create
  - update
  - delete
- apiGroups:
  - extensions
  resources:
  - ingresses
  verbs:
  - create
  - get
  - list
  - delete
  - update
```

## Task params

These are the task parms to manage the task execution

- name: `CreateCertificate`
  This param enables / disables the creation of the selfsigned certificate for the ingress
  - value: "true" / "false"
  - default: "true"
- name: `CreateIngress`
  This param enables / disables the creation of the ingress for the event listener
  - value: "true" / "false"
  - default: "true"
- name: `CreateWebhook`
  This param enables / disables the creation of the webhook configuration in the github repo
  - value: "true" / "false"
  - default: "true"
- name: `CreateEventListener`
  This param enables / disables the creation of the event listener
  - value: "true" / "false"
  - default: "true"
- name: `CertificateKeyPassphrase`
  This param is the phrase that protects private key.  This must be provided when the selfsigned certificate is created. 
  - value: any string  
- name: `CertificateSecretName`
  This param is the secret name for ingress certificate.  The secret with this name should not exist if the selfsigned certificate creation is enabled.  
  - value: valid kubernates identifier string
- name: `ExternalUrl`
  This param is the external access URl for the event listener.  Examble: eventlistener1.xx.yy.zz.aa.nip.io 
  - value: valid ip address
- name: `GithubOwner`
  This param is the github owner name (github.com/**onwer**/repo) 
  - value: string
- name: `GithubRepo`
  This param is the github repo name (github.com/onwer/**repo**)
  - value: string
- name: `GithubSecretName`
  This param is the secret name for github access token. The key **userName** must have the github user name and **accessToken** must have the github access token  
  - value: kubernetes secret name string
- name: `GithubUrl`
  This param is the github side address.  The defult value **github.com** works for the public git hub.  For the github enterprize, this param have to have the site address.  Example: **github.yourcompany.com**   
  - value: github site address string
  - default: "github.com"
- name: `EventListenerName`
  This param has the event listener name 
  - value: valid kubernates identifier string
- name: `TriggerBinding`
  This param is the trigger binding set in the event listener 
  - value: triggerbinding CR instance name
- name: `TriggerTemplate`
  This param has the trigger template set in the event listener 
  - value: triggertemplate CR instance name
- name: `TriggerServiceAccount`
  This param is the service account name set in the event listener 
  - value: service account name

### Create Certificate

The github sends events to the event listner ingress.  It is configured to use the TLS to make the transport secure.  The ingress must have the certifiate to establish the TLS connection.  Ideally the certificate signed by the CA is used.  If the CA singed certificate is not available, the selfsigned certificate can be used.  The create webhook task creates the selfsigned certificate and sets it in the kubeernetes secret to make it available for the use of the ingress.

The following task params must be set for the creation of the certificate.

- `CreateCertificate`
  value: "true"
- `CertificateKeyPassphrase`
  value: any passphrase string  
- `CertificateSecretName`
  value: valid kubernates identifier string. (unused name) 
- `ExternalUrl`
  value: IP address for the identity for the site (event listener)

### Create Ingress

The ingress exposes the event listener to the outside of the cluster.

The following task params must be set for the creation of the ingress.

- `CreateIngress`
  value: "true"
- `ExternalUrl`
  value: External IP address for the event listener. Example: eventlistener1.xx.yy.zz.aa.nip.io
- `EventListenerName`
  value: The name of the event listener to be exposed
- `CertificateSecretName`
  value: The secret name of the certifiate

### Create Webhook

The webhook in the github repo sends the event to the event listener.

The following task params must be set for the configuration of the webhook.

- `CreateWebhook`
  value: "true"
- `ExternalUrl`
  value: The event listener ingress address
- `GithubOwner`
  value: github owner string
- `GithubRepo`
  value: github repo string
- `GithubSecretName`
  value: kubernetes secret name string that has the user name and the access token for the github
- `GithubUrl`
  value: only necessary for the github enterprize. github site address string

### Create Event Listener

The event listner receives the webhook event and invokes the pipelinerun based on the trigger binding and trigger template.  This task creates the eventlistener with only one pair of the triggerbinding and trigger template.

The following task params must be set for the creation of the event listener.

- `CreateEventListener`
  value: "true"
- `EventListenerName`
  This param has the event listener name
  value: event listener name
- `TriggerBinding`
  This param is the trigger binding set in the event listener
  value: triggerbinding CR instance name
- `TriggerTemplate`
  value: triggertemplate CR instnace name
- `TriggerServiceAccount`
  value: the service account name used for the event listener
