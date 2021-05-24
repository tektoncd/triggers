# Tekton Triggers Roadmap

In 2021, we'd like to get the Triggers APIs to a beta level of stability, add popular 
feature requests, and plan for a v1 release.

## H1 2021:
* Get to Beta!
    * [Trigger Templates - JSON escaping, validation, and templating](https://github.com/tektoncd/triggers/issues/697) 
    * Trigger CRD - Path based Triggers
    * v1beta1 version upgrade
    * Beta docs/examples overhaul

* Operator features:
    * [RBAC/Permission setup for multi-tenant usage](https://github.com/tektoncd/triggers/issues/77)
    * Customizable EventListeners
        * Scale to Zero (Knative) EventListener
        * More podTemplate customization options
        
* Performance
    * Define SLIs/SLOs
    * Metrics for EventListeners
    
* TriggerInvocations/Results integration (store incoming events in Results)

## H2:
* Post Beta End User features
    * Polling support
    * Config as Code support
    * [Alternate ways of running Triggers](https://github.com/tektoncd/triggers/issues/504)
        * Less verbose way to doing common things e.g. cron/git
        * Easier setup e.g. setup webhooks automatically
        * Make the CLI part of tkn surface (tkn trigger run)
    * Catalog support story
        * Define what should be reusable 

* Notifications Integration

* Plan for v1!
