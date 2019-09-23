# Validate Event Tekton Task
An optional task can be provided with an EventListener trigger to validate an event's payload and headers. No resource should be provided to this task.

EventListener passes event body as `EventBody` and headers encoded to json as `EventHeaders` params to the taskrun.

Additionally, if any Parameters are defined as part of `validate` under `event-listener`, they are also provided to taskrun during execution.

Sample Task provided for [`validate-github-event`](validate-github-event.yaml) has been provided.
