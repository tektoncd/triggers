# Knative event adapter

1. You can create knative service by use below yaml:

    ```yaml
    kind: Service
    metadata:
      name: event-adapter
    spec:
      template:
        metadata:
          labels:
            receive-adapter: mygitlab
        spec:
          containers:
          - image: local.registry/eventadapter-c5a0c17f3f3dedc5bdf73565d874ec4e:v0.14
            env:
             - name: SYSTEM_NAMESPACE
               valueFrom:
                 fieldRef:
                   fieldPath: metadata.namespace
    ```

1. eventadapter will load config: name: event-config-adapter, namespace: default. 
  You can change by this command args: `--configName=   --configNamespace`

1. here is example config: `$()`config like triggerBindings, means get field from request data.

```yaml
apiVersion: v1
data:
  adapters: |
    # skip after = 000000
    - filters:
      - key: $(data.after)
        value: "^0[0]*$"
    # skip not a push event
    - filters:
      - key: $(data.object_kind)
        value: push
        negative: true
    - filters:
      destListeners:
      - eventListenerName: $(params.projectname)-listener
        eventListenerNamespace: tekton-run
        params:
        - name: projectname
          value: $(data.repository.name)
        - name: commitid
          value: $(data.after)
        reqHeadFields:
          Content-Type: "application/json"
        reqBodyTemplate: |
          {
            "head_commit": { "id": "$(params.commitid)" }
          }
kind: ConfigMap
metadata:
  name: event-config-adapter
  namespace: default
```

