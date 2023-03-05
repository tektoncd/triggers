<p>Packages:</p>
<ul>
<li>
<a href="#triggers.tekton.dev%2fv1alpha1">triggers.tekton.dev/v1alpha1</a>
</li>
<li>
<a href="#triggers.tekton.dev%2fv1beta1">triggers.tekton.dev/v1beta1</a>
</li>
</ul>
<h2 id="triggers.tekton.dev/v1alpha1">triggers.tekton.dev/v1alpha1</h2>
<div>
<p>Package v1alpha1 contains API Schema definitions for the triggers v1alpha1 API group</p>
</div>
Resource Types:
<ul><li>
<a href="#triggers.tekton.dev/v1alpha1.ClusterTriggerBinding">ClusterTriggerBinding</a>
</li><li>
<a href="#triggers.tekton.dev/v1alpha1.EventListener">EventListener</a>
</li><li>
<a href="#triggers.tekton.dev/v1alpha1.Trigger">Trigger</a>
</li><li>
<a href="#triggers.tekton.dev/v1alpha1.TriggerBinding">TriggerBinding</a>
</li></ul>
<h3 id="triggers.tekton.dev/v1alpha1.ClusterTriggerBinding">ClusterTriggerBinding
</h3>
<div>
<p>ClusterTriggerBinding is a TriggerBinding with a cluster scope.
ClusterTriggerBindings are used to represent TriggerBindings that
should be publicly addressable from any namespace in the cluster.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>ClusterTriggerBinding</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerBindingSpec">
TriggerBindingSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the ClusterTriggerBinding from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Param">
[]Param
</a>
</em>
</td>
<td>
<p>Params defines the parameter mapping from the given input event.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerBindingStatus">
TriggerBindingStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.EventListener">EventListener
</h3>
<div>
<p>EventListener exposes a service to accept HTTP event payloads.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>EventListener</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.EventListenerSpec">
EventListenerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the EventListener from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggers</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.EventListenerTrigger">
[]EventListenerTrigger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.NamespaceSelector">
NamespaceSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resources</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Resources">
Resources
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.EventListenerStatus">
EventListenerStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.Trigger">Trigger
</h3>
<div>
<p>Trigger defines a mapping of an input event to parameters. This is used
to extract information from events to be passed to TriggerTemplates within a
Trigger.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Trigger</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpec">
TriggerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the Trigger</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>bindings</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpecBinding">
[]TriggerSpecBinding
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpecTemplate">
TriggerSpecTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountName optionally associates credentials with each trigger;
Unlike EventListeners, this should be scoped to the same namespace
as the Trigger itself</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerBinding">TriggerBinding
</h3>
<div>
<p>TriggerBinding defines a mapping of an input event to parameters. This is used
to extract information from events to be passed to TriggerTemplates within a
Trigger.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>TriggerBinding</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerBindingSpec">
TriggerBindingSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the TriggerBinding</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Param">
[]Param
</a>
</em>
</td>
<td>
<p>Params defines the parameter mapping from the given input event.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerBindingStatus">
TriggerBindingStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.BitbucketInterceptor">BitbucketInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>BitbucketInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>eventTypes</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.CELInterceptor">CELInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>CELInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>filter</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>overlays</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.CELOverlay">
[]CELOverlay
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.CELOverlay">CELOverlay
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.CELInterceptor">CELInterceptor</a>)
</p>
<div>
<p>CELOverlay provides a way to modify the request body using DeprecatedCEL expressions</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>expression</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.ClientConfig">ClientConfig
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.ClusterInterceptorSpec">ClusterInterceptorSpec</a>, <a href="#triggers.tekton.dev/v1alpha1.InterceptorSpec">InterceptorSpec</a>)
</p>
<div>
<p>ClientConfig describes how a client can communicate with the Interceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>caBundle</code><br/>
<em>
[]byte
</em>
</td>
<td>
<p>CaBundle is a PEM encoded CA bundle which will be used to validate the clusterinterceptor server certificate</p>
</td>
</tr>
<tr>
<td>
<code>url</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis#URL">
knative.dev/pkg/apis.URL
</a>
</em>
</td>
<td>
<p>URL is a fully formed URL pointing to the interceptor
Mutually exclusive with Service</p>
</td>
</tr>
<tr>
<td>
<code>service</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ServiceReference">
ServiceReference
</a>
</em>
</td>
<td>
<p>Service is a reference to a Service object where the interceptor is running
Mutually exclusive with URL</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.ClusterInterceptor">ClusterInterceptor
</h3>
<div>
<p>ClusterInterceptor describes a pluggable interceptor including configuration
such as the fields it accepts and its deployment address. The type is based on
the Validating/MutatingWebhookConfiguration types for configuring AdmissionWebhooks</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ClusterInterceptorSpec">
ClusterInterceptorSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>clientConfig</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ClientConfig">
ClientConfig
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ClusterInterceptorStatus">
ClusterInterceptorStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.ClusterInterceptorSpec">ClusterInterceptorSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.ClusterInterceptor">ClusterInterceptor</a>)
</p>
<div>
<p>ClusterInterceptorSpec describes the Spec for an ClusterInterceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientConfig</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ClientConfig">
ClientConfig
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.ClusterInterceptorStatus">ClusterInterceptorStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.ClusterInterceptor">ClusterInterceptor</a>)
</p>
<div>
<p>ClusterInterceptorStatus holds the status of the ClusterInterceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Status</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#Status">
knative.dev/pkg/apis/duck/v1.Status
</a>
</em>
</td>
<td>
<p>
(Members of <code>Status</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>AddressStatus</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#AddressStatus">
knative.dev/pkg/apis/duck/v1.AddressStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>AddressStatus</code> are embedded into this type.)
</p>
<p>ClusterInterceptor is Addressable and exposes the URL where the Interceptor is running</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.CustomResource">CustomResource
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.Resources">Resources</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>RawExtension</code><br/>
<em>
k8s.io/apimachinery/pkg/runtime.RawExtension
</em>
</td>
<td>
<p>
(Members of <code>RawExtension</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.EventListenerConfig">EventListenerConfig
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerStatus">EventListenerStatus</a>)
</p>
<div>
<p>EventListenerConfig stores configuration for resources generated by the
EventListener</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>generatedName</code><br/>
<em>
string
</em>
</td>
<td>
<p>GeneratedResourceName is the name given to all resources reconciled by
the EventListener</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.EventListenerSpec">EventListenerSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListener">EventListener</a>)
</p>
<div>
<p>EventListenerSpec defines the desired state of the EventListener, represented
by a list of Triggers.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggers</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.EventListenerTrigger">
[]EventListenerTrigger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.NamespaceSelector">
NamespaceSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resources</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Resources">
Resources
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.EventListenerStatus">EventListenerStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListener">EventListener</a>)
</p>
<div>
<p>EventListenerStatus holds the status of the EventListener</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Status</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#Status">
knative.dev/pkg/apis/duck/v1.Status
</a>
</em>
</td>
<td>
<p>
(Members of <code>Status</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>AddressStatus</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1alpha1#AddressStatus">
knative.dev/pkg/apis/duck/v1alpha1.AddressStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>AddressStatus</code> are embedded into this type.)
</p>
<p>EventListener is Addressable. It currently exposes the service DNS
address of the the EventListener sink</p>
</td>
</tr>
<tr>
<td>
<code>configuration</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.EventListenerConfig">
EventListenerConfig
</a>
</em>
</td>
<td>
<p>Configuration stores configuration for the EventListener service</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.EventListenerTrigger">EventListenerTrigger
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerSpec">EventListenerSpec</a>)
</p>
<div>
<p>EventListenerTrigger represents a connection between TriggerBinding, Params,
and TriggerTemplate; TriggerBinding provides extracted values for
TriggerTemplate to then create resources from. TriggerRef can also be
provided instead of TriggerBinding, Interceptors and TriggerTemplate</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>bindings</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpecBinding">
[]TriggerSpecBinding
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpecTemplate">
TriggerSpecTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggerRef</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountName optionally associates credentials with each trigger;
more granular authorization for
who is allowed to utilize the associated pipeline
vs. defaulting to whatever permissions are associated
with the entire EventListener and associated sink facilitates
multi-tenant model based scenarios</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.GitHubInterceptor">GitHubInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>GitHubInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>eventTypes</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.GitLabInterceptor">GitLabInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>GitLabInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>eventTypes</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.Interceptor">Interceptor
</h3>
<div>
<p>Interceptor describes a pluggable interceptor including configuration
such as the fields it accepts and its deployment address. The type is based on
the Validating/MutatingWebhookConfiguration types for configuring AdmissionWebhooks</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.InterceptorSpec">
InterceptorSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>clientConfig</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ClientConfig">
ClientConfig
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.InterceptorStatus">
InterceptorStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorInterface">InterceptorInterface
</h3>
<div>
</div>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorKind">InterceptorKind
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.InterceptorRef">InterceptorRef</a>)
</p>
<div>
<p>InterceptorKind defines the type of Interceptor used by the Trigger.</p>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ClusterInterceptor&#34;</p></td>
<td><p>ClusterInterceptorKind indicates that Interceptor type has a cluster scope.</p>
</td>
</tr><tr><td><p>&#34;NamespacedInterceptor&#34;</p></td>
<td><p>NamespacedInterceptorKind indicated that interceptor has a namespaced scope</p>
</td>
</tr></tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorParams">InterceptorParams
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>InterceptorParams defines a key-value pair that can be passed on an interceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorRef">InterceptorRef
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>InterceptorRef provides a Reference to a ClusterInterceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name of the referent; More info: <a href="http://kubernetes.io/docs/user-guide/identifiers#names">http://kubernetes.io/docs/user-guide/identifiers#names</a></p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.InterceptorKind">
InterceptorKind
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>InterceptorKind indicates the kind of the Interceptor, namespaced or cluster scoped.</p>
</td>
</tr>
<tr>
<td>
<code>apiVersion</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>API version of the referent</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorRequest">InterceptorRequest
</h3>
<div>
<p>Do not generate DeepCopy(). See #827</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>body</code><br/>
<em>
string
</em>
</td>
<td>
<p>Body is the incoming HTTP event body. We use a &ldquo;string&rdquo; representation of the JSON body
in order to preserve the body exactly as it was sent (including spaces etc.). This is necessary
for some interceptors e.g. GitHub for validating the body with a signature. While []byte can also
store an exact representation of the body, <code>json.Marshal</code> will compact []byte to a base64 encoded
string which means that we will lose the spaces any time we marshal this struct.</p>
</td>
</tr>
<tr>
<td>
<code>header</code><br/>
<em>
map[string][]string
</em>
</td>
<td>
<p>Header are the headers for the incoming HTTP event</p>
</td>
</tr>
<tr>
<td>
<code>form</code><br/>
<em>
map[string][]string
</em>
</td>
<td>
<p>Form is the form data for the incoming HTTP event</p>
</td>
</tr>
<tr>
<td>
<code>extensions</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>Extensions are extra values that are added by previous interceptors in a chain</p>
</td>
</tr>
<tr>
<td>
<code>interceptor_params</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>InterceptorParams are the user specified params for interceptor in the Trigger</p>
</td>
</tr>
<tr>
<td>
<code>context</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerContext">
TriggerContext
</a>
</em>
</td>
<td>
<p>Context contains additional metadata about the event being processed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorResponse">InterceptorResponse
</h3>
<div>
<p>Do not generate Deepcopy(). See #827</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>extensions</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>Extensions are additional fields that is added to the interceptor event.</p>
</td>
</tr>
<tr>
<td>
<code>continue</code><br/>
<em>
bool
</em>
</td>
<td>
<p>Continue indicates if the EventListener should continue processing the Trigger or not</p>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Status">
Status
</a>
</em>
</td>
<td>
<p>Status is an Error status containing details on any interceptor processing errors</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorSpec">InterceptorSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.Interceptor">Interceptor</a>)
</p>
<div>
<p>InterceptorSpec describes the Spec for an Interceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientConfig</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ClientConfig">
ClientConfig
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.InterceptorStatus">InterceptorStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.Interceptor">Interceptor</a>)
</p>
<div>
<p>InterceptorStatus holds the status of the Interceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Status</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#Status">
knative.dev/pkg/apis/duck/v1.Status
</a>
</em>
</td>
<td>
<p>
(Members of <code>Status</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>AddressStatus</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#AddressStatus">
knative.dev/pkg/apis/duck/v1.AddressStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>AddressStatus</code> are embedded into this type.)
</p>
<p>Interceptor is Addressable and exposes the URL where the Interceptor is running</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.KubernetesResource">KubernetesResource
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.Resources">Resources</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>replicas</code><br/>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceType</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#servicetype-v1-core">
Kubernetes core/v1.ServiceType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#WithPodSpec">
knative.dev/pkg/apis/duck/v1.WithPodSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#PodSpecable">
knative.dev/pkg/apis/duck/v1.PodSpecable
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.NamespaceSelector">NamespaceSelector
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerSpec">EventListenerSpec</a>)
</p>
<div>
<p>NamespaceSelector is a selector for selecting either all namespaces or a
list of namespaces.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>matchNames</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>List of namespace names.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.Param">Param
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerBindingSpec">TriggerBindingSpec</a>)
</p>
<div>
<p>Param defines a string value to be used for a ParamSpec with the same name.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.ParamSpec">ParamSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerTemplateSpec">TriggerTemplateSpec</a>)
</p>
<div>
<p>ParamSpec defines an arbitrary named  input whose value can be supplied by a
<code>Param</code>.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name declares the name by which a parameter is referenced.</p>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Description is a user-facing description of the parameter that may be
used to populate a UI.</p>
</td>
</tr>
<tr>
<td>
<code>default</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default is the value a parameter takes if no input value via a Param is supplied.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.Resources">Resources
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerSpec">EventListenerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>kubernetesResource</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.KubernetesResource">
KubernetesResource
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>customResource</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.CustomResource">
CustomResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.SecretRef">SecretRef
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.BitbucketInterceptor">BitbucketInterceptor</a>, <a href="#triggers.tekton.dev/v1alpha1.GitHubInterceptor">GitHubInterceptor</a>, <a href="#triggers.tekton.dev/v1alpha1.GitLabInterceptor">GitLabInterceptor</a>)
</p>
<div>
<p>SecretRef contains the information required to reference a single secret string
This is needed because the other secretRef types are not cross-namespace and do not
actually contain the &ldquo;SecretName&rdquo; field, which allows us to access a single secret value.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretKey</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>secretName</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.ServiceReference">ServiceReference
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.ClientConfig">ClientConfig</a>)
</p>
<div>
<p>ServiceReference is a reference to a Service object
with an optional path</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the service</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code><br/>
<em>
string
</em>
</td>
<td>
<p>Namespace is the namespace of the service</p>
</td>
</tr>
<tr>
<td>
<code>path</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Path is an optional URL path</p>
</td>
</tr>
<tr>
<td>
<code>port</code><br/>
<em>
int32
</em>
</td>
<td>
<p>Port is a valid port number</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.Status">Status
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.InterceptorResponse">InterceptorResponse</a>, <a href="#triggers.tekton.dev/v1alpha1.StatusError">StatusError</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>code</code><br/>
<em>
<a href="https://pkg.go.dev/google.golang.org/grpc/codes#Code">
google.golang.org/grpc/codes.Code
</a>
</em>
</td>
<td>
<p>The status code, which should be an enum value of [google.rpc.Code][google.rpc.Code].</p>
</td>
</tr>
<tr>
<td>
<code>message</code><br/>
<em>
string
</em>
</td>
<td>
<p>A developer-facing error message, which should be in English.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.StatusError">StatusError
</h3>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>s</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Status">
Status
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerBindingInterface">TriggerBindingInterface
</h3>
<div>
<p>TriggerBindingInterface is implemented by TriggerBinding and ClusterTriggerBinding</p>
</div>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerBindingKind">TriggerBindingKind
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerSpecBinding">TriggerSpecBinding</a>)
</p>
<div>
<p>Check that EventListener may be validated and defaulted.
TriggerBindingKind defines the type of TriggerBinding used by the EventListener.</p>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ClusterTriggerBinding&#34;</p></td>
<td><p>ClusterTriggerBindingKind indicates that triggerbinding type has a cluster scope.</p>
</td>
</tr><tr><td><p>&#34;TriggerBinding&#34;</p></td>
<td><p>NamespacedTriggerBindingKind indicates that triggerbinding type has a namespace scope.</p>
</td>
</tr></tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerBindingSpec">TriggerBindingSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.ClusterTriggerBinding">ClusterTriggerBinding</a>, <a href="#triggers.tekton.dev/v1alpha1.TriggerBinding">TriggerBinding</a>)
</p>
<div>
<p>TriggerBindingSpec defines the desired state of the TriggerBinding.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.Param">
[]Param
</a>
</em>
</td>
<td>
<p>Params defines the parameter mapping from the given input event.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerBindingStatus">TriggerBindingStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.ClusterTriggerBinding">ClusterTriggerBinding</a>, <a href="#triggers.tekton.dev/v1alpha1.TriggerBinding">TriggerBinding</a>)
</p>
<div>
<p>TriggerBindingStatus defines the observed state of TriggerBinding.</p>
</div>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerContext">TriggerContext
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.InterceptorRequest">InterceptorRequest</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>event_url</code><br/>
<em>
string
</em>
</td>
<td>
<p>EventURL is the URL of the incoming event</p>
</td>
</tr>
<tr>
<td>
<code>event_id</code><br/>
<em>
string
</em>
</td>
<td>
<p>EventID is a unique ID assigned by Triggers to each event</p>
</td>
</tr>
<tr>
<td>
<code>trigger_id</code><br/>
<em>
string
</em>
</td>
<td>
<p>TriggerID is of the form namespace/$ns/triggers/$name</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerTrigger">EventListenerTrigger</a>, <a href="#triggers.tekton.dev/v1alpha1.TriggerSpec">TriggerSpec</a>)
</p>
<div>
<p>TriggerInterceptor provides a hook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Optional name to identify the current interceptor configuration</p>
</td>
</tr>
<tr>
<td>
<code>ref</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.InterceptorRef">
InterceptorRef
</a>
</em>
</td>
<td>
<p>Ref refers to the Interceptor to use</p>
</td>
</tr>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.InterceptorParams">
[]InterceptorParams
</a>
</em>
</td>
<td>
<p>Params are the params to send to the interceptor</p>
</td>
</tr>
<tr>
<td>
<code>webhook</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.WebhookInterceptor">
WebhookInterceptor
</a>
</em>
</td>
<td>
<p>WebhookInterceptor refers to an old style webhook interceptor service</p>
</td>
</tr>
<tr>
<td>
<code>github</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.GitHubInterceptor">
GitHubInterceptor
</a>
</em>
</td>
<td>
<p>Deprecated old fields below</p>
</td>
</tr>
<tr>
<td>
<code>gitlab</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.GitLabInterceptor">
GitLabInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cel</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.CELInterceptor">
CELInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bitbucket</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.BitbucketInterceptor">
BitbucketInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerResourceTemplate">TriggerResourceTemplate
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerTemplateSpec">TriggerTemplateSpec</a>)
</p>
<div>
<p>TriggerResourceTemplate describes a resource to create</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>RawExtension</code><br/>
<em>
k8s.io/apimachinery/pkg/runtime.RawExtension
</em>
</td>
<td>
<p>
(Members of <code>RawExtension</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerSpec">TriggerSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.Trigger">Trigger</a>)
</p>
<div>
<p>TriggerSpec represents a connection between TriggerSpecBinding,
and TriggerSpecTemplate; TriggerSpecBinding provides extracted values for
TriggerSpecTemplate to then create resources from.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>bindings</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpecBinding">
[]TriggerSpecBinding
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerSpecTemplate">
TriggerSpecTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountName optionally associates credentials with each trigger;
Unlike EventListeners, this should be scoped to the same namespace
as the Trigger itself</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerSpecBinding">TriggerSpecBinding
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerTrigger">EventListenerTrigger</a>, <a href="#triggers.tekton.dev/v1alpha1.TriggerSpec">TriggerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the binding param
Mutually exclusive with Ref</p>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
string
</em>
</td>
<td>
<p>Value is the value of the binding param. Can contain JSONPath
Has to be pointer since &ldquo;&rdquo; is a valid value
Required if Name is also specified.</p>
</td>
</tr>
<tr>
<td>
<code>ref</code><br/>
<em>
string
</em>
</td>
<td>
<p>Ref is a reference to a TriggerBinding kind.
Mutually exclusive with Name</p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerBindingKind">
TriggerBindingKind
</a>
</em>
</td>
<td>
<p>Kind can only be provided if Ref is also provided. Defaults to TriggerBinding</p>
</td>
</tr>
<tr>
<td>
<code>apiversion</code><br/>
<em>
string
</em>
</td>
<td>
<p>APIVersion of the binding ref</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerSpecTemplate">TriggerSpecTemplate
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.EventListenerTrigger">EventListenerTrigger</a>, <a href="#triggers.tekton.dev/v1alpha1.TriggerSpec">TriggerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ref</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>apiversion</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerTemplateSpec">
TriggerTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerTemplate">TriggerTemplate
</h3>
<div>
<p>TriggerTemplate takes parameters and uses them to create CRDs</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerTemplateSpec">
TriggerTemplateSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the TriggerTemplate from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ParamSpec">
[]ParamSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resourcetemplates</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerResourceTemplate">
[]TriggerResourceTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerTemplateStatus">
TriggerTemplateStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerTemplateSpec">TriggerTemplateSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerSpecTemplate">TriggerSpecTemplate</a>, <a href="#triggers.tekton.dev/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>
<div>
<p>TriggerTemplateSpec holds the desired state of TriggerTemplate</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.ParamSpec">
[]ParamSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resourcetemplates</code><br/>
<em>
<a href="#triggers.tekton.dev/v1alpha1.TriggerResourceTemplate">
[]TriggerResourceTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1alpha1.TriggerTemplateStatus">TriggerTemplateStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerTemplate">TriggerTemplate</a>)
</p>
<div>
<p>TriggerTemplateStatus describes the desired state of TriggerTemplate</p>
</div>
<h3 id="triggers.tekton.dev/v1alpha1.WebhookInterceptor">WebhookInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1alpha1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>WebhookInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>objectRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObjectRef is a reference to an object that will resolve to a cluster DNS
name to use as the EventInterceptor. Either objectRef or url can be specified</p>
</td>
</tr>
<tr>
<td>
<code>url</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis#URL">
knative.dev/pkg/apis.URL
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>header</code><br/>
<em>
<a href="https://pkg.go.dev/github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1#Param">
[]github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1.Param
</a>
</em>
</td>
<td>
<p>Header is a group of key-value pairs that can be appended to the
interceptor request headers. This allows the interceptor to make
decisions specific to an EventListenerTrigger.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<h2 id="triggers.tekton.dev/v1beta1">triggers.tekton.dev/v1beta1</h2>
<div>
<p>package v1beta1 contains API Schema definitions for the triggers v1beta1 API group</p>
</div>
Resource Types:
<ul><li>
<a href="#triggers.tekton.dev/v1beta1.ClusterTriggerBinding">ClusterTriggerBinding</a>
</li><li>
<a href="#triggers.tekton.dev/v1beta1.EventListener">EventListener</a>
</li><li>
<a href="#triggers.tekton.dev/v1beta1.Trigger">Trigger</a>
</li><li>
<a href="#triggers.tekton.dev/v1beta1.TriggerBinding">TriggerBinding</a>
</li></ul>
<h3 id="triggers.tekton.dev/v1beta1.ClusterTriggerBinding">ClusterTriggerBinding
</h3>
<div>
<p>ClusterTriggerBinding is a TriggerBinding with a cluster scope.
ClusterTriggerBindings are used to represent TriggerBindings that
should be publicly addressable from any namespace in the cluster.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>ClusterTriggerBinding</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerBindingSpec">
TriggerBindingSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the ClusterTriggerBinding from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Param">
[]Param
</a>
</em>
</td>
<td>
<p>Params defines the parameter mapping from the given input event.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerBindingStatus">
TriggerBindingStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListener">EventListener
</h3>
<div>
<p>EventListener exposes a service to accept HTTP event payloads.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>EventListener</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerSpec">
EventListenerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the EventListener from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggers</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerTrigger">
[]EventListenerTrigger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggerGroups</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerTriggerGroup">
[]EventListenerTriggerGroup
</a>
</em>
</td>
<td>
<p>Trigger groups allow for centralized processing of an interceptor chain</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.NamespaceSelector">
NamespaceSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resources</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Resources">
Resources
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cloudEventURI</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerStatus">
EventListenerStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.Trigger">Trigger
</h3>
<div>
<p>Trigger defines a mapping of an input event to parameters. This is used
to extract information from events to be passed to TriggerTemplates within a
Trigger.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Trigger</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpec">
TriggerSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the Trigger</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>bindings</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpecBinding">
[]TriggerSpecBinding
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpecTemplate">
TriggerSpecTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountName optionally associates credentials with each trigger;
Unlike EventListeners, this should be scoped to the same namespace
as the Trigger itself</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerBinding">TriggerBinding
</h3>
<div>
<p>TriggerBinding defines a mapping of an input event to parameters. This is used
to extract information from events to be passed to TriggerTemplates within a
Trigger.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
triggers.tekton.dev/v1beta1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>TriggerBinding</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerBindingSpec">
TriggerBindingSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the TriggerBinding</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Param">
[]Param
</a>
</em>
</td>
<td>
<p>Params defines the parameter mapping from the given input event.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerBindingStatus">
TriggerBindingStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.BitbucketInterceptor">BitbucketInterceptor
</h3>
<div>
<p>BitbucketInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>eventTypes</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.CELInterceptor">CELInterceptor
</h3>
<div>
<p>CELInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>filter</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>overlays</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.CELOverlay">
[]CELOverlay
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.CELOverlay">CELOverlay
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.CELInterceptor">CELInterceptor</a>)
</p>
<div>
<p>CELOverlay provides a way to modify the request body using CEL expressions</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>expression</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.CheckType">CheckType
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.GithubOwners">GithubOwners</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;all&#34;</p></td>
<td><p>Set the checkType to all if both repo members or org members can submit or comment on PR to proceed</p>
</td>
</tr><tr><td><p>&#34;none&#34;</p></td>
<td><p>Set the checkType to none if neither of repo members or org members can not submit or comment on PR to proceed</p>
</td>
</tr><tr><td><p>&#34;orgMembers&#34;</p></td>
<td><p>Set the checkType to orgMembers to allow org members to submit or comment on PR to proceed</p>
</td>
</tr><tr><td><p>&#34;repoMembers&#34;</p></td>
<td><p>Set the checkType to repoMembers to allow repo members to submit or comment on PR to proceed</p>
</td>
</tr></tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.CustomResource">CustomResource
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.Resources">Resources</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>RawExtension</code><br/>
<em>
k8s.io/apimachinery/pkg/runtime.RawExtension
</em>
</td>
<td>
<p>
(Members of <code>RawExtension</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListenerConfig">EventListenerConfig
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerStatus">EventListenerStatus</a>)
</p>
<div>
<p>EventListenerConfig stores configuration for resources generated by the
EventListener</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>generatedName</code><br/>
<em>
string
</em>
</td>
<td>
<p>GeneratedResourceName is the name given to all resources reconciled by
the EventListener</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListenerSpec">EventListenerSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListener">EventListener</a>)
</p>
<div>
<p>EventListenerSpec defines the desired state of the EventListener, represented
by a list of Triggers.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggers</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerTrigger">
[]EventListenerTrigger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggerGroups</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerTriggerGroup">
[]EventListenerTriggerGroup
</a>
</em>
</td>
<td>
<p>Trigger groups allow for centralized processing of an interceptor chain</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.NamespaceSelector">
NamespaceSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resources</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Resources">
Resources
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>cloudEventURI</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListenerStatus">EventListenerStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListener">EventListener</a>)
</p>
<div>
<p>EventListenerStatus holds the status of the EventListener</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Status</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#Status">
knative.dev/pkg/apis/duck/v1.Status
</a>
</em>
</td>
<td>
<p>
(Members of <code>Status</code> are embedded into this type.)
</p>
</td>
</tr>
<tr>
<td>
<code>AddressStatus</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1beta1#AddressStatus">
knative.dev/pkg/apis/duck/v1beta1.AddressStatus
</a>
</em>
</td>
<td>
<p>
(Members of <code>AddressStatus</code> are embedded into this type.)
</p>
<p>EventListener is Addressable. It currently exposes the service DNS
address of the the EventListener sink</p>
</td>
</tr>
<tr>
<td>
<code>configuration</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerConfig">
EventListenerConfig
</a>
</em>
</td>
<td>
<p>Configuration stores configuration for the EventListener service</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListenerTrigger">EventListenerTrigger
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerSpec">EventListenerSpec</a>)
</p>
<div>
<p>EventListenerTrigger represents a connection between TriggerBinding, Params,
and TriggerTemplate; TriggerBinding provides extracted values for
TriggerTemplate to then create resources from. TriggerRef can also be
provided instead of TriggerBinding, Interceptors and TriggerTemplate</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>bindings</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpecBinding">
[]TriggerSpecBinding
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpecTemplate">
TriggerSpecTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggerRef</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountName optionally associates credentials with each trigger;
more granular authorization for
who is allowed to utilize the associated pipeline
vs. defaulting to whatever permissions are associated
with the entire EventListener and associated sink facilitates
multi-tenant model based scenarios</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListenerTriggerGroup">EventListenerTriggerGroup
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerSpec">EventListenerSpec</a>)
</p>
<div>
<p>EventListenerTriggerGroup defines a group of Triggers that share a common set of interceptors</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>triggerSelector</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.EventListenerTriggerSelector">
EventListenerTriggerSelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.EventListenerTriggerSelector">EventListenerTriggerSelector
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerTriggerGroup">EventListenerTriggerGroup</a>)
</p>
<div>
<p>EventListenerTriggerSelector  defines ways to select a group of triggers using their metadata</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespaceSelector</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.NamespaceSelector">
NamespaceSelector
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.GitHubInterceptor">GitHubInterceptor
</h3>
<div>
<p>GitHubInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>eventTypes</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>addChangedFiles</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.GithubAddChangedFiles">
GithubAddChangedFiles
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>githubOwners</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.GithubOwners">
GithubOwners
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.GitLabInterceptor">GitLabInterceptor
</h3>
<div>
<p>GitLabInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretRef</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>eventTypes</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.GithubAddChangedFiles">GithubAddChangedFiles
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.GitHubInterceptor">GitHubInterceptor</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code><br/>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>personalAccessToken</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.GithubOwners">GithubOwners
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.GitHubInterceptor">GitHubInterceptor</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enabled</code><br/>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>personalAccessToken</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.SecretRef">
SecretRef
</a>
</em>
</td>
<td>
<p>This param/variable is required for private repos or when checkType is set to orgMembers or repoMembers or all</p>
</td>
</tr>
<tr>
<td>
<code>checkType</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.CheckType">
CheckType
</a>
</em>
</td>
<td>
<p>Set the value to one of the supported values (orgMembers, repoMembers, both, none)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.InterceptorInterface">InterceptorInterface
</h3>
<div>
</div>
<h3 id="triggers.tekton.dev/v1beta1.InterceptorKind">InterceptorKind
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.InterceptorRef">InterceptorRef</a>)
</p>
<div>
<p>InterceptorKind defines the type of Interceptor used by the Trigger.</p>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ClusterInterceptor&#34;</p></td>
<td><p>ClusterInterceptorKind indicates that Interceptor type has a cluster scope.</p>
</td>
</tr><tr><td><p>&#34;NamespacedInterceptor&#34;</p></td>
<td><p>NamespacedInterceptorKind indicates that Interceptor type has a namespace scope.</p>
</td>
</tr></tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.InterceptorParams">InterceptorParams
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>InterceptorParams defines a key-value pair that can be passed on an interceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1.JSON
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.InterceptorRef">InterceptorRef
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>InterceptorRef provides a Reference to a ClusterInterceptor</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name of the referent; More info: <a href="http://kubernetes.io/docs/user-guide/identifiers#names">http://kubernetes.io/docs/user-guide/identifiers#names</a></p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.InterceptorKind">
InterceptorKind
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>InterceptorKind indicates the kind of the Interceptor, namespaced or cluster scoped.</p>
</td>
</tr>
<tr>
<td>
<code>apiVersion</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>API version of the referent</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.InterceptorRequest">InterceptorRequest
</h3>
<div>
<p>Do not generate DeepCopy(). See #827</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>body</code><br/>
<em>
string
</em>
</td>
<td>
<p>Body is the incoming HTTP event body. We use a &ldquo;string&rdquo; representation of the JSON body
in order to preserve the body exactly as it was sent (including spaces etc.). This is necessary
for some interceptors e.g. GitHub for validating the body with a signature. While []byte can also
store an exact representation of the body, <code>json.Marshal</code> will compact []byte to a base64 encoded
string which means that we will lose the spaces any time we marshal this struct.</p>
</td>
</tr>
<tr>
<td>
<code>header</code><br/>
<em>
map[string][]string
</em>
</td>
<td>
<p>Header are the headers for the incoming HTTP event</p>
</td>
</tr>
<tr>
<td>
<code>extensions</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>Extensions are extra values that are added by previous interceptors in a chain</p>
</td>
</tr>
<tr>
<td>
<code>interceptor_params</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>InterceptorParams are the user specified params for interceptor in the Trigger</p>
</td>
</tr>
<tr>
<td>
<code>context</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerContext">
TriggerContext
</a>
</em>
</td>
<td>
<p>Context contains additional metadata about the event being processed</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.InterceptorResponse">InterceptorResponse
</h3>
<div>
<p>Do not generate Deepcopy(). See #827</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>extensions</code><br/>
<em>
map[string]interface{}
</em>
</td>
<td>
<p>Extensions are additional fields that is added to the interceptor event.</p>
</td>
</tr>
<tr>
<td>
<code>continue</code><br/>
<em>
bool
</em>
</td>
<td>
<p>Continue indicates if the EventListener should continue processing the Trigger or not</p>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Status">
Status
</a>
</em>
</td>
<td>
<p>Status is an Error status containing details on any interceptor processing errors</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.KubernetesResource">KubernetesResource
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.Resources">Resources</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>replicas</code><br/>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceType</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#servicetype-v1-core">
Kubernetes core/v1.ServiceType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>servicePort</code><br/>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#WithPodSpec">
knative.dev/pkg/apis/duck/v1.WithPodSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis/duck/v1#PodSpecable">
knative.dev/pkg/apis/duck/v1.PodSpecable
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.NamespaceSelector">NamespaceSelector
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerSpec">EventListenerSpec</a>, <a href="#triggers.tekton.dev/v1beta1.EventListenerTriggerSelector">EventListenerTriggerSelector</a>)
</p>
<div>
<p>NamespaceSelector is a selector for selecting either all namespaces or a
list of namespaces.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>matchNames</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>List of namespace names.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.Param">Param
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerBindingSpec">TriggerBindingSpec</a>)
</p>
<div>
<p>Param defines a string value to be used for a ParamSpec with the same name.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.ParamSpec">ParamSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerTemplateSpec">TriggerTemplateSpec</a>)
</p>
<div>
<p>ParamSpec defines an arbitrary named  input whose value can be supplied by a
<code>Param</code>.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name declares the name by which a parameter is referenced.</p>
</td>
</tr>
<tr>
<td>
<code>description</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Description is a user-facing description of the parameter that may be
used to populate a UI.</p>
</td>
</tr>
<tr>
<td>
<code>default</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default is the value a parameter takes if no input value via a Param is supplied.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.Resources">Resources
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerSpec">EventListenerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>kubernetesResource</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.KubernetesResource">
KubernetesResource
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>customResource</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.CustomResource">
CustomResource
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.SecretRef">SecretRef
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.BitbucketInterceptor">BitbucketInterceptor</a>, <a href="#triggers.tekton.dev/v1beta1.GitHubInterceptor">GitHubInterceptor</a>, <a href="#triggers.tekton.dev/v1beta1.GitLabInterceptor">GitLabInterceptor</a>, <a href="#triggers.tekton.dev/v1beta1.GithubAddChangedFiles">GithubAddChangedFiles</a>, <a href="#triggers.tekton.dev/v1beta1.GithubOwners">GithubOwners</a>)
</p>
<div>
<p>SecretRef contains the information required to reference a single secret string
This is needed because the other secretRef types are not cross-namespace and do not
actually contain the &ldquo;SecretName&rdquo; field, which allows us to access a single secret value.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>secretKey</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>secretName</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.SlackInterceptor">SlackInterceptor
</h3>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>requestedFields</code><br/>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.Status">Status
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.InterceptorResponse">InterceptorResponse</a>, <a href="#triggers.tekton.dev/v1beta1.StatusError">StatusError</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>code</code><br/>
<em>
<a href="https://pkg.go.dev/google.golang.org/grpc/codes#Code">
google.golang.org/grpc/codes.Code
</a>
</em>
</td>
<td>
<p>The status code, which should be an enum value of [google.rpc.Code][google.rpc.Code].</p>
</td>
</tr>
<tr>
<td>
<code>message</code><br/>
<em>
string
</em>
</td>
<td>
<p>A developer-facing error message, which should be in English.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.StatusError">StatusError
</h3>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>s</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Status">
Status
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerBindingInterface">TriggerBindingInterface
</h3>
<div>
<p>TriggerBindingInterface is implemented by TriggerBinding and ClusterTriggerBinding</p>
</div>
<h3 id="triggers.tekton.dev/v1beta1.TriggerBindingKind">TriggerBindingKind
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerSpecBinding">TriggerSpecBinding</a>)
</p>
<div>
<p>Check that EventListener may be validated and defaulted.
TriggerBindingKind defines the type of TriggerBinding used by the EventListener.</p>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;ClusterTriggerBinding&#34;</p></td>
<td><p>ClusterTriggerBindingKind indicates that triggerbinding type has a cluster scope.</p>
</td>
</tr><tr><td><p>&#34;TriggerBinding&#34;</p></td>
<td><p>NamespacedTriggerBindingKind indicates that triggerbinding type has a namespace scope.</p>
</td>
</tr></tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerBindingSpec">TriggerBindingSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.ClusterTriggerBinding">ClusterTriggerBinding</a>, <a href="#triggers.tekton.dev/v1beta1.TriggerBinding">TriggerBinding</a>)
</p>
<div>
<p>TriggerBindingSpec defines the desired state of the TriggerBinding.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.Param">
[]Param
</a>
</em>
</td>
<td>
<p>Params defines the parameter mapping from the given input event.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerBindingStatus">TriggerBindingStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.ClusterTriggerBinding">ClusterTriggerBinding</a>, <a href="#triggers.tekton.dev/v1beta1.TriggerBinding">TriggerBinding</a>)
</p>
<div>
<p>TriggerBindingStatus defines the observed state of TriggerBinding.</p>
</div>
<h3 id="triggers.tekton.dev/v1beta1.TriggerContext">TriggerContext
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.InterceptorRequest">InterceptorRequest</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>event_url</code><br/>
<em>
string
</em>
</td>
<td>
<p>EventURL is the URL of the incoming event</p>
</td>
</tr>
<tr>
<td>
<code>event_id</code><br/>
<em>
string
</em>
</td>
<td>
<p>EventID is a unique ID assigned by Triggers to each event</p>
</td>
</tr>
<tr>
<td>
<code>trigger_id</code><br/>
<em>
string
</em>
</td>
<td>
<p>TriggerID is of the form namespace/$ns/triggers/$name</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerInterceptor">TriggerInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerTrigger">EventListenerTrigger</a>, <a href="#triggers.tekton.dev/v1beta1.EventListenerTriggerGroup">EventListenerTriggerGroup</a>, <a href="#triggers.tekton.dev/v1beta1.TriggerSpec">TriggerSpec</a>)
</p>
<div>
<p>TriggerInterceptor provides a hook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Optional name to identify the current interceptor configuration</p>
</td>
</tr>
<tr>
<td>
<code>ref</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.InterceptorRef">
InterceptorRef
</a>
</em>
</td>
<td>
<p>Ref refers to the Interceptor to use</p>
</td>
</tr>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.InterceptorParams">
[]InterceptorParams
</a>
</em>
</td>
<td>
<p>Params are the params to send to the interceptor</p>
</td>
</tr>
<tr>
<td>
<code>webhook</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.WebhookInterceptor">
WebhookInterceptor
</a>
</em>
</td>
<td>
<p>WebhookInterceptor refers to an old style webhook interceptor service</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerResourceTemplate">TriggerResourceTemplate
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerTemplateSpec">TriggerTemplateSpec</a>)
</p>
<div>
<p>TriggerResourceTemplate describes a resource to create</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>RawExtension</code><br/>
<em>
k8s.io/apimachinery/pkg/runtime.RawExtension
</em>
</td>
<td>
<p>
(Members of <code>RawExtension</code> are embedded into this type.)
</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerSpec">TriggerSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.Trigger">Trigger</a>)
</p>
<div>
<p>TriggerSpec represents a connection between TriggerSpecBinding,
and TriggerSpecTemplate; TriggerSpecBinding provides extracted values for
TriggerSpecTemplate to then create resources from.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>bindings</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpecBinding">
[]TriggerSpecBinding
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerSpecTemplate">
TriggerSpecTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>interceptors</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">
[]TriggerInterceptor
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serviceAccountName</code><br/>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ServiceAccountName optionally associates credentials with each trigger;
Unlike EventListeners, this should be scoped to the same namespace
as the Trigger itself</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerSpecBinding">TriggerSpecBinding
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerTrigger">EventListenerTrigger</a>, <a href="#triggers.tekton.dev/v1beta1.TriggerSpec">TriggerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the binding param
Mutually exclusive with Ref</p>
</td>
</tr>
<tr>
<td>
<code>value</code><br/>
<em>
string
</em>
</td>
<td>
<p>Value is the value of the binding param. Can contain JSONPath
Has to be pointer since &ldquo;&rdquo; is a valid value
Required if Name is also specified.</p>
</td>
</tr>
<tr>
<td>
<code>ref</code><br/>
<em>
string
</em>
</td>
<td>
<p>Ref is a reference to a TriggerBinding kind.
Mutually exclusive with Name</p>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerBindingKind">
TriggerBindingKind
</a>
</em>
</td>
<td>
<p>Kind can only be provided if Ref is also provided. Defaults to TriggerBinding</p>
</td>
</tr>
<tr>
<td>
<code>apiversion</code><br/>
<em>
string
</em>
</td>
<td>
<p>APIVersion of the binding ref</p>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerSpecTemplate">TriggerSpecTemplate
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.EventListenerTrigger">EventListenerTrigger</a>, <a href="#triggers.tekton.dev/v1beta1.TriggerSpec">TriggerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ref</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>apiversion</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerTemplateSpec">
TriggerTemplateSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerTemplate">TriggerTemplate
</h3>
<div>
<p>TriggerTemplate takes parameters and uses them to create CRDs</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerTemplateSpec">
TriggerTemplateSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec holds the desired state of the TriggerTemplate from the client</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.ParamSpec">
[]ParamSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resourcetemplates</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerResourceTemplate">
[]TriggerResourceTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerTemplateStatus">
TriggerTemplateStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerTemplateSpec">TriggerTemplateSpec
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerSpecTemplate">TriggerSpecTemplate</a>, <a href="#triggers.tekton.dev/v1beta1.TriggerTemplate">TriggerTemplate</a>)
</p>
<div>
<p>TriggerTemplateSpec holds the desired state of TriggerTemplate</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>params</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.ParamSpec">
[]ParamSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resourcetemplates</code><br/>
<em>
<a href="#triggers.tekton.dev/v1beta1.TriggerResourceTemplate">
[]TriggerResourceTemplate
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="triggers.tekton.dev/v1beta1.TriggerTemplateStatus">TriggerTemplateStatus
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerTemplate">TriggerTemplate</a>)
</p>
<div>
<p>TriggerTemplateStatus describes the desired state of TriggerTemplate</p>
</div>
<h3 id="triggers.tekton.dev/v1beta1.WebhookInterceptor">WebhookInterceptor
</h3>
<p>
(<em>Appears on:</em><a href="#triggers.tekton.dev/v1beta1.TriggerInterceptor">TriggerInterceptor</a>)
</p>
<div>
<p>WebhookInterceptor provides a webhook to intercept and pre-process events</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>objectRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectreference-v1-core">
Kubernetes core/v1.ObjectReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObjectRef is a reference to an object that will resolve to a cluster DNS
name to use as the EventInterceptor. Either objectRef or url can be specified</p>
</td>
</tr>
<tr>
<td>
<code>url</code><br/>
<em>
<a href="https://pkg.go.dev/knative.dev/pkg/apis#URL">
knative.dev/pkg/apis.URL
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>header</code><br/>
<em>
<a href="https://pkg.go.dev/github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1#Param">
[]github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1.Param
</a>
</em>
</td>
<td>
<p>Header is a group of key-value pairs that can be appended to the
interceptor request headers. This allows the interceptor to make
decisions specific to an EventListenerTrigger.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
.
</em></p>
