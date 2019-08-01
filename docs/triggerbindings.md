# TriggerBindings
As per the name, `TriggerBindings` bind events to `TriggerTemplates`.
`TriggerBindings` build atop `TriggerTemplates` by enabling you to capture fields within an event payload and store them as parameters.
These parameters are passed into the referenced `TriggerTemplate`.
Although similar, the separation of these two resources was deliberate to encourage `TriggerTemplate` definitions to be reusable.
Further, it is as this level that the service account is connected, which specifies what permissions the resources will be created (or at least attempted) with.

```YAML
apiVersion: tekton.dev/v1alpha1
kind: TriggerBinding
metadata:
  name: simple-pipeline-binding
  namespace: default
spec:
  event:
    class: cloudevent
    type: com.github.push
  templateBindings:
    - templateRef:
        name: simple-pipeline-template
      params:
        - name: gitrevision
          value: $(event.head_commit.id)
        - name: gitrepositoryurl
          value: $(event.repository.url)
      serviceAccount: default
```

One or more `TriggerBindings` are collected together into an [`EventListener`](eventlisteners.md), which is where the pod is actually instantiated that "listens" for the respective events.
