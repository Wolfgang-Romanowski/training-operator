\# Observability UI Plugins



Using the Observability UI, you can install and manage plugins that extend the observability functionality of the OpenShift web console. Plugins are installed and managed by the Observability Operator.



\## Plugins



\- \[dashboards](#dashboards): Add enhanced dashboards to the OpenShift web console. This plugin allows you to add other Prometheus datasources present in the cluster, apart from the in-cluster one, to the default dashboards.

\- \[troubleshooting-panel](#troubleshooting-panel): Add the troubleshooting panel to the OpenShift web console. This plugin adds a troubleshooting panel to the console dashboard, which queries and displays results from \[Korrel8r](https://github.com/korrel8r/korrel8r) to help troubleshoot issues.

\- \[distributed-tracing](#distributed-tracing): Add the Observability > Traces page to the Openshift web console. This plugin allows a user to select a \[Tempo](https://docs.openshift.com/container-platform/4.13/observability/distr\_tracing/distr\_tracing\_arch/distr-tracing-architecture.html#distr-tracing-architecture\_distributed-tracing-architecture) instance and view trace data from it.

\- \[monitoring](#monitoring): Add the a number of Observing pages to the Openshift web related to Alerting. This plugin allows a user to view Alerts, Silences, and Alert rules.



| \_\_COO Version\_\_ |   \_\_OCP Versions\_\_  | \_\_Dashboards\_\_ | \_\_Distributed Tracing\_\_ | \_\_Logging\_\_ | \_\_Troubleshooting Panel\_\_ | \_\_Monitoring\_\_ |

| --------------- | ------------------- | -------------- | ----------------------- | ----------- | ------------------------- | ---------------|

| 0.2.0           | 4.11                |       ✔        |             ✘           |       ✘     |             ✘             |       ✘       |

| 0.3.0 - 0.4.0   | 4.11 - 4.15         |       ✔        |             ✔           |       ✔     |             ✘             |       ✘       |

| 0.3.0 - 0.4.0   | 4.16+               |       ✔        |             ✔           |       ✔     |             ✔             |       ✘       |

| 1.0.0+          | 4.11 - 4.14         |       ✔        |             ✔           |       ✔     |             ✘             |       ✘       |

| 1.0.0+          | 4.15                |       ✔        |             ✔           |       ✔     |             ✘             |       ✔       |

| 1.0.0+          | 4.16+               |       ✔        |             ✔           |       ✔     |             ✔             |       ✔       |



Some plugin offer additional features that are available dependant on the cluster version. COO will always deploy all features available for the cluster it is running on.



\### Dashboards



The plugin will search for datasources as ConfigMaps in the `openshift-config-managed` namespace with the `console.openshift.io/dashboard-datasource: 'true'` label. The namespace `openshift-config-managed` is required, more details on how to create a datasource ConfigMap can be found in the \[console-dashboards-plugin](https://github.com/openshift/console-dashboards-plugin/blob/main/docs/add-datasource.md)



\#### Plugin Creation



To enable the console dashboards plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the console dashboards plugin:



```yaml

apiVersion: observability.openshift.io/v1alpha1

kind: UIPlugin

metadata:

&nbsp; name: dashboards

spec:

&nbsp; type: Dashboards

```



\#### Feature Matrix



| \_\_COO Version\_\_ |   \_\_OCP Versions\_\_  | \_\_Features\_\_                                          |

| --------------- | ------------------- | ----------------------------------------------------- |

| 0.2.0+          | 4.11+               | \_No features configuration, just core functionality\_  |



\### Troubleshooting Panel



The plugin adds a UI panel meant to assist in the troubleshooting journey, through exploring related pages. Creating this `UIPlugin` will deploy a \[Korrel8r](https://github.com/korrel8r/korrel8r) service named `korrel8r` in the same namespace which is able to locate related observability signals and kubernetes resources from its correlation engine.



To use the Troubleshooting Panel, in the admin perspective navigate to `Observe > Alerts` and then select an alert. If the alert has correlated items then a "Troubleshooting Panel" will appear above the chart on the alert detail page. This button opens a panel consisting of query details and a topology graph of the query results. The alert page you are on is converted into a Korrel8r query string and sent to the `korrel8r` service. The results are displayed as a graph network connecting the returned signals and resources. The nodes on the graph will take you to the corresponding OpenShift wab console pages when clicked.



\#### Plugin Creation



To enable the troubleshooting panel console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the troubleshooting panel console plugin:



```yaml

apiVersion: observability.openshift.io/v1alpha1

kind: UIPlugin

metadata:

&nbsp; name: troubleshooting-panel

spec:

&nbsp; type: TroubleshootingPanel

```



\#### Feature Matrix



| \_\_COO Version\_\_ |   \_\_OCP Versions\_\_  | \_\_Features\_\_                                          |

| --------------- | ------------------- | ----------------------------------------------------- |

| 0.3.0+          | 4.16+               | \_No features configuration, just core functionality\_  |



\### Distributed Tracing



The plugin adds tracing related UI features to the OpenShift web console. A new tab can be located in the admin perspective under `Observe > Traces`. This tab allows a user to select a supported Tempo instance (\[TempoStack](https://docs.openshift.com/container-platform/4.16/observability/distr\_tracing/distr\_tracing\_tempo/distr-tracing-tempo-installing.html#installing-a-tempostack-instance) or \[TempoMonolithic](https://docs.openshift.com/container-platform/4.16/observability/distr\_tracing/distr\_tracing\_tempo/distr-tracing-tempo-installing.html#installing-a-tempomonolithic-instance) with multi-tenancy) running in their cluster as well as a set a time range and query for the traces being loaded. These traces are displayed on a scatter-plot showing the trace start time, duration, and number of spans. Underneath the scatter plot there is a list of traces showing information such as the `Trace Name`, number of `Spans`, and `Duration`. The trace name contains a link which takes the user to the trace detail page for the selected trace containing a Gantt Chart of all of the spans within the trace. Once selected, the spans show a breakdown of their configured attributes.



\#### Plugin Creation



To enable to distributed tracing console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the distributed tracing console plugin:



```yaml

apiVersion: observability.openshift.io/v1alpha1

kind: UIPlugin

metadata:

&nbsp; name: distributed-tracing

spec:

&nbsp; type: DistributedTracing

```



\#### Feature Matrix



| \_\_COO Version\_\_ |   \_\_OCP Versions\_\_  | \_\_Features\_\_                                          |

| --------------- | ------------------- | ----------------------------------------------------- |

| 0.3.0+          | 4.11+               | \_No features configuration, just core functionality\_  |



\### Logging



The plugin adds various Logging UI functionalities to the OpenShift web console. The core functionality of this plugin adds a new admin perspective tab `Observe > Logs`. This page includes query and log filters, the list of logs, and a histogram showing log frequency by severity. The logs can be filtered by an number of tags, such as `tenant` and `namespace`. The page also has controls for the length of time to query over, the refresh rate of the logging page, and whether to show kubernetes resource information in the results, such as `pod` and `container`. The results section shows a list of collapsed logs, which can then be expanded to show more detailed information for each log.



When a \_\_TroubleshootingPanel\_\_ `UIPlugin` is deployed the plugin will connect the \[Korrel8r](https://github.com/korrel8r/korrel8r) service and add direct links from the admin perspective `Observe > Logs` page to the `Observe > Metrics` page with a correlated PromQL query. It will also add a "See Related Logs" link from the admin perspective `Observe > Alerting` on an alerting detail page to the `Observe > Logs` page with a correlated filter set selected.



\#### Feature List



| \_\_Feature\_\_   | \_\_Description\_\_                                                                                                                                                            | \_\_Support Level\_\_ |

| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------- |

| `dev-console` | Adds the logging view to the developer perspective                                                                                                                         | General Availability |

| `alerts`      | Merges the OpenShift console alerts with log-based alerts defined in the Loki ruler. Adds a log-based metrics chart in the alert detail view                               | General Availability |

| `dev-alerts`  | Merges the OpenShift console alerts with log-based alerts defined in the Loki ruler. Adds a log-based metrics chart in the alert detail view for the developer perspective | General Availability |



\#### Feature Matrix



| \_\_COO version\_\_ | \_\_OCP versions\_\_ | \_\_Features\_\_                                          |

| --------------- | ---------------- | ----------------------------------------------------- |

| 0.3.0+          | 4.11             | \_No features configuration, just core functionality\_  |

| 0.3.0+          | 4.12             | `dev-console`                                         |

| 0.3.0+          | 4.13             | `dev-console`, `alerts`                               |

| 0.3.0+          | 4.14+            | `dev-console`, `alerts`, `dev-alerts`                 |



\#### Plugin Creation



To enable to logging view plugin, create a `UIPlugin` CR. This CR has three parameters located under `spec.logging`, used to control the behavior of the logging-view-plugin. The `spec.logging.lokiStack` required parameter locates the LokiStack instance in the `openshift-logging` namespace to connect to. The `spec.logging.logLimit` and the `spec.logging.timeout` determine the number of logs returned from a query and the time before the query timeouts respectively.



The following example shows how to create a CR to enable the logging view plugin:



```yaml

apiVersion: observability.openshift.io/v1alpha1

kind: UIPlugin

metadata:

&nbsp; name: logging

spec:

&nbsp; type: Logging

&nbsp; logging:

&nbsp;   lokiStack:

&nbsp;     name: logging-loki

&nbsp;   logsLimit: 50

&nbsp;   timeout: 30s

&nbsp;   schema: otel

```



\### Monitoring



The plugin adds monitoring related UI features to the OpenShift web console, related to the Advance Cluster Management (ACM) perspective, incidents (cluster health analysis), and \[Perses](https://github.com/perses/perses). A number of new pages and features are enabled through this plugin. Including, but not limited to:

\- `ACM > Observe > Alerting`

\- `ACM > Observe > Alerting > Silences`

\- `ACM > Observe > Alerting > Alert rules`

\- `OCP > Observe > Dashboards (Perses)`

\- `OCP > Observe > Incidents`



To deploy ACM related features the `acm-alerting` configuration must be enabled. In the UIPlugin Custom Resource (CR) you must pass the Alertmanager and ThanosQuerier Service endpoint (e.g. `https://alertmanager.open-cluster-management-observability.svc:9095` and `https://rbac-query-proxy.open-cluster-management-observability.svc:8443`). See the example in the next section `Plugin Creation.`



To deploy the Incidents feature, the `incidents` configuration must be enabled. See the example in the next section, `Plugin Creation.` 



To deploy the Perses dashboard feature, the `perses-dashboards` configuration must be enabled. In the UIPlugin CR, you can optionally pass the service name and namespace of your Perses instance (e.g., `serviceName: perses-api-http` and `namespace: perses`). If these fields are left blank and `spec.monitoring.perses.enabled: true`, then default values will be assigned. These default values are `serviceName: perses-api-http` and `namespace: perses`. See the example in the next section, `Plugin Creation.` 

Besides, when `spec.monitoring.perses.enabled: true`, Accelerator Perses dashboard and Accelerator Perses datasource are both created.



ObO/COO operator creates the following roles:

\- persesdashboard-editor-role - ability to create, read, update and delete perses dashboards CRD instance presented on ObO/COO operator under PersesDashboards tab, and view perses dashboards presentation in Dashboards (Perses)

\- persesdashboard-viewer-role - ability to only read/view perses dashboards CRD instance presented on ObO/COO operator under PersesDashboards tab, and view perses dashboards presentation in Dashboards (Perses)

\- persesdatasource-editor-role - ability to create, read, update and delete perses datasources CRD instance presented on ObO/COO operator under PersesDatasources tab, and view perses dashboards with data being loaded from perses datasource in Dashboards (Perses)

\- persesdatasource-viewer-role - ability to only read/view perses datasources CRD instance presented on ObO/COO operator under PersesDatasources tab, and view perses dashboards with data being loaded from perses datasource in Dashboards (Perses)



When assigned via ClusterRoleBinding, user has access to all perses dashboards and perses datasources presented in all namespaces/projects. When assigned via RoleBinding, user has access to all perses dashboards and perses datasources presented in a given namespace/project.



Examples:

\- user1 RoleBinding as persesdashboard-viewer-role and persesdatasource-viewer-role in openshift-cluster-observability-operator namespace:

```yaml

kind: RoleBinding

apiVersion: rbac.authorization.k8s.io/v1

metadata:

&nbsp; name: user1-viewer-dashboard

&nbsp; namespace: openshift-cluster-observability-operator

subjects:

&nbsp; - kind: User

&nbsp;   apiGroup: rbac.authorization.k8s.io

&nbsp;   name: user1

roleRef:

&nbsp; apiGroup: rbac.authorization.k8s.io

&nbsp; kind: ClusterRole

&nbsp; name: persesdashboard-viewer-role



kind: RoleBinding

apiVersion: rbac.authorization.k8s.io/v1

metadata:

&nbsp; name: user1-viewer-datasource

&nbsp; namespace: openshift-cluster-observability-operator

subjects:

&nbsp; - kind: User

&nbsp;   apiGroup: rbac.authorization.k8s.io

&nbsp;   name: user1

roleRef:

&nbsp; apiGroup: rbac.authorization.k8s.io

&nbsp; kind: ClusterRole

&nbsp; name: persesdatasource-viewer-role

```



\- user1 ClusterRoleBinding as persesdashboard-editor-role and persesdatasource-editor-role:

```yaml

kind: ClusterRoleBinding

apiVersion: rbac.authorization.k8s.io/v1

metadata:

&nbsp; name: user1-editor-dashboard

subjects:

&nbsp; - kind: User

&nbsp;   apiGroup: rbac.authorization.k8s.io

&nbsp;   name: user1

roleRef:

&nbsp; apiGroup: rbac.authorization.k8s.io

&nbsp; kind: ClusterRole

&nbsp; name: persesdashboard-editor-role



kind: ClusterRoleBinding

apiVersion: rbac.authorization.k8s.io/v1

metadata:

&nbsp; name: user1-editor-datasource

subjects:

&nbsp; - kind: User

&nbsp;   apiGroup: rbac.authorization.k8s.io

&nbsp;   name: user1

roleRef:

&nbsp; apiGroup: rbac.authorization.k8s.io

&nbsp; kind: ClusterRole

&nbsp; name: persesdatasource-editor-role

```

&nbsp;

Other pages which are typically distributed with the monitoring-plugin, such as `Admin > Observe > Dashboards`, are only available in the monitoring-plugin when deployed through \[CMO](https://github.com/openshift/cluster-monitoring-operator).



\#### Plugin Creation



To enable to monitoring console plugin, create a `UIPlugin` CR. The following example shows how to create a CR to enable the monitoring console plugin:



```yaml

apiVersion: observability.openshift.io/v1alpha1

kind: UIPlugin

metadata:

&nbsp; name: monitoring

spec:

&nbsp; type: Monitoring

&nbsp; monitoring:

&nbsp;   acm:

&nbsp;     enabled: true

&nbsp;     alertmanager:

&nbsp;       url: 'https://alertmanager.open-cluster-management-observability.svc:9095'

&nbsp;     thanosQuerier:

&nbsp;       url: 'https://rbac-query-proxy.open-cluster-management-observability.svc:8443'

&nbsp;   perses:

&nbsp;     enabled: true

&nbsp;   incidents:

&nbsp;     enabled: true

```



\#### Feature List



| \_\_Feature\_\_         | \_\_Description\_\_                                                                                                          | \_\_Support Level\_\_ |

| ------------------- | ------------------------------------------------------------------------------------------------------------------------ | ----------------- |

| `acm-alerting`      | Adds alerting UI to multi-cluster view. Configures proxies to connect with any alertmanager and thanos-querier.          | Dev Preview      |

| `incidents`         | Adds incidents UI to `Observe` section of OpenShift Console Platform. Deploys the \[Cluster Health Analyzer](https://github.com/openshift/cluster-health-analyzer) and configures proxies in the plugin to connect with it. | Tech Preview      |

| `perses-dashboards` | Adds perses UI to `Observe` section of OpenShift Console Platform. Configures proxies to connect with a Perses instance. Installs Accelerator Perses Dashboard and Accelerator Perses Datasource. | Dev Preview      |





\#### Feature Matrix



| \_\_COO Version\_\_ |   \_\_OCP Versions\_\_  | \_\_Features\_\_                      |

| --------------- | ------------------- | --------------------------------- |

| 1.0.0+          | 4.14+               | `acm-alerting`                    |

| 1.1.0+          | 4.15+               | `acm-alerting, perses-dashboards` |

| 1.2.0+          | 4.19+               | `acm-alerting, perses-dashboards, incidents` |



RHOAISTRAT-575 -  Platform Metrics and Alerts for Self-Managed and Managed OpenShift AICentralized

Authors: Kevin Howell Steven Tobin Cesar Francisco San Nicolas Martinez Marian Macik

Status: Proposed



Approvers/Stakeholders: 

Reviewed By

Name(s)

Date Reviewed

Approved

BU













Security













SRE













Platform













Trusty AI













Model Registry













Dashboard













Model Registry













Feature Store













IDE













EDGE













Serving Runtimes \& Accelerators













Model management \& Metrics













Ray/KubeRay













Codeflare













Data Science Pipelines













DevOps













UX













Docs

&nbsp;Manuela Ansaldo Jennifer Ciroli









Other















Introduction/problem statement

Red Hat provides many building blocks that can be used to configure observability for any workload running on OpenShift, but OpenShift AI should provide a default out-of-the-box configuration for observability, so that a customer can see key health indicators for their OpenShift AI installations and AI workloads. Metric collection also enables alerting and autoscaling capabilities.



The observability configuration needs to provide a simple consistent way for OpenShift AI components to contribute metrics, traces, and logs, so that, as new features are implemented, the product can provide consistent observability capabilities.



Recognizing that many customers already utilize their own observability solutions, OpenShift AI should provide straightforward mechanisms for exporting metrics, traces, and logs to 3rd party observability platforms.

Personas

OpenShift Cluster Administrator - technical user with Kubernetes/OpenShift administration knowledge \& elevated cluster permissions.

OpenShift Cluster User - technical user with Kubernetes/OpenShift knowledge - typically managing an application deployed on OpenShift. May or may not also be an MLOps or AI Application Developer.

OpenShift AI Cluster Admin - technical user that manages the RHOAI installation including RHOAI configuration. Does not necessarily have cluster administrator access.

OpenShift AI User

Data Scientist - Machine Learning focused user. May not have Kubernetes/OpenShift knowledge.

MLOps Engineer - technical user with Kubernetes/OpenShift knowledge responsible for supporting Data Science use cases (e.g. training, tuning, inference deployments).

AI Application Developer - technical user with Kubernetes/OpenShift knowledge responsible for integrating AI technologies into applications.

Definitions of terms

Core Concepts

Observability lets you understand a system from the outside by letting you ask questions about that system without knowing its inner workings.

Telemetry refers to data emitted from a system and its behavior. The data can come in the form of traces, metrics, and logs.

Metrics are aggregations over a period of time of numeric data about your infrastructure or application. Examples include: system error rate, CPU utilization, and request rate for a given service.

Scraping is a method of collecting metrics, where a scraper periodically gathers metrics from known endpoints (typically HTTP endpoints).

A distributed trace, more commonly known as a trace, records the paths taken by requests (made by an application or end-user) as they propagate through multi-service architectures, like microservice and serverless applications.

A log is a timestamped message emitted by services or other components.

Instrumentation is the process of having services or other components emit signals, such as metrics, traces, and logs.

Auto-instrumentation is a process where the OpenTelemetry operator injects instrumentation libraries into workloads.

The default monitoring stack is a set of components, including Prometheus and Alertmanager, that are deployed as part of OpenShift Container Platform. 

User Workload Monitoring is an optional mode of the default monitoring stack that provides a single secondary Prometheus instance that stores metrics from user workloads.

Upstream Projects

Prometheus collects and stores metrics as time series data.

Alertmanager is a Prometheus component that provides alert management and routing.

Tempo is an open source distributed tracing backend.

Loki is an open source log aggregation backend.

Perses is a dashboard tool that you can use to display Prometheus metrics \& Tempo traces.

The OpenTelemetry Collector offers a vendor-agnostic implementation of how to receive, process and export telemetry data.

KEDA stands for Kubernetes Event-Driven Autoscaling, a project that provides extensible autoscaling, with an expansive ecosystem of integrations.

Downstream Components

Cluster Observability Operator is an optional operator from OpenShift that manages Prometheus, Alertmanager, and Perses.

Tempo Operator is an optional operator from OpenShift that manages Tempo.

Red Hat build of OpenTelemetry provides an Operator that manages OpenTelemetry Collector instances and instrumentation configuration.

Cluster Logging Operator is an optional Operator from OpenShift that provides log collection using Loki and log forwarding capabilities.

Custom Metrics Autoscaler is an operator that provides configurable replica autoscaling driven by metrics, based on KEDA.

Internal Services

Red Hat Observability Service is an internal deployment of Observatorium project maintained by the Openshift Monitoring Team, that stores customer telemetry metrics for internal use cases. Referred to as Observatorium in this document.

Grafana is a visualization tool deployed widely for interacting with observability. https://telemeter-lts-dashboards.datahub.redhat.com/ is the instance commonly used for OpenShift AI.

Dataverse is a Data Platform that provides storage and management of internal Red Hat data including a Data Warehouse/Lake.

Tableau is a visualization service used for internal data at Red Hat. Tableau uses Dataverse and Amazon RedShift as data sources.

User stories

PRIORITY: 

P0 for mandatory for MVP

P1 reduces the friction and increases the functional usefulness of the operator

P2 useful, but does not prevent the use of the operator if not available

STATUS: 

C for Committed, 

D for Deferred 

T for Tentative

PHASE: 

MVP = Minimum Viable Product (a.k.a. Milestone 1)

M# = Milestone #

&nbsp;	

ID

Priority

Description

Status -  Phase

RHOAIENG-25561 

P0

Enable centralized RHOAI metrics collection

MVP - C

RHOAIENG-25562

P0

Enable centralized RHOAI traces collection

MVP - C

RHOAIENG-26160

P0

Enable built in alerting

MVP - C

RHOAIENG-25576

P1

Enable centrally managed observability dashboards

M1 - C

RHOAIENG-25563

P2

Build RHOAI adoption metrics pipeline

M2 - D





Requirements for solution

The following notable tickets depend on capabilities provided by this solution:

RHOAISTRAT-541 Implement Live Drift Monitoring for Generative AI (Systems + Models)

RHOAIRFE-693 Introduce E2E Observability for Llama Stack in the product

RHOAIRFE-570 GPU metrics for distributed workloads 

RHOAIRFE-551 vLLM traces for RHOAI

RHOAIRFE-518 Metrics Logging and Visualization Across Epochs in Experiments and Model Registry



Other related issues are captured in RHOAI Observability Jiras

Current architecture

Currently, monitoring is included for managed OpenShift AI installations only; some components (e.g. model server) integrate with User Workload Monitoring (UWM) if present. Additionally, the official OpenShift AI docs showcase how to use Grafana to observe model serving metrics, based on work from the AI BU. The use case for this monitoring is providing alerting and metrics for SRE to use in support of customer RHOAI installations.



The AI BU also has a kickstart for LlamaStack Observability, that illustrates how to configure many of these components (as well as Grafana and UWM).



The current monitoring aggregates recording and alerting rules from components into prometheus configuration, and deploys the following components:

Alertmanager - Routes alerts to SRE

Blackbox-exporter - Prometheus component to monitor user-facing endpoints

Prometheus - Independently deployed instance of prometheus for metrics



Proposed Architecture



Telemetry Collection

The ODH operator defines an instance of the otel-collector in the gateway mode. The Red Hat Build of OpenTelemetry operator manages this instance. This deployment has 2 replicas by default.



The ODH operator manages configuration of the otel-collector, provide central configuration of telemetry, including common processing (e.g. captures Kubernetes identifiers, including namespace and pod identifiers).

Metrics

The ODH operator defines a central Prometheus instance, managed by the Cluster Observability Operator. This Prometheus instance stores metrics in persistent volumes (one per replica).



The platform configures the otel-collector to perform metrics scraping. (Note: each otel-collector replica scrapes independently, resulting in two scrapes by default). Each Prometheus instance scrapes metrics from the otel-collector deployment (Note: this requires a support exception from the OBS team until the prometheus exporter goes GA).



The otel-collector discovers workloads (both component workloads and user workloads) through use of a well-known common label: monitoring.opendatahub.io/scrape: "true". The otel-collector supports 2 different strategies for discovery:

Components defining ServiceMonitors with the common label to configure discovery.

Components defining PodMonitors with the common label to configure discovery.



The ODH operator manages scraping configuration for Kueue as well as hardware accelerator vendor metrics. The hardware accelerator metrics are normalized following the same conventions as OpenShift Container Platform (see OBSDA-1087), ensuring OpenShift AI and OpenShift Container Platform have consistent views of the accelerators.



The metrics data is queryable through a Thanos Querier frontend (to handle deduplication and merging across the Prometheus replicas), exposed via a stable endpoint. Components may use this to implement autoscaling via the Custom Metrics Autoscaler. J-Proxy uses this endpoint for autoscaling purposes. Perses dashboards use the querier as a data source as well.



The platform provides a namespace restricted view using a combination of kube-rbac-proxy and prom-label-proxy restricts access to users having PodMetrics access (this pattern is used in OpenShift Container Platform).



The platform provides an unrestricted view, restricted to users having NodeMetrics access.



Depending on the use case, access to the querier either uses a port which uses the namespace-restricted view or a separate port which is unrestricted, but authorized via kube-rbac-proxy. This requires a ClusterRole or Role to be added to a component's service account.



The customer can optionally configure the otel-collector to export metrics to a 3rd party observability platform. Support is according to the Red Hat Build of OpenTelemetry support policy; as of 4.18:

GA: otlp

Tech Preview: prometheus, prometheusremotewrite, kafka, cloudwatch, file



If the customer opts into sending Red Hat telemetry, then the otel-collector exports a subset of aggregated metrics to Observatorium (this telemetry is parallel to existing OpenShift Container Platform Telemetry, allowing additional flexibility in collection of adoption metrics). Received customer telemetry will be made available in a couple of different ways:

OpenShift AI engineers use Grafana dashboards to access the OpenShift AI Observatorium tenant (similar to Telemeter Grafana dashboards).

P\&GE Business Insights provides Tableau dashboards.

Traces

The ODH operator provisions an instance of Tempo for storing distributed traces. Tempo stores in user-provided object storage or a persistent volume. When the customer uses a persistent volume, the ODH operator deploys Tempo in monolithic mode (appropriate for smaller deployments). When the customer uses object storage, the ODH operator deploys Tempo in microservices mode (appropriate for larger deployments). If the customer changes storage, existing trace data is lost, and the ODH operator re-provisions Tempo.



The ODH operator defines an Instrumentation CR, which the OpenTelemetry operator uses to inject tracing configuration into component and customer workloads. The OpenTelemetry operator configures workloads that use the OpenTelemetry SDK to submit trace data to the central otel-collector. This also configures sampling rate (which may be necessary to tune in very large deployments).



Instrumentation can be accomplished in two different ways:

Integration of the OpenTelemetry SDK into application code.

Injection of instrumentation libraries via auto-instrumentation.



Components are encouraged to adopt the OpenTelemetry SDK natively (this involves using opentelemetry libraries or libraries that themselves integrate with the opentelemetry SDK), and this is the preferred integration pattern, since "the Red Hat build of OpenTelemetry Operator only supports the injection mechanism of the instrumentation libraries but does not support instrumentation libraries", and auto-instrumentation drastically alters the shape of workloads, including injection of an init-container, whereas native integration simply injects key environment variables.



The ODH operator configures the otel-collector to receive and process trace data, exporting it to the Tempo instance.



The customer can optionally configure the otel-collector to export traces to a 3rd party observability platform. Support is according to the Red Hat Build of OpenTelemetry support policy; as of 4.18:

GA: otlp

Tech preview: kafka, awsxray, file

Logs

Initially, OpenShift AI documentation recommends the use of the Cluster Logging Operator for customers with log retention needs. Future work will provide out-of-the-box configuration and integration with the Cluster Logging Operator.



Using the Cluster Logging Operator, customers can store logs using Loki and forward logs to external 3rd party observability platforms, including cloudwatch, elasticsearch, kafka, and splunk.

Visualizations \& Dashboards

The ODH operator defines a central Perses instance (managed via the Cluster Observability Operator), configured to query both metrics (via Thanos/Prometheus) and traces (via Tempo). Components can utilize Perses to build re-usable dashboards, and customers can build custom dashboards using Perses. Note: Perses does not support logs yet, but plans to in a future release.



Because OpenShift AI uses a UI that is completely separate from OpenShift's console, and to give users observability without needing to have OpenShift console access, OpenShift AI provides its own Perses UI, separate from the OpenShift Console UIPlugin integration introduced in the Cluster Observability Operator.



Component teams provide dashboards using the PersesDashboard CR, which allows them to be referenced both in the component's UI as well as globally across the cluster. The OpenShift AI Platform team will provide guidelines (including best practices and recommendations), but the work to implement dashboards will be done by individual component teams.



OpenShift AI will re-use the styling that OpenShift Container Platform uses for the Perses integration.



Perses UI (upstream):



Perses dashboards in OpenShift:



Perses traces in OpenShift:



Security Considerations



Pod-to-pod network communication

Scraping protocol (http or https) is controlled by components. Enabling TLS across all scraping endpoints is likely to be an ongoing effort. The relative risk depends on the sensitivity of the metrics data, and while metrics are typically not considered sensitive, the OpenShift AI Platform team recommends following the best practices for scraping defined in the Red Hat Monitoring Group Handbook.

Prometheus gathering metrics from otel-collector \& trace data pushed to the otel-collector - the otel-collector is configured using the generated serving certificate for the headless service.

The otel-collector pushes of traces to Tempo uses the Tempo gateway service, which uses TLS.

Data at rest

Observability data is stored in two possible types of storage:

Persistent volumes - where the storage configuration of the underlying cluster may provide encryption at rest.

Object storage provided by the customer - where the object storage provider may provide encryption at rest.

Retention

Metrics will be retained for 90 days by default, with configurable retention.

Traces will be retained for 2 days (this is the Tempo default).

Authentication

The otel-collector scrapes of workload metrics are component controlled, and should utilize mTLS via a service account for authentication/authorization.

Prometheus' requests to gather metrics from otel-collector utilizes mTLS via a service account for authentication/authorization.

Components pushing traces to the otel-collector gateway are unauthenticated.

The otel-collector connection to Tempo is authenticated via ServiceAccount token.

The thanos querier endpoint is protected by kube-rbac-proxy, which supports tokens and client certificates.

Authorization

Scraping access is primarily controlled by NetworkPolicy. Each component creates a network policy allowing the "redhat-ods-monitoring" monitoring namespace to scrape all ports used in workload metrics endpoints.

Prometheus is authorized to gather metrics from the otel-collector deployment by being in the same namespace.

Components and user workloads' abilities to send traces is controlled by NetworkPolicy. All workloads on a cluster will be able to send traces by default.

The ODH operator explicitly authorizes otel-collector to write traces to Tempo via a ClusterRole.

The thanos querier endpoint is protected by kube-rbac-proxy with two endpoints, one that restricts metrics access to namespaces that the user has access to (for general OpenShift AI Users), and the other requiring an elevated role (for the OpenShift AI Cluster Admin persona).

Quality Assurance Considerations

We have a bunch of tests about Monitoring in the ods-ci repo, that are currently running against a RHOAI installation in a managed cluster as part of the nightlies and as part of the RHOAI platform signoffs. Given that in the new architecture metrics will still be published to the Prometheus instance, we just need to adjust the test suite to query the new instance.

Operator dependencies (Cluster Observability Operator. OpenTelemetry Operator, Tempo Operator for Phase 1) need to be added as new dependent operators for all the ODH/RHOAI deployments, including proper mirroring in Disconnected environments.

Test in Self managed environments through ODH/RHOAI nightlies (running the Monitoring test-expression in the ods-ci repository)

Test in Managed environments through RHOAI nightlies (ODH is not supported in managed environments) (running the Monitoring test-expression in the ods-ci repository)

Test in Disconnected environment through ODH/RHOAI nightlies (running the Monitoring test-expression in the ods-ci repository)

Tracing and Logging test suites will need to be implemented to verify the propagation to Tempo, Loki respectively.

Managed Service Considerations

Service impact(s) that SRE will need to consider or be aware of 

RHODS - MT-SRE Acceptance Criteria Checklist

Metrics must be in place for SRE

Example SOP.

Check with the SRE team and AI services managers if in doubt.

Resource Considerations

Include the PSAP team in the review if there are resource considerations.

Resources consumed by the feature (deviations from defaults are in bold)

Prometheus

2 replicas

Requests (each)

110m CPU

306Mi memory

Limits (each)

510m CPU

522Mi memory

Storage (each)

5Gi PVC

Alertmanager

2 replicas

Requests (each)

CPU - 100m

Memory - 250Mi

Limits (each)

CPU - 1

Memory  - 250Mi

Thanos Querier

1 replica

Requests (each)

CPU - 100m

Memory - 64Mi

Limits (each)

CPU - 1

Memory - 64Mi

Tempo

1 replica

Requests (each)

CPU - 100m

Memory - 256Mi

Limits (each)

CPU - 1

Memory - 256Mi

Otel-collector

2 replicas

Requests (each)

CPU - 100m

Memory - 256Mi

Limits (each)

CPU - 1

Memory - 256Mi

kube-rbac-proxy and prom-label-proxy resource usage expected to be minimal (will scale alongside thanosquerier replicas)

What performance testing will be performed?

We will seek guidance from the observability team around what prior load testing has already been done for the components we're using. There should be no need to perform additional testing.

Documentation Considerations

What information needs to be passed to the user to inform them about this change?

Why should users use the new out-of-the-box monitoring capabilities of OpenShift AI?

OpenShift Cluster Administrators and OpenShift AI Cluster Admins can monitor resource utilization for storage and compute (including accelerators; i.e. GPU utilization).

Data Scientists and MLOps Engineers can see metrics for the performance of a deployed model.

AI Application Developers can see metrics and traces for their AI-enabled applications.

Users need to understand how to configure the feature, especially:

How to opt-in to adoption metrics

How to manage metrics \& traces access

How to configure storage size

Users need to be directed to the appropriate documentation for the individual components, especially:

Which exporters are supported for 3rd party integrations

Large scale users may need to scale individual components

NOTE: per-component monitoring changes/documentation (e.g. RHOAIRFE-572) is out-of-scope for this STRAT, and will be handled in related follow-up work.

What user tasks change as a result of this proposal, and how?

Administrators will need to install additional operators before configuring OpenShift AI

Users are able to create custom dashboards to observe OpenShift AI

There is now a new option that doesn't require UWM, the following docs need updates to make users aware of the new option:

Chapter 2. Serving models on the single-model serving platform | Serving models | Red Hat OpenShift AI Self-Managed 

Alternatives Considered / Rejected

Grafana

Perses is the solution chosen by the Openshift Observability team (OBSDA-390) to replace Grafana as the supported visualisation tool for metrics in OpenShift (in the long term). It's being integrated into the Cluster Observability Operator. We chose to use Perses to align.



Additionally, nothing in this proposal discourages the use of Grafana as a supported 3rd party tool.

otel-collector sidecar

This option requires an additional sidecar on every workload pod in order to collect traces. We'll use the gateway pattern instead to minimize resource usage.

Prometheus scraping

Having the central Prometheus perform the scraping does not allow us to take advantage of centralized processing of telemetry via the otel-collector. While there is not currently any required processing that can't be accomplished with Prometheus scraping, consistently using the otel-collector to process metrics gives us a single place for future work to add attributes to the signals (e.g., tenancy attributes). Additionally, with some components, specifically LlamaStack deployments, the metrics are sent via OTLP anyways.



When prometheus does the scraping, it requires the export of 3rd party metrics to scrape Prometheus periodically.

prometheusremotewrite

We explored using the prometheusremotewrite exporter during PoC, but its configuration was complex since it involved multiple writers and multiple destinations (2 otel-collector replicas to 2 prometheus replicas).

In-cluster monitoring

This is inflexible, can't be scaled, and is not officially supported. Product managers expressed concerns over risks of overloading the in-cluster monitoring stack.

User Workload Monitoring

We cannot be sure User Workload Monitoring is enabled.

We can’t enforce our configuration.

We can’t guarantee our configuration does not interfere with the customer configuration; an OpenShift Cluster Administrator or User on a shared cluster may set up and customize User Workload Monitoring for different purposes (e.g. non-AI applications).

Kubernetes Service

Using a kubernetes service to query the Prometheus replicas is feasible, but it makes the results inconsistent, as there are slight deviations in query results depending on the replica that services the query.

Authenticated Metrics Scrapes

Strategies supported by Prometheus itself, as well as otel-collector are:

Basic auth

Bearer token

OAuth2

mTLS

Prometheus sending to Observatorium

Having Prometheus send adoption metrics to Observatorium would remove a network hop; see details here: https://rhobs-handbook.netlify.app/services/rhobs/configuring-clients-for-observatorium.md/#using-a-self-managed-prometheus-server 



However, using otel-collector to send adoption metrics centralizes processing of the metrics, giving us a place to filter and transform metrics as needed.

Challenges

Prometheus is optimized to scrape many targets rather than a single target with many samples. We do not have insights on performance of Prometheus federation, and choosing to use federation to have Prometheus ingest metrics from the otel-collector could have scalability issues.

Very large deployments may require manually scaling the Prometheus instance, OpenTelemetry collector deployment, or other deployments.

Future work in the Observability components may address this challenge (e.g. work-in-progress autoscaling for Prometheus).

Scaling brings non-trivial considerations (see e.g. Scaling the Collector | OpenTelemetry )

Auto-instrumentation may not be viable for every workload; we may have to manually configure some workload.

Authentication could be hardened over time.

Dependencies

Dependent products

OpenShift Container Platform

Image: openshift4/ose-prom-label-proxy

Image: openshift4/ose-kube-rbac-proxy

Cluster Observability Operator

Tempo Operator

Red Hat build of OpenTelemetry

The centralized monitoring capabilities are intended to make the use of User Workload Monitoring unnecessary. However, we will not actively prevent User Workload Monitoring from working. Components using user workload monitoring should plan to migrate to using new monitoring services as soon as feasible.

New feature/component considerations

Code will land in the existing operator codebase.

Because OpenShift AI will be dependent on the other operators (Tempo, OpenTelemetry, Cluster Observability Operator), OpenShift AI customers will be dependent on those Operators for security updates when necessary.

Disconnected installations should mirror these dependent operators if the user wishes to use the out-of-the-box observability.

References/Resources

Observability primer | OpenTelemetry 

Miro - OpenShift AI Observability Architecture

RHOAI Observability Jiras

Cluster Observability Operator | Red Hat Product Documentation

Distributed tracing | Red Hat Product Documentation

Red Hat build of OpenTelemetry | Red Hat Product Documentation 

Auto-instrumentation in the Red Hat build of OpenTelemetry Operator 

Logging | OpenShift Container Platform | 4.18 | Red Hat Documentation 

Review Status

Expected life cycle for each ADD will be as follows, no ADD once moved past Draft can be ‘deleted’ or ‘removed’.  : 

Draft: An ADD in Draft indicates its a concept under consideration, being scoped, being written up. No action is expected from anyone other than the author, and there are no time pressures towards moving from the draft state.

Proposed : ADDs can move to the proposed state once a sponsor has signed up, and the ADD can then start to be socialized within the team / with PMs and with RFCs / RFEs can be scoped in.

Reviewing : Once moved to Reviewing state, the ADD must be brought to the SIG-SRE weekly call for consideration based on feedback already received. An ADD from this stage can either be moved back to the Proposed state, if gaps are identified or conversations need to be closed, or its Accepted / Rejected. And actions as follow-up should be executed.

Accepted : Once an ADD is ‘accepted’, it must be moved into read-only mode and no further changes are allowed.

Rejected : From the reviewing state, it's possible to move ADDs to a rejected state if they do not align with goals, or if stakeholders have fundamental concerns.

Superseded : ADDs once they move to Accepted state can no longer be changed, since goals and execution is established. Changes would be implemented by replacement ADDs with the previous ADD being moved to ‘superseded’ state, and a comment being added into the ADD metadata itself indicating which ADD now supersedes this specific one. The general pattern to follow is the IETF RFC flow.


Red Hat Monitoring Group Handbook

Handbook

Proposals

Blog #monitoring

 Search this site…

 Search this site…

Handbook

Contributing

Products

Observability Operator

Openshift Cluster Monitoring

Instrumentation guidelines

Collecting metrics with Prometheus

Alerting

Dashboards

Sending metrics via Telemetry

Frequently asked questions

Projects

Go

go-grpc-middleware

prometheus/client\_golang

Observability

Kube State Metrics

kube-rbac-proxy

Observatorium

Prometheus

Prometheus Operator

Thanos

prometheus-api-client-python

Proposals

Accepted

\## Evolution of the Observability Operator (fka MSO)

2021-06: Handbook

Done

2021-06: Proposal Process

Proposals Process

Rejected

Services

Red Hat Observability Service

Analytics based on Observability Data

Configuring Clients for Red Hat’s Observatorium Instance

Consumption Billing

Rules and alerting capabilities

Use Cases

MST

Telemetry (Telemeter)

Team

Observability Platform

Observability Platform Team SRE Processes

Team member onboarding

&nbsp;Print entire section

Targeted audience

Background

Requirements

Sending metrics via Telemetry step-by-step

Request approval

Configure recording rules

Modify the Telemeter client’s configuration

Synchronize the Telemeter server’s configuration

Frequently asked questions (FAQ)

What is the cardinality of a metric?

Why is there a limit on the number of metrics that can be collected?

Will Telemetry automatically collect alerts?

How do I ship metrics via Telemetry for older OCP releases?

How do I get access to Telemetry?

How do I instrument my component for Prometheus?

How does the in-cluster monitoring stack scrape metrics from my component?

How is the communication secured between the Telemeter client and server?

Glossary

Telemetry

Telemeter client

In-cluster monitoring stack

Products

Openshift Cluster Monitoring

Sending metrics via Telemetry

Sending metrics via Telemetry

&nbsp;9 minute read



Sending metrics via Telemetry

Targeted audience

This document is intended for OpenShift developers that want to ship new metrics to the Red Hat Telemetry service.



Background

Before going to the details, a few words about Telemetry and the process to add a new metric..



What is Telemetry?



Telemetry is a system operated and hosted by Red Hat that allows to collect data from connected clusters to enable subscription management automation, monitor the health of clusters, assist with support, and improve customer experience.



What does sending metrics via Telemetry mean?



You should send the metrics via Telemetry when you want and need to see these metrics for all OpenShift clusters. This is primarily for gaining insights on how OpenShift is used, troubleshooting and monitoring the fleet of clusters. Users can already see these metrics in their clusters via Prometheus even when not available via Telemetry.



How are metrics shipped via Telemetry?



Only metrics which are already collected by the in-cluster monitoring stack can be shipped via Telemetry. The telemeter-client pod running in the openshift-monitoring namespace collects metrics from the prometheus-k8s service every 4m30s using the /federate endpoint and ships the samples to the Telemetry endpoint using a custom protocol.



How long will it take for my new telemetry metrics to show up?



Please start this process and involve the monitoring team as early as possible. The process described in this document includes a thorough review of the underlying metrics and labels. The monitoring team will try to understand your use case and perhaps propose improvements and optimizations. Metric, label and rule names will be reviewed for following best practices. This can take several review rounds over multiple weeks.



Requirements

Shipping metrics via Telemetry is only possible for components running in namespaces with the openshift.io/cluster-monitoring=true label. In practice, it means that your component falls into one of these 2 categories:



Your operator/operand is included in the OCP payload (e.g. it is a core/platform component).

Your operator/operand is deployed via OLM and has been certified by Red Hat.

Your component should already be instrumented and scraped by the in-cluster monitoring stack using ServiceMonitor and/or PodMonitor objects.



Sending metrics via Telemetry step-by-step

The overall process is as follows:



Request approval from the monitoring team.

Configure recording rules using PrometheusRule objects.

Modify the configuration of the Telemeter client in the Cluster Monitoring Operator repository to collect the new metrics.

Synchronize the Telemeter server’s configuration from the Cluster Monitoring Operator project.

Wait for the Telemeter server’s configuration to be rolled out to production.

Request approval

The first step is to identify which metrics you want to send via Telemetry and what is the cardinality of the metrics (e.g. how many timeseries it will be in total). Typically you start with metrics that show how your component is being used. In practice, we recommend to start shipping not more than:



1 to 3 metrics.

1 to 10 timeseries per metric.

10 timeseries in total.

If you are above these limits, you have 2 choices:



(recommended) aggregate the metrics before sending. For instance: sum all values for a given metric.

request an exception from the monitoring team. The exception requires approval from upper management so make sure that your request is motivated!

Finally your metric MUST NOT contain any personally identifiable information (names, email addresses, information about user workloads).



Use the following information to file 1 JIRA ticket per metric in the MON project:



Type: Task

Title: Send metric <metric name> via Telemetry

Label: telemetry-review-request

Description template:

h1. Request for sending data via telemetry



The goal is to collect metrics about ... because ...



<Metric name> represents ...



Labels

\* <label 1>, possible values are ...

\* <label 2>, possible values are ...



The cardinality of the metric is at most <X>.



Component exposing the metric: https://github.com/<org>/<project>

Reach out to @team-telemetry on the #forum-openshift-monitoring or #forum-observatorium Slack channels for an explicit approval (e.g. in-cluster and RHOBS team leads).



Configure recording rules

Recording rules are required to reduce the cardinality of the metrics being shipped.



Even for low-cardinality metrics, we require to aggregate them before shipping to Telemetry to remove unnecessary labels such as instance or pod. This will also protect the telemetry backend against future label additions to the underlying metrics.



Let’s take a concrete example: each Prometheus pod exposes a prometheus\_tsdb\_head\_series metric which tracks the number of active timeseries. There can be up to 4 Prometheus pods in a given cluster (2 pods in openshift-monitoring and 2 in openshift-user-workload-monitoring when user-defined monitoring is enabled). To reduce the number of timeseries shipped via Telemetry, we configure the following recording rule to sum the values by namespace and job labels:



apiVersion: monitoring.coreos.com/v1

kind: PrometheusRule

apiVersion: monitoring.coreos.com/v1

kind: PrometheusRule

metadata:

&nbsp; name: cluster-monitoring-operator-prometheus-rules

&nbsp; namespace: openshift-monitoring

spec:

&nbsp; groups:

&nbsp; - name: openshift-monitoring.rules

&nbsp;   rules:

&nbsp;   - expr: |-

&nbsp;       sum by (job,namespace) (

&nbsp;         max without(instance) (

&nbsp;           prometheus\_tsdb\_head\_series{namespace=~"openshift-monitoring|openshift-user-workload-monitoring"}

&nbsp;         )

&nbsp;       )

&nbsp;     record: openshift:prometheus\_tsdb\_head\_series:sum

Your PrometheusRule object(s) should be created by your operator with your ServiceMonitor and/or PodMonitor objects.



Modify the Telemeter client’s configuration

Clone the cluster-monitoring-operator repository locally.



Modify the /manifests/0000\_50\_cluster-monitoring-operator\_04-config.yaml file to add the metric to the allowed list. Include comments to:



Identify the team owning the metric.

Provide a short description.

(optional) Indicate which team(s) will consume the metric, it helps knowing who to contact if changes are made in the future.

&nbsp;   #

&nbsp;   # owners: (@openshift/openshift-team-monitoring)

&nbsp;   #

&nbsp;   # openshift:prometheus\_tsdb\_head\_series:sum tracks the total number of active series

&nbsp;   - '{\_\_name\_\_="openshift:prometheus\_tsdb\_head\_series:sum"}'

Run

make --always-make docs

Commit the changes into Git and open a pull request in the openshift/cluster-monitoring-operator repository linking to the initial JIRA ticket.



Ask for a review on the #forum-monitoring Slack channel.



Synchronize the Telemeter server’s configuration

Once the pull request in the cluster-monitoring-operator repository is merged, the configuration of the Telemetry server needs to be synchronized.



Clone the rhobs/configuration repository.



Run



make whitelisted\_metrics \&\& make

Commit the changes into Git and open a pull request in the rhobs/configuration repository.



Ask for a review on the #forum-observatorium Slack channel.



Once merged, the updated configuration should be rolled out to the production Telemetry within a few days. After this happens, clusters running the next (e.g. master) OCP version should start sending the new metric(s) to Telemetry.



Frequently asked questions (FAQ)

What is the cardinality of a metric?

A given metric may have different labels (aka dimensions) that helps refining the characteristics of the thing being measured. Each unique combination of a metric name + optional key/value pairs represents a timeseries in the Prometheus parlance. And the total number of active timeseries for a given metric name represents the cardinality of the metric.



For example, consider a component exposing a fictuous my\_component\_ready metric:



my\_component\_ready 1

The metric has no label but because Prometheus will automatically attach target labels such as pod and instance, the total cardinality could be 1 (single replica), 2 (2 replicas), …



To find out the current cardinality of a metric on a live cluster, you can run this PromQL query:



count(my\_component\_ready)

Now consider another metric tracking HTTP requests:



http\_requests\_total{method="GET", code="200", path="/"} 10

http\_requests\_total{method="GET", code="404", path="/foo"} 1

http\_requests\_total{method="POST", code="200", path="/"} 12

http\_requests\_total{method="POST", code="500", path="/login"} 2

While you may think that the cardinality is 4 because there are 4 timeseries, this isn’t true because we can’t really predict in advance all values for the code and path labels. This is what is called a high-cardinality metric. An even worse case would be a metric with a userid or ip label (we would say that this metric has unbounded cardinality).



On top of that, pod churn (e.g. pods being rolled-out because of version upgrades) also increase the cardinality because the values of target-based labels (such pod and instance) would change.



Because Prometheus keeps all active timeseries in-memory for indexing, the more timeseries, the more memory is required. The same is true for the Telemeter server. Which is why we want to keep the cardinality of metrics shipped via Telemetry under a reasonable value (typically less than 5).



Why is there a limit on the number of metrics that can be collected?

See the previous section. Every metric shipped to Telemetry has to be multiplied by the number of connected clusters that may be sending that metric. Pushing too many metrics from a single cluster may cause service degradation and resource exhaustion on both the in-cluster monitoring stack and on the Telemetry server side.



Will Telemetry automatically collect alerts?

Yes, the Telemeter client is already configured to collect and send firing alerts. On Telemetry side, the alerts can be queried using the alerts metric.



How do I ship metrics via Telemetry for older OCP releases?

Once you have updated the telemeter-client configuration in the master branch, you can create backports to older OCP releases. The procedure follows the usual OCP backport process which involves creating bug tickets in the OCPBUGS project (preferably assigned to your component) and opening pull requests in openshift/cluster-monitoring-operator against the desired release-4.x branches.



How do I get access to Telemetry?

Check https://gitlab.cee.redhat.com/data-hub/dh-docs/-/blob/master/docs/interacting-with-telemetry-data.adoc



How do I instrument my component for Prometheus?

Please refer to the following links for more details:



Metric and label naming (upstream Prometheus documentation).

Instrumentation (upstream Prometheus documentation).

Instrumenting Kubernetes

You can also reach out to the OpenShift monitoring team for advice.



How does the in-cluster monitoring stack scrape metrics from my component?

If your component’s metrics aren’t already collected by the in-cluster monitoring stack, you need to deploy at least one ServiceMonitor or one PodMonitor resource in your component’s namespace.



If your component is deployed by the Cluster Version Operator (CVO), it is enough to add the manifest to the CVO payload.



Again you can reach out to the OpenShift monitoring team for advice.



How is the communication secured between the Telemeter client and server?

The Telemetry client authenticates against the Telemeter server using the cluster’s pull secret. The Telemeter server verifies that the pull secret is valid and matches with the cluster’s identifier. The Telemetry protocol uses HTTPS for encryption.



Finally the Telemeter server will only allow metrics which are explicitly allowed by its running configuraiton.



Glossary

Telemetry

Also know as Telemetry or Telemeter server. A service operated by Red Hat that receives metrics from all OCP connected clusters.



Telemeter client

The telemeter-client pod runnning in the openshift-monitoring namespace. It is responsible for collecting the platform metrics at regular intervals (every 4m30s) and sending them to the Telemetry server.



In-cluster monitoring stack

The prometheus-k8s-0 and prometheus-k8s-1 pods running in the openshift-monitoring namespace. They are in charge of collecting metrics from the OpenShift components (operators+operands) and evaluating the associated alerting and recording rules. The Prometheus pods are configured using ServiceMonitor, PodMonitor and PrometheusRule custom resources coming from namespaces with the openshift.io/cluster-monitoring=true label.



Last modified May 9, 2025

© 2025 The Red Hat Monitoring Group Authors All Rights Reserved


# Custom Resource State Metrics



This section describes how to add metrics based on the state of a custom resource without writing a custom resource

registry and running your own build of KSM.



\## Configuration



A YAML configuration file described below is required to define your custom resources and the fields to turn into metrics.



Two flags can be used:



\* `--custom-resource-state-config "inline yaml (see example)"` or

\* `--custom-resource-state-config-file /path/to/config.yaml`



If both flags are provided, the inline configuration will take precedence.

When multiple entries for the same resource exist, kube-state-metrics will exit with an error.

This includes configuration which refers to a different API version.



```yaml

apiVersion: apps/v1

kind: Deployment

metadata:

&nbsp; name: kube-state-metrics

&nbsp; namespace: kube-system

spec:

&nbsp; template:

&nbsp;   spec:

&nbsp;     containers:

&nbsp;     - name: kube-state-metrics

&nbsp;       args:

&nbsp;         - --custom-resource-state-config

&nbsp;         # in YAML files, | allows a multi-line string to be passed as a flag value

&nbsp;         # see https://yaml-multiline.info

&nbsp;         -  |

&nbsp;             kind: CustomResourceStateMetrics

&nbsp;             spec:

&nbsp;               resources:

&nbsp;                 - groupVersionKind:

&nbsp;                     group: myteam.io

&nbsp;                     version: "v1"

&nbsp;                     kind: Foo

&nbsp;                   metrics:

&nbsp;                     - name: active\_count

&nbsp;                       help: "Count of active Foo"

&nbsp;                       each:

&nbsp;                         type: Gauge

&nbsp;                         ...

```



It's also possible to configure kube-state-metrics to run in a `custom-resource-mode` only. In addition to specifying one of `--custom-resource-state-config\*` flags, you could set `--custom-resource-state-only` to `true`.

With this configuration only the known custom resources configured in `--custom-resource-state-config\*` will be taken into account by kube-state-metrics.



```yaml

apiVersion: apps/v1

kind: Deployment

metadata:

&nbsp; name: kube-state-metrics

&nbsp; namespace: kube-system

spec:

&nbsp; template:

&nbsp;   spec:

&nbsp;     containers:

&nbsp;     - name: kube-state-metrics

&nbsp;       args:

&nbsp;         - --custom-resource-state-config

&nbsp;         # in YAML files, | allows a multi-line string to be passed as a flag value

&nbsp;         # see https://yaml-multiline.info

&nbsp;         -  |

&nbsp;             kind: CustomResourceStateMetrics

&nbsp;             spec:

&nbsp;               resources:

&nbsp;                 - groupVersionKind:

&nbsp;                     group: myteam.io

&nbsp;                     version: "v1"

&nbsp;                     kind: Foo

&nbsp;                   metrics:

&nbsp;                     - name: active\_count

&nbsp;                       help: "Count of active Foo"

&nbsp;                       each:

&nbsp;                         type: Gauge

&nbsp;                         ...

&nbsp;         - --custom-resource-state-only=true

```



NOTE: The `customresource\_group`, `customresource\_version`, and `customresource\_kind` common labels are reserved, and will be overwritten by the values from the `groupVersionKind` field.



\### RBAC-enabled Clusters



Please be aware that kube-state-metrics needs list and watch permissions granted to `customresourcedefinitions.apiextensions.k8s.io` as well as to the resources you want to gather metrics from.



\### Examples



The examples in this section will use the following custom resource:



```yaml

kind: Foo

apiVersion: myteam.io/vl

metadata:

&nbsp;   annotations:

&nbsp;       bar: baz

&nbsp;       qux: quxx

&nbsp;   labels:

&nbsp;       foo: bar

&nbsp;   name: foo

spec:

&nbsp;   version: v1.2.3

&nbsp;   order:

&nbsp;       - id: 1

&nbsp;         value: true

&nbsp;       - id: 3

&nbsp;         value: false

&nbsp;   replicas: 1

&nbsp;   refs:

&nbsp;       - my\_other\_foo

&nbsp;       - foo\_2

&nbsp;       - foo\_with\_extensions

status:

&nbsp;   phase: Pending

&nbsp;   active:

&nbsp;       type-a: 1

&nbsp;       type-b: 3

&nbsp;   conditions:

&nbsp;       - name: a

&nbsp;         value: 45

&nbsp;       - name: b

&nbsp;         value: 66

&nbsp;   sub:

&nbsp;       type-a:

&nbsp;           active: 1

&nbsp;           ready: 2

&nbsp;       type-b:

&nbsp;           active: 3

&nbsp;           ready: 4

&nbsp;   uptime: 43.21

```



\#### Single Values



The config:



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: myteam.io

&nbsp;       kind: "Foo"

&nbsp;       version: "v1"

&nbsp;     metrics:

&nbsp;       - name: "uptime"

&nbsp;         help: "Foo uptime"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, uptime]

```



Produces the metric:



```prometheus

kube\_customresource\_uptime{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1"} 43.21

```



\#### Multiple Metrics/Kitchen Sink



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: myteam.io

&nbsp;       kind: "Foo"

&nbsp;       version: "v1"

&nbsp;     # labels can be added to all metrics from a resource

&nbsp;     commonLabels:

&nbsp;       crd\_type: "foo"

&nbsp;     labelsFromPath:

&nbsp;       name: \[metadata, name]

&nbsp;     metrics:

&nbsp;       - name: "ready\_count"

&nbsp;         help: "Number Foo Bars ready"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             # targeting an object or array will produce a metric for each element

&nbsp;             # labelsFromPath and value are relative to this path

&nbsp;             path: \[status, sub]



&nbsp;             # if path targets an object, the object key will be used as label value

&nbsp;             # This is not supported for StateSet type as all values will be truthy, which is redundant.

&nbsp;             labelFromKey: type

&nbsp;             # label values can be resolved specific to this path 

&nbsp;             labelsFromPath:

&nbsp;               active: \[active]

&nbsp;             # The actual field to use as metric value. Should be a number, boolean or RFC3339 timestamp string.

&nbsp;             valueFrom: \[ready]

&nbsp;         commonLabels:

&nbsp;           custom\_metric: "yes"

&nbsp;         labelsFromPath:

&nbsp;           # whole objects may be copied into labels by prefixing with "\*"

&nbsp;           # \*anything will be copied into labels, with the highest sorted \* strings first

&nbsp;           "\*": \[metadata, labels]

&nbsp;           # a prefix before the asterisk will be used as a label prefix

&nbsp;           "lorem\_\*": \[metadata, annotations]

&nbsp;           "\*\*": \[metadata, annotations]

&nbsp;           

&nbsp;           # or specific fields may be copied. these fields will always override values from \*s

&nbsp;           name: \[metadata, name]

&nbsp;           foo: \[metadata, labels, foo]

```



Produces the following metrics:



```prometheus

kube\_customresource\_ready\_count{customresource\_group="myteam.io", customresource\_kind="Foo", 

customresource\_version="v1", active="1",custom\_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-a",

lorem\_bar="baz",lorem\_qux="quxx",} 2

kube\_customresource\_ready\_count{customresource\_group="myteam.io", customresource\_kind="Foo", 

customresource\_version="v1", active="3",custom\_metric="yes",foo="bar",name="foo",bar="baz",qux="quxx",type="type-b",

lorem\_bar="baz",lorem\_qux="quxx",} 4

```



\#### Non-map Arrays



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: myteam.io

&nbsp;       kind: "Foo"

&nbsp;       version: "v1"

&nbsp;     labelsFromPath:

&nbsp;       name: \[metadata, name]

&nbsp;     metrics:

&nbsp;       - name: "ref\_info"

&nbsp;         help: "Reference to other Foo"

&nbsp;         each:

&nbsp;           type: Info

&nbsp;           info:

&nbsp;             # targeting an array will produce a metric for each element

&nbsp;             # labelsFromPath and value are relative to this path

&nbsp;             path: \[spec, refs]



&nbsp;             # if path targets a list of values (e.g. strings or numbers, not objects or maps), individual values can

&nbsp;             # referenced by a label using this syntax

&nbsp;             labelsFromPath:

&nbsp;               ref: \[]

```



Produces the following metrics:



```prometheus

kube\_customresource\_ref\_info{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", name="foo",ref="my\_other\_foo"} 1

kube\_customresource\_ref\_info{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", name="foo",ref="foo\_2"} 1

kube\_customresource\_ref\_info{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", name="foo",ref="foo\_with\_extensions"} 1

```



\#### Same Metrics with Different Labels



```yaml

&nbsp; recommendation:

&nbsp;   containerRecommendations:

&nbsp;   - containerName: consumer

&nbsp;     lowerBound:

&nbsp;       cpu: 100m

&nbsp;       memory: 262144k

```



For example in VPA we have above attributes and we want to have a same metrics for both CPU and Memory, you can use below config:



```

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: autoscaling.k8s.io

&nbsp;       kind: "VerticalPodAutoscaler"

&nbsp;       version: "v1"

&nbsp;     labelsFromPath:

&nbsp;       verticalpodautoscaler: \[metadata, name]

&nbsp;       namespace: \[metadata, namespace]

&nbsp;       target\_api\_version: \[apiVersion]

&nbsp;       target\_kind: \[spec, targetRef, kind]

&nbsp;       target\_name: \[spec, targetRef, name]

&nbsp;     metrics:

&nbsp;       # for memory

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound"

&nbsp;         help: "Minimum memory resources the container can use before the VerticalPodAutoscaler updater evicts it."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[lowerBound, memory]

&nbsp;       # for CPU

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound"

&nbsp;         help: "Minimum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[lowerBound, cpu]

```



Produces the following metrics:



```prometheus

\# HELP kube\_customresource\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound Minimum memory resources the container can use before the VerticalPodAutoscaler updater evicts it.

\# TYPE kube\_customresource\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound gauge

kube\_customresource\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound{container="consumer",customresource\_group="autoscaling.k8s.io",customresource\_kind="VerticalPodAutoscaler",customresource\_version="v1",namespace="namespace-example",resource="memory",target\_api\_version="apps/v1",target\_kind="Deployment",target\_name="target-name-example",unit="byte",verticalpodautoscaler="vpa-example"} 123456

\# HELP kube\_customresource\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound Minimum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it.

\# TYPE kube\_customresource\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound gauge

kube\_customresource\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound{container="consumer",customresource\_group="autoscaling.k8s.io",customresource\_kind="VerticalPodAutoscaler",customresource\_version="v1",namespace="namespace-example",resource="cpu",target\_api\_version="apps/v1",target\_kind="Deployment",target\_name="target-name-example",unit="core",verticalpodautoscaler="vpa-example"} 0.1

```



\#### VerticalPodAutoscaler



In v2.9.0 the `vericalpodautoscalers` resource was removed from the list of default resources. In order to generate metrics for `verticalpodautoscalers`, you can use the following Custom Resource State config:



```yaml

\# Using --resource=verticalpodautoscalers, we get the following output:

\# HELP kube\_verticalpodautoscaler\_annotations Kubernetes annotations converted to Prometheus labels.

\# TYPE kube\_verticalpodautoscaler\_annotations gauge

\# kube\_verticalpodautoscaler\_annotations{namespace="default",verticalpodautoscaler="hamster-vpa",target\_api\_version="apps/v1",target\_kind="Deployment",target\_name="hamster"} 1

\# A similar result can be achieved by specifying the following in --custom-resource-state-config:

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: autoscaling.k8s.io

&nbsp;       kind: "VerticalPodAutoscaler"

&nbsp;       version: "v1"

&nbsp;     labelsFromPath:

&nbsp;       verticalpodautoscaler: \[metadata, name]

&nbsp;       namespace: \[metadata, namespace]

&nbsp;       target\_api\_version: \[apiVersion]

&nbsp;       target\_kind: \[spec, targetRef, kind]

&nbsp;       target\_name: \[spec, targetRef, name]

&nbsp;     metrics:

&nbsp;       - name: "annotations"

&nbsp;         help: "Kubernetes annotations converted to Prometheus labels."

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[metadata, annotations]

\# This will output the following metric:

\# HELP kube\_customresource\_autoscaling\_annotations Kubernetes annotations converted to Prometheus labels.

\# TYPE kube\_customresource\_autoscaling\_annotations gauge

\# kube\_customresource\_autoscaling\_annotations{customresource\_group="autoscaling.k8s.io", customresource\_kind="VerticalPodAutoscaler", customresource\_version="v1", namespace="default",target\_api\_version="autoscaling.k8s.io/v1",target\_kind="Deployment",target\_name="hamster",verticalpodautoscaler="hamster-vpa"} 123

```



The above configuration was tested on \[this](https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/examples/hamster.yaml) VPA configuration, with an added annotation (`foo: 123`).



\#### All VerticalPodAutoscaler Metrics



As an addition for the above configuration, here's the complete `CustomResourceStateMetrics` spec to re-enable all of the VPA metrics which are removed from the list of the default resources:



<details>



&nbsp;<summary>VPA CustomResourceStateMetrics</summary>



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: autoscaling.k8s.io

&nbsp;       kind: "VerticalPodAutoscaler"

&nbsp;       version: "v1"

&nbsp;     labelsFromPath:

&nbsp;       namespace: \[metadata, namespace]

&nbsp;       target\_api\_version: \[spec, targetRef, apiVersion]

&nbsp;       target\_kind: \[spec, targetRef, kind]

&nbsp;       target\_name: \[spec, targetRef, name]

&nbsp;       verticalpodautoscaler: \[metadata, name]

&nbsp;     metricNamePrefix: "kube"

&nbsp;     metrics:

&nbsp;       # kube\_verticalpodautoscaler\_annotations

&nbsp;       - name: "verticalpodautoscaler\_annotations"

&nbsp;         help: "Kubernetes annotations converted to Prometheus labels."

&nbsp;         each:

&nbsp;           type: Info

&nbsp;           info:

&nbsp;             labelsFromPath:

&nbsp;               annotation\_\*: \[metadata, annotations]

&nbsp;               name: \[metadata, name]

&nbsp;       # kube\_verticalpodautoscaler\_labels

&nbsp;       - name: "verticalpodautoscaler\_labels"

&nbsp;         help: "Kubernetes labels converted to Prometheus labels."

&nbsp;         each:

&nbsp;           type: Info

&nbsp;           info:

&nbsp;             labelsFromPath:

&nbsp;               label\_\*: \[metadata, labels]

&nbsp;               name: \[metadata, name]

&nbsp;       # kube\_verticalpodautoscaler\_spec\_updatepolicy\_updatemode

&nbsp;       - name: "verticalpodautoscaler\_spec\_updatepolicy\_updatemode"

&nbsp;         help: "Update mode of the VerticalPodAutoscaler."

&nbsp;         each:

&nbsp;           type: StateSet

&nbsp;           stateSet:

&nbsp;             labelName: "update\_mode"

&nbsp;             path: \[spec, updatePolicy, updateMode]

&nbsp;             list: \["Auto", "Initial", "Off", "Recreate"]

&nbsp;       # Memory kube\_verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_minallowed\_memory

&nbsp;       - name: "verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_minallowed\_memory"

&nbsp;         help: "Minimum memory resources the VerticalPodAutoscaler can set for containers matching the name."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[spec, resourcePolicy, containerPolicies]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[minAllowed, memory]

&nbsp;       # CPU kube\_verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_minallowed\_cpu

&nbsp;       - name: "verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_minallowed\_cpu"

&nbsp;         help: "Minimum cpu resources the VerticalPodAutoscaler can set for containers matching the name."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[spec, resourcePolicy, containerPolicies]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[minAllowed, cpu]

&nbsp;       # Memory kube\_verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_maxallowed\_memory

&nbsp;       - name: "verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_maxallowed\_memory"

&nbsp;         help: "Maximum memory resources the VerticalPodAutoscaler can set for containers matching the name."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[spec, resourcePolicy, containerPolicies]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[maxAllowed, memory]

&nbsp;       # CPU kube\_verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_maxallowed\_cpu

&nbsp;       - name: "verticalpodautoscaler\_spec\_resourcepolicy\_container\_policies\_maxallowed\_cpu"

&nbsp;         help: "Maximum cpu resources the VerticalPodAutoscaler can set for containers matching the name."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[spec, resourcePolicy, containerPolicies]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[maxAllowed, cpu]

&nbsp;       # Memory kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound\_memory

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound\_memory"

&nbsp;         help: "Minimum memory resources the container can use before the VerticalPodAutoscaler updater evicts it."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[lowerBound, memory]

&nbsp;       # CPU kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound\_cpu

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_lowerbound\_cpu"

&nbsp;         help: "Minimum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[lowerBound, cpu]

&nbsp;       # Memory kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_upperbound\_memory

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_upperbound\_memory"

&nbsp;         help: "Maximum memory resources the container can use before the VerticalPodAutoscaler updater evicts it."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[upperBound, memory]

&nbsp;       # CPU kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_upperbound\_cpu

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_upperbound\_cpu"

&nbsp;         help: "Maximum cpu resources the container can use before the VerticalPodAutoscaler updater evicts it."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[upperBound, cpu]

&nbsp;       # Memory kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_target\_memory

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_target\_memory"

&nbsp;         help: "Target memory resources the VerticalPodAutoscaler recommends for the container."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[target, memory]

&nbsp;       # CPU kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_target\_cpu

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_target\_cpu"

&nbsp;         help: "Target cpu resources the VerticalPodAutoscaler recommends for the container."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[target, cpu]

&nbsp;       # Memory kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_uncappedtarget\_memory

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_uncappedtarget\_memory"

&nbsp;         help: "Target memory resources the VerticalPodAutoscaler recommends for the container ignoring bounds."

&nbsp;         commonLabels:

&nbsp;           unit: "byte"

&nbsp;           resource: "memory"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[uncappedTarget, memory]

&nbsp;       # CPU kube\_verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_uncappedtarget\_cpu

&nbsp;       - name: "verticalpodautoscaler\_status\_recommendation\_containerrecommendations\_uncappedtarget\_cpu"

&nbsp;         help: "Target memory resources the VerticalPodAutoscaler recommends for the container ignoring bounds."

&nbsp;         commonLabels:

&nbsp;           unit: "core"

&nbsp;           resource: "cpu"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, recommendation, containerRecommendations]

&nbsp;             labelsFromPath:

&nbsp;               container: \[containerName]

&nbsp;             valueFrom: \[uncappedTarget, cpu]

```



</details>



\### Metric types



The configuration supports three kind of metrics from the \[OpenMetrics specification](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md).



The metric type is specified by the `type` field and its specific configuration at the types specific struct.



\#### Gauge



> Gauges are current measurements, such as bytes of memory currently used or the number of items in a queue. For gauges the absolute value is what is of interest to a user. \[\[0]](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#gauge)



Example:



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: myteam.io

&nbsp;       kind: "Foo"

&nbsp;       version: "v1"

&nbsp;     metrics:

&nbsp;       - name: "uptime"

&nbsp;         help: "Foo uptime"

&nbsp;         each:

&nbsp;           type: Gauge

&nbsp;           gauge:

&nbsp;             path: \[status, uptime]

```



Produces the metric:



```prometheus

kube\_customresource\_uptime{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1"} 43.21

```



\##### Type conversion and special handling



Gauges produce values of type float64 but custom resources can be of all kinds of types.

Kube-state-metrics performs implicit type conversions for a lot of type.

Supported types are:



\* (u)int32/64, int, float32 and byte are cast to float64

\* `nil` is generally mapped to `0.0` if NilIsZero is `true`, otherwise it will throw an error

\* for bool `true` is mapped to `1.0` and `false` is mapped to `0.0`

\* for string the following logic applies

&nbsp; \* `"true"` and `"yes"` are mapped to `1.0`, `"false"`, `"no"` and `"unknown"` are mapped to `0.0` (all case-insensitive)

&nbsp; \* RFC3339 times are parsed to float timestamp  

&nbsp; \* Quantities like "250m" or "512Gi" are parsed to float using <https://github.com/kubernetes/apimachinery/blob/master/pkg/api/resource/quantity.go>

&nbsp; \* Percentages ending with a "%" are parsed to float

&nbsp; \* finally the string is parsed to float using <https://pkg.go.dev/strconv#ParseFloat> which should support all common number formats. If that fails an error is yielded



\##### Example for status conditions on Kubernetes Controllers



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp; - groupVersionKind:

&nbsp;     group: myteam.io

&nbsp;     kind: "Foo"

&nbsp;     version: "v1"

&nbsp;   labelsFromPath:

&nbsp;     name:

&nbsp;     - metadata

&nbsp;     - name

&nbsp;     namespace:

&nbsp;     - metadata

&nbsp;     - namespace

&nbsp;   metrics:

&nbsp;   - name: "foo\_status"

&nbsp;     help: "status condition "

&nbsp;     each:

&nbsp;       type: Gauge

&nbsp;       gauge:

&nbsp;         path: \[status, conditions]

&nbsp;         labelsFromPath:

&nbsp;           type: \["type"]

&nbsp;         valueFrom: \["status"]

```



This will work for kubernetes controller CRs which expose status conditions according to the kubernetes api (<https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition>):



```yaml

status:

&nbsp; conditions:

&nbsp;   - lastTransitionTime: "2019-10-22T16:29:31Z"

&nbsp;     status: "True"

&nbsp;     type: Ready

```



kube\_customresource\_foo\_status{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", type="Ready"} 1.0



\#### StateSet



> StateSets represent a series of related boolean values, also called a bitset. If ENUMs need to be encoded this MAY be done via StateSet. \[\[1]](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#stateset)



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: myteam.io

&nbsp;       kind: "Foo"

&nbsp;       version: "v1"

&nbsp;     metrics:

&nbsp;       - name: "status\_phase"

&nbsp;         help: "Foo status\_phase"

&nbsp;         each:

&nbsp;           type: StateSet

&nbsp;           stateSet:

&nbsp;             labelName: phase

&nbsp;             path: \[status, phase]

&nbsp;             list: \[Pending, Bar, Baz]

```



Metrics of type `StateSet` will generate a metric for each value defined in `list` for each resource.

The value will be 1, if the value matches the one in list.



Produces the metric:



```prometheus

kube\_customresource\_status\_phase{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", phase="Pending"} 1

kube\_customresource\_status\_phase{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", phase="Bar"} 0

kube\_customresource\_status\_phase{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", phase="Baz"} 0

```



\#### Info



> Info metrics are used to expose textual information which SHOULD NOT change during process lifetime. Common examples are an application's version, revision control commit, and the version of a compiler. \[\[2]](https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#info)



Metrics of type `Info` will always have a value of 1.



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: myteam.io

&nbsp;       kind: "Foo"

&nbsp;       version: "v1"

&nbsp;     metrics:

&nbsp;       - name: "version"

&nbsp;         help: "Foo version"

&nbsp;         each:

&nbsp;           type: Info

&nbsp;           info:

&nbsp;             labelsFromPath:

&nbsp;               version: \[spec, version]

```



Produces the metric:



```prometheus

kube\_customresource\_version{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1", version="v1.2.3"} 1

```



\### Naming



The default metric names are prefixed to avoid collisions with other metrics.

By default, a metric prefix of `kube\_` concatenated with your custom resource's group+version+kind is used.

You can override this behavior with the `metricNamePrefix` field.



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind: ...

&nbsp;     metricNamePrefix: myteam\_foos

&nbsp;     metrics:

&nbsp;       - name: uptime

&nbsp;         # ...

```



Produces:



```prometheus

myteam\_foos\_uptime{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1"} 43.21

```



To omit namespace and/or subsystem altogether, set them to the empty string:



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind: ...

&nbsp;     metricNamePrefix: ""

&nbsp;     metrics:

&nbsp;       - name: uptime

&nbsp;         # ...

```



Produces:



```prometheus

uptime{customresource\_group="myteam.io", customresource\_kind="Foo", customresource\_version="v1"} 43.21

```



\### Logging



If a metric path is registered but not found on a custom resource, an error will be logged. For some resources,

this may produce a lot of noise. The error log \[verbosity]\[vlog] for a metric or resource can be set with `errorLogV` on

the resource or metric:



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind: ...

&nbsp;     errorLogV: 0  # 0 = default for errors

&nbsp;     metrics:

&nbsp;       - name: uptime

&nbsp;         errorLogV: 10  # only log at high verbosity

```



\[vlog]: https://github.com/go-logr/logr#why-v-levels



\### Path Syntax



Paths are specified as a list of strings. Each string is a path segment, resolved dynamically against the data of the custom resource.

If any part of a path is missing, the result is nil.



Examples:



```yaml

\# simple path lookup

\[spec, replicas]                         # spec.replicas == 1



\# indexing an array

\[spec, order, "0", value]                # spec.order\[0].value = true



\# finding an element in a list by key=value  

\[status, conditions, "\[name=a]", value]  # status.conditions\[0].value = 45



\# if the value to be matched is a number or boolean, the value is compared as a number or boolean  

\[status, conditions, "\[value=66]", name]  # status.conditions\[1].name = "b"



\# For generally matching against a field in an object schema, use the following syntax:

\[metadata, "name=foo"] # if v, ok := metadata\[name]; ok \&\& v == "foo" { return v; } else { /\* ignore \*/ }

```



\### Wildcard matching of version and kind fields



The Custom Resource State (CRS hereon) configuration also allows you to monitor all versions and/or kinds that come under a group. It watches

the installed CRDs for this purpose. Taking the aforementioned `Foo` object as reference, the configuration below allows

you to monitor all objects under all versions \*and\* all kinds that come under the `myteam.io` group.



```yaml

kind: CustomResourceStateMetrics

spec:

&nbsp; resources:

&nbsp;   - groupVersionKind:

&nbsp;       group: "myteam.io"

&nbsp;       version: "\*" # Set to `v1 to monitor all kinds under `myteam.io/v1`. Wildcard matches all installed versions that come under this group.

&nbsp;       kind: "\*" # Set to `Foo` to monitor all `Foo` objects under the `myteam.io` group (under all versions). Wildcard matches all installed kinds that come under this group (and version, if specified).

&nbsp;     metrics:

&nbsp;       - name: "myobject\_info"

&nbsp;         help: "Foo Bar Baz"

&nbsp;         each:

&nbsp;           type: Info

&nbsp;           info:

&nbsp;             path: \[metadata]

&nbsp;             labelsFromPath:

&nbsp;               object: \[name]

&nbsp;               namespace: \[namespace]

```



The configuration above produces these metrics.



```yaml

kube\_customresource\_myobject\_info{customresource\_group="myteam.io",customresource\_kind="Foo",customresource\_version="v1",namespace="ns",object="foo"} 1

kube\_customresource\_myobject\_info{customresource\_group="myteam.io",customresource\_kind="Bar",customresource\_version="v1",namespace="ns",object="bar"} 1

```



\#### Note



\* For cases where the GVKs defined in a CRD have multiple versions under a single group for the same kind, as expected, the wildcard value will resolve to \*all\* versions, but a query for any specific version will return all resources under all versions, in that versions' representation. This basically means that for two such versions `A` and `B`,  if a resource exists under `B`, it will reflect in the metrics generated for `A` as well, in addition to any resources of itself, and vice-versa. This logic is based on the \[current `list`ing behavior](https://github.com/kubernetes/client-go/issues/1251#issuecomment-1544083071) of the client-go library.

\* The introduction of this feature further discourages (and discontinues) the use of native objects in the CRS featureset, since these do not have an explicit CRD associated with them, and conflict with internal stores defined specifically for such native resources. Please consider opening an issue or raising a PR if you'd like to expand on the current metric labelsets for them. Also, any such configuration will be ignored, and no metrics will be generated for the same.





RHOAISTRAT-575 - Platform Metrics and

Alerts for Self-Managed and Managed

OpenShift AICentralized

Authors: Kevin Howell Steven Tobin Cesar Francisco San Nicolas Martinez Marian Macik

Status: Reviewing

Approvers/Stakeholders:

Reviewed By Name(s) Date

Reviewed

Approved

Platform Lindani Phiri Jul 14, 2025 ✅

Serving Runtimes Vedant Mahabaleshwar…

JooHo Lee

Jul 31, 2025 ✅

Docs Jennifer Ciroli

Amy Duquette

LlamaStack Roland Huß

Vaishnavi Hire

Charlie Doern

Observability \& Insights Roy Nissim Jul 28, 20… ✅

Introduction/problem statement

Red Hat provides many building blocks that can be used to configure observability for any

workload running on OpenShift, but OpenShift AI should provide a default out-of-the-box

configuration for observability, so that a customer can see key health indicators for their

OpenShift AI installations and AI workloads. Metric collection also enables alerting and

autoscaling capabilities.

ODH/OpenShift AI Architectural Design Document (ADD) Template

The observability configuration needs to provide a simple consistent way for OpenShift AI

components to contribute metrics, traces, and logs, so that, as new features are implemented,

the product can provide consistent observability capabilities.

Recognizing that many customers already utilize their own observability solutions, OpenShift AI

should provide straightforward mechanisms for exporting metrics, traces, and logs to 3rd party

observability platforms.

Personas

● OpenShift Cluster Administrator - technical user with Kubernetes/OpenShift

administration knowledge \& elevated cluster permissions.

● OpenShift Cluster User - technical user with Kubernetes/OpenShift knowledge -

typically managing an application deployed on OpenShift. May or may not also be an

MLOps or AI Application Developer.

● OpenShift AI Cluster Admin - technical user that manages the RHOAI installation

including RHOAI configuration. Does not necessarily have cluster administrator access.

● OpenShift AI User

○ Data Scientist - Machine Learning focused user. May not have

Kubernetes/OpenShift knowledge.

○ MLOps Engineer - technical user with Kubernetes/OpenShift knowledge

responsible for supporting Data Science use cases (e.g. training, tuning,

inference deployments).

○ AI Application Developer - technical user with Kubernetes/OpenShift

knowledge responsible for integrating AI technologies into applications.

Definitions of terms

1

Core Concepts

● Observability lets you understand a system from the outside by letting you ask

questions about that system without knowing its inner workings.

● Telemetry refers to data emitted from a system and its behavior. The data can come in

the form of traces, metrics, and logs.

● Metrics are aggregations over a period of time of numeric data about your

infrastructure or application. Examples include: system error rate, CPU utilization, and

request rate for a given service.

1 Many term definitions from the OpenTelemetry project's definitions, projects' documentation

ODH/OpenShift AI Architectural Design Document (ADD) Template

● Scraping is a method of collecting metrics, where a scraper periodically gathers metrics

from known endpoints (typically HTTP endpoints).

● A distributed trace, more commonly known as a trace, records the paths taken by

requests (made by an application or end-user) as they propagate through multi-service

architectures, like microservice and serverless applications.

● A log is a timestamped message emitted by services or other components.

● Instrumentation is the process of having services or other components emit signals,

such as metrics, traces, and logs.

● Auto-instrumentation is a process where the OpenTelemetry operator injects

instrumentation libraries into workloads.

● The default monitoring stack is a set of components, including Prometheus and

Alertmanager, that are deployed as part of OpenShift Container Platform.

● User Workload Monitoring (UWM) is an optional mode of the default monitoring

stack that provides a single secondary Prometheus instance that stores metrics from

user workloads.

Upstream Projects

● Prometheus collects and stores metrics as time series data.

● Alertmanager is a Prometheus component that provides alert management and

routing.

● Tempo is an open source distributed tracing backend.

● Loki is an open source log aggregation backend.

● Perses is a dashboard tool that you can use to display Prometheus metrics \& Tempo

traces

2

.

● The OpenTelemetry Collector offers a vendor-agnostic implementation of how to

receive, process and export telemetry data.

● KEDA stands for Kubernetes Event-Driven Autoscaling, a project that provides

extensible autoscaling, with an expansive ecosystem of integrations.

Downstream Components

● Cluster Observability Operator is an optional operator from OpenShift that manages

Prometheus, Alertmanager, and Perses.

● Tempo Operator is an optional operator from OpenShift that manages Tempo.

● Red Hat build of OpenTelemetry provides an Operator that manages OpenTelemetry

Collector instances and instrumentation configuration.

2 See alternative: Grafana

ODH/OpenShift AI Architectural Design Document (ADD) Template

● Cluster Logging Operator is an optional Operator from OpenShift that provides log

collection using Loki and log forwarding capabilities.

● Custom Metrics Autoscaler is an operator that provides configurable replica

autoscaling driven by metrics, based on KEDA.

Internal Services

● Red Hat Observability Service is an internal deployment of Observatorium project

maintained by the Openshift Monitoring Team, that stores customer telemetry metrics

for internal use cases. Referred to as Observatorium in this document.

● Grafana is a visualization tool deployed widely for interacting with observability.

https://telemeter-lts-dashboards.datahub.redhat.com/ is the instance commonly used

for OpenShift AI.

● Dataverse is a Data Platform that provides storage and management of internal Red

Hat data including a Data Warehouse/Lake.

● Tableau is a visualization service used for internal data at Red Hat. Tableau uses

Dataverse and Amazon RedShift as data sources.

User stories

PRIORITY:

● P0 for mandatory for MVP

● P1 reduces the friction and increases the functional usefulness of the operator

● P2 useful, but does not prevent the use of the operator if not available

STATUS:

● C for Committed,

● D for Deferred

● T for Tentative

PHASE:

● MVP = Minimum Viable Product (a.k.a. Milestone 1)

● M# = Milestone #

ID Priority Description Status - Phase

RHOAIENG

-25561

P0 Enable centralized RHOAI metrics

collection

MVP - C

RHOAIENG

-25562

P0 Enable centralized RHOAI traces collection MVP - C

ODH/OpenShift AI Architectural Design Document (ADD) Template

RHOAIENG

-26160

P0 Enable built in alerting MVP - C

RHOAIENG

-25576

P1 Enable centrally managed observability

dashboards

M1 - C

RHOAIENG

-25563

P2 Build RHOAI adoption metrics pipeline M2 - D

Requirements for solution

The following notable tickets depend on capabilities provided by this solution:

● RHOAISTRAT-541 Implement Live Drift Monitoring for Generative AI (Systems +

Models)

● RHOAIRFE-693 Introduce E2E Observability for Llama Stack in the product

○ RHOAIRFE-544 Introduce Product Metrics and Observability Metrics for

OpenShift AI RAG/Agentic Components

● RHOAIRFE-570 GPU metrics for distributed workloads

● RHOAIRFE-551 vLLM traces for RHOAI

● RHOAIRFE-518 Metrics Logging and Visualization Across Epochs in Experiments and

Model Registry

Other related issues are captured in RHOAI Observability Jiras

Current architecture

Currently, monitoring is included for managed OpenShift AI installations only; some

components (e.g. model server) integrate with User Workload Monitoring (UWM) if present.

Additionally, the official OpenShift AI docs showcase how to use Grafana to observe model

serving metrics, based on work from the AI BU. The use case for this monitoring is providing

alerting and metrics for SRE to use in support of customer RHOAI installations.

The AI BU also has a kickstart for LlamaStack Observability, that illustrates how to configure

many of these components (as well as Grafana and UWM).

The current monitoring aggregates recording and alerting rules from components into

prometheus configuration, and deploys the following components:

● Alertmanager - Routes alerts to SRE

ODH/OpenShift AI Architectural Design Document (ADD) Template

● Blackbox-exporter - Prometheus component to monitor user-facing endpoints

● Prometheus - Independently deployed instance of prometheus for metrics

Proposed Architecture

ODH/OpenShift AI Architectural Design Document (ADD) Template

Telemetry Collection

The ODH operator defines an instance of the otel-collector in the gateway mode

3

. The Red Hat

Build of OpenTelemetry operator manages this instance. This deployment has 2 replicas by

default.

The ODH operator manages configuration of the otel-collector, provide central configuration of

telemetry, including common processing (e.g. captures Kubernetes identifiers, including

namespace and pod identifiers).

Metrics

The ODH operator defines a central Prometheus instance, managed by the Cluster

Observability Operator

4

. This Prometheus instance stores metrics in persistent volumes (one

per replica).

The platform configures the otel-collector to perform metrics scraping

5

. (Note: each

otel-collector replica scrapes independently, resulting in two scrapes by default). Each

5 See alternatives: Prometheus scraping and prometheusremotewrite

4 See alternatives In-cluster monitoring and User-workload monitoring

3 See alternative: otel-collector sidecar

ODH/OpenShift AI Architectural Design Document (ADD) Template

Prometheus instance scrapes metrics from the otel-collector deployment (Note: this requires a

support exception from the OBS team until the prometheus exporter goes GA)

6

.

The otel-collector discovers workloads (both component workloads and user workloads)

through use of a well-known common label: monitoring.opendatahub.io/scrape: "true". The

otel-collector supports 2 different strategies for discovery:

1\. Components defining ServiceMonitors with the common label to configure discovery.

2\. Components defining PodMonitors with the common label to configure discovery.

The ODH operator manages scraping configuration for Kueue as well as hardware accelerator

vendor metrics. The hardware accelerator metrics are normalized following the same

conventions as OpenShift Container Platform (see OBSDA-1087), ensuring OpenShift AI and

OpenShift Container Platform have consistent views of the accelerators.

The metrics data is queryable through a Thanos Querier frontend

7

(to handle deduplication and

merging across the Prometheus replicas), exposed via a stable endpoint. Components may use

this to implement autoscaling via the Custom Metrics Autoscaler. J-Proxy uses this endpoint for

autoscaling purposes. Perses dashboards use the querier as a data source as well.

The platform provides a namespace restricted view using a combination of kube-rbac-proxy and

prom-label-proxy restricts access to users having PodMetrics access (this pattern is used in

OpenShift Container Platform)

8

.

The platform provides an unrestricted view, restricted to users having NodeMetrics access.

Depending on the use case, access to the querier either uses a port which uses the

namespace-restricted view or a separate port which is unrestricted, but authorized via

kube-rbac-proxy. This requires a ClusterRole or Role to be added to a component's service

account.

8

https://github.com/openshift/cluster-monitoring-operator/blob/34568a9c7f7a0eeb1e8948d03cfc4049

adf72362/jsonnet/components/thanos-querier.libsonnet#L395-L624

https://github.com/openshift/cluster-monitoring-operator/blob/34568a9c7f7a0eeb1e8948d03cfc4049

adf72362/jsonnet/components/thanos-querier.libsonnet#L110-L180

7 See alternative: Kubernetes Service

6 Scaling risks discussed in Challenges

ODH/OpenShift AI Architectural Design Document (ADD) Template

The customer can optionally configure the otel-collector to export metrics to a 3rd party

observability platform. Support is according to the Red Hat Build of OpenTelemetry support

policy; as of 4.18:

● GA: otlp

● Tech Preview: prometheus, prometheusremotewrite, kafka, cloudwatch, file

If the customer opts into sending Red Hat telemetry, then the otel-collector exports a subset of

aggregated metrics to Observatorium (this telemetry is parallel to existing OpenShift Container

Platform Telemetry, allowing additional flexibility in collection of adoption metrics)

9

. Received

customer telemetry will be made available in a couple of different ways:

● OpenShift AI engineers use Grafana dashboards to access the OpenShift AI

Observatorium tenant (similar to Telemeter Grafana dashboards).

● P\&GE Business Insights provides Tableau dashboards.

Alerts

Initially, the platform extends the existing alerts from the managed deployment to be available

in both self-managed and managed installations.

Initially (while in tech preview) the alerts are viewable by administrators by manually exposing

the alertmanager port.

Furthermore, customers can configure alert routing by using the AlertmanagerConfig CR

(monitoring.rhobs/v1alpha1 AlertmanagerConfig)

10

.

Future work will Implement additional alerts.

Traces

The ODH operator provisions an instance of Tempo for storing distributed traces. Tempo stores

in user-provided object storage or a persistent volume. When the customer uses a persistent

volume, the ODH operator deploys Tempo in monolithic mode (appropriate for smaller

deployments). When the customer uses object storage, the ODH operator deploys Tempo in

microservices mode (appropriate for larger deployments). If the customer changes storage,

existing trace data is lost, and the ODH operator re-provisions Tempo.

10 This CR uses a different group than usual, because it uses the COO's embedded prometheus operator.

9 Alternative considered: Prometheus sending to Observatorium

ODH/OpenShift AI Architectural Design Document (ADD) Template

The ODH operator defines an Instrumentation CR, which the OpenTelemetry operator uses to

inject tracing configuration into component and customer workloads. The OpenTelemetry

operator configures workloads that use the OpenTelemetry SDK to submit trace data to the

central otel-collector. This also configures sampling rate (which may be necessary to tune in

very large deployments).

Instrumentation can be accomplished in two different ways:

1\. Integration of the OpenTelemetry SDK into application code.

2\. Injection of instrumentation libraries via auto-instrumentation.

Components are encouraged to adopt the OpenTelemetry SDK natively (this involves using

opentelemetry libraries or libraries that themselves integrate with the opentelemetry SDK), and

this is the preferred integration pattern, since "the Red Hat build of OpenTelemetry Operator

only supports the injection mechanism of the instrumentation libraries but does not support

instrumentation libraries", and auto-instrumentation drastically alters the shape of workloads,

including injection of an init-container, whereas native integration simply injects key environment

variables.

The ODH operator configures the otel-collector to receive and process trace data, exporting it

to the Tempo instance.

The customer can optionally configure the otel-collector to export traces to a 3rd party

observability platform. Support is according to the Red Hat Build of OpenTelemetry support

policy; as of 4.18:

● GA: otlp

● Tech preview: kafka, awsxray, file

Logs

Initially, OpenShift AI documentation recommends the use of the Cluster Logging Operator for

customers with log retention needs. Future work will provide out-of-the-box configuration and

integration with the Cluster Logging Operator.

Using the Cluster Logging Operator, customers can store logs using Loki and forward logs to

external 3rd party observability platforms, including cloudwatch, elasticsearch, kafka, and

splunk.

ODH/OpenShift AI Architectural Design Document (ADD) Template

Visualizations \& Dashboards

The ODH operator defines a central Perses instance (managed via the Cluster Observability

Operator), configured to query both metrics (via Thanos/Prometheus) and traces (via Tempo).

Components can utilize Perses to build re-usable dashboards, and customers can build custom

dashboards using Perses. Note: Perses does not support logs yet, but plans to in a future

release.

Because OpenShift AI uses a UI that is completely separate from OpenShift's console, and to

give users observability without needing to have OpenShift console access, OpenShift AI

provides its own Perses UI, separate from the OpenShift Console UIPlugin integration

introduced in the Cluster Observability Operator.

Component teams provide dashboards using the PersesDashboard CR, which allows them to be

referenced both in the component's UI as well as globally across the cluster. The OpenShift AI

Platform team will provide guidelines (including best practices and recommendations), but the

work to implement dashboards will be done by individual component teams.

OpenShift AI will re-use the styling that OpenShift Container Platform uses for the Perses

integration.

Perses UI (upstream):

Perses dashboards in OpenShift:

ODH/OpenShift AI Architectural Design Document (ADD) Template

Perses traces in OpenShift:

ODH/OpenShift AI Architectural Design Document (ADD) Template

Security Considerations

Pod-to-pod network communication

● Scraping protocol (http or https) is controlled by components. Enabling TLS across all

scraping endpoints is likely to be an ongoing effort. The relative risk depends on the

sensitivity of the metrics data, and while metrics are typically not considered sensitive,

the OpenShift AI Platform team recommends following the best practices for scraping

defined in the Red Hat Monitoring Group Handbook.

● Prometheus gathering metrics from otel-collector \& trace data pushed to the

otel-collector - the otel-collector is configured using the generated serving certificate

for the headless service

11

.

● The otel-collector pushes of traces to Tempo use the Tempo gateway service, which

uses TLS.

Data at rest

Observability data is stored in two possible types of storage:

1\. Persistent volumes - where the storage configuration of the underlying cluster may

provide encryption at rest.

2\. Object storage provided by the customer - where the object storage provider may

provide encryption at rest.

11

https://github.com/open-telemetry/opentelemetry-operator/blob/e8f834acb8643c3e68fa6b268819c9

31ffbceace/README.md?plain=1#L81

ODH/OpenShift AI Architectural Design Document (ADD) Template

Retention

● Metrics will be retained for 90 days by default, with configurable retention.

● Traces will be retained for 2 days (this is the Tempo default).

Authentication

● The otel-collector scrapes of workload metrics are component controlled, and should

utilize mTLS via a service account for authentication/authorization.

● Prometheus' requests to gather metrics from otel-collector utilizes mTLS via a service

account for authentication/authorization.

● Components pushing traces to the otel-collector gateway are unauthenticated

12

.

● The otel-collector connection to Tempo is authenticated via ServiceAccount token.

● The thanos querier endpoint is protected by kube-rbac-proxy, which supports tokens

and client certificates.

Authorization

● Scraping access is primarily controlled by NetworkPolicy. Each component creates a

network policy allowing the "redhat-ods-monitoring" monitoring namespace to scrape all

ports used in workload metrics endpoints.

● Prometheus is authorized to gather metrics from the otel-collector deployment by being

in the same namespace

13

.

● Components and user workloads' abilities to send traces is controlled by NetworkPolicy.

All workloads on a cluster will be able to send traces by default.

● The ODH operator explicitly authorizes otel-collector to write traces to Tempo via a

ClusterRole.

● The thanos querier endpoint is protected by kube-rbac-proxy with two endpoints, one

that restricts metrics access to namespaces that the user has access to (for general

OpenShift AI Users), and the other requiring an elevated role (for the OpenShift AI

Cluster Admin persona).

13At this time, the otel-collector has plugins that handle authentication, but none that handle authorization

independently, though some options exist, same as Authenticated Metrics Scrapes.

12 Unfortunately, the OTEL SDK does not have a standard way to specify authentication. It would be

possible on a per-language basis (e.g. Python, Go, etc).

ODH/OpenShift AI Architectural Design Document (ADD) Template

Quality Assurance Considerations

● We have a bunch of tests about Monitoring in the ods-ci repo, that are currently running

against a RHOAI installation in a managed cluster as part of the nightlies and as part of

the RHOAI platform signoffs. Given that in the new architecture metrics will still be

published to the Prometheus instance, we just need to adjust the test suite to query the

new instance.

● Operator dependencies (Cluster Observability Operator. OpenTelemetry Operator,

Tempo Operator for Phase 1) need to be added as new dependent operators for all the

ODH/RHOAI deployments, including proper mirroring in Disconnected environments.

● Test in Self managed environments through ODH/RHOAI nightlies (running the

Monitoring test-expression in the ods-ci repository)

● Test in Managed environments through RHOAI nightlies (ODH is not supported in

managed environments) (running the Monitoring test-expression in the ods-ci

repository)

● Test in Disconnected environment through ODH/RHOAI nightlies (running the

Monitoring test-expression in the ods-ci repository)

● Tracing and Logging test suites will need to be implemented to verify the propagation to

Tempo, Loki respectively.

Managed Service Considerations

Service impact(s) that SRE will need to consider or be aware of

● RHODS - MT-SRE Acceptance Criteria Checklist

● Metrics must be in place for SRE

● Example SOP.

● Check with the SRE team and AI services managers if in doubt.

Resource Considerations

Include the PSAP team in the review if there are resource considerations.

● Resources consumed by the feature (deviations from defaults are in bold)

○ Prometheus

■ 2 replicas

■ Requests (each)

● 110m CPU

● 306Mi memory

■ Limits (each)

ODH/OpenShift AI Architectural Design Document (ADD) Template

● 510m CPU

● 522Mi memory

■ Storage (each)

● 5Gi PVC

○ Alertmanager

■ 2 replicas

■ Requests (each)

● CPU - 100m

● Memory - 250Mi

■ Limits (each)

● CPU - 1

● Memory - 250Mi

○ Thanos Querier

■ 1 replica

■ Requests (each)

● CPU - 100m

● Memory - 64Mi

■ Limits (each)

● CPU - 1

● Memory - 64Mi

○ Tempo

■ 1 replica

■ Requests (each)

● CPU - 100m

● Memory - 256Mi

■ Limits (each)

● CPU - 1

● Memory - 256Mi

○ Otel-collector

■ 2 replicas

■ Requests (each)

● CPU - 100m

● Memory - 256Mi

■ Limits (each)

● CPU - 1

● Memory - 256Mi

○ kube-rbac-proxy and prom-label-proxy resource usage expected to be minimal

(will scale alongside thanosquerier replicas)

ODH/OpenShift AI Architectural Design Document (ADD) Template

● What performance testing will be performed?

○ We will seek guidance from the observability team around what prior load testing

has already been done for the components we're using. There should be no need

to perform additional testing.

Documentation Considerations

● What information needs to be passed to the user to inform them about this change?

○ Why should users use the new out-of-the-box monitoring capabilities of

OpenShift AI?

■ OpenShift Cluster Administrators and OpenShift AI Cluster Admins can

monitor resource utilization for storage and compute (including

accelerators; i.e. GPU utilization).

■ Data Scientists and MLOps Engineers can see metrics for the

performance of a deployed model.

■ AI Application Developers can see metrics and traces for their AI-enabled

applications.

○ Users need to understand how to configure the feature, especially:

■ How to opt-in to adoption metrics

■ How to manage metrics \& traces access

■ How to configure storage size

○ Users need to be directed to the appropriate documentation for the individual

components, especially:

■ Which exporters are supported for 3rd party integrations

○ Large scale users may need to scale individual components

○ NOTE: per-component monitoring changes/documentation (e.g.

RHOAIRFE-572) is out-of-scope for this STRAT, and will be handled in related

follow-up work.

● What user tasks change as a result of this proposal, and how?

○ Administrators will need to install additional operators before configuring

OpenShift AI

○ Users are able to create custom dashboards to observe OpenShift AI

○ There is now a new option that doesn't require UWM, the following docs need

updates to make users aware of the new option:

■ Chapter 2. Serving models on the single-model serving platform | Serving

models | Red Hat OpenShift AI Self-Managed

ODH/OpenShift AI Architectural Design Document (ADD) Template

Alternatives Considered / Rejected

Grafana

Perses is the solution chosen by the Openshift Observability team (OBSDA-390) to replace

Grafana as the supported visualisation tool for metrics in OpenShift (in the long term). It's being

integrated into the Cluster Observability Operator. We chose to use Perses to align.

Additionally, nothing in this proposal discourages the use of Grafana as a supported 3rd party

tool.

otel-collector sidecar

This option requires an additional sidecar on every workload pod in order to collect traces. We'll

use the gateway pattern instead to minimize resource usage.

Prometheus scraping

Having the central Prometheus perform the scraping does not allow us to take advantage of

centralized processing of telemetry via the otel-collector. While there is not currently any

required processing that can't be accomplished with Prometheus scraping, consistently using

the otel-collector to process metrics gives us a single place for future work to add attributes to

the signals (e.g., tenancy attributes). Additionally, with some components, specifically

LlamaStack deployments, the metrics are sent via OTLP anyways.

When prometheus does the scraping, it requires the export of 3rd party metrics to scrape

Prometheus periodically.

prometheusremotewrite

We explored using the prometheusremotewrite exporter during PoC, but its configuration was

complex since it involved multiple writers and multiple destinations (2 otel-collector replicas to 2

prometheus replicas).

In-cluster monitoring

This is inflexible, can't be scaled, and is not officially supported. Product managers expressed

concerns over risks of overloading the in-cluster monitoring stack.

ODH/OpenShift AI Architectural Design Document (ADD) Template

User Workload Monitoring

● We cannot be sure User Workload Monitoring is enabled.

● We can’t enforce our configuration.

● We can’t guarantee our configuration does not interfere with the customer configuration;

an OpenShift Cluster Administrator or User on a shared cluster may set up and

customize User Workload Monitoring for different purposes (e.g. non-AI applications).

NOTE: components already integrating with UWM will need to follow the migration instructions.

Kubernetes Service

Using a kubernetes service to query the Prometheus replicas is feasible, but it makes the results

inconsistent, as there are slight deviations in query results depending on the replica that

services the query.

Authenticated Metrics Scrapes

Strategies supported by Prometheus itself, as well as otel-collector are:

● Basic auth

● Bearer token

● OAuth2

● mTLS

Prometheus sending to Observatorium

Having Prometheus send adoption metrics to Observatorium would remove a network hop; see

details here:

https://rhobs-handbook.netlify.app/services/rhobs/configuring-clients-for-observatorium.md/

\#using-a-self-managed-prometheus-server

However, using otel-collector to send adoption metrics centralizes processing of the metrics,

giving us a place to filter and transform metrics as needed.

Challenges

● Prometheus is optimized to scrape many targets rather than a single target with many

samples. We do not have insights on performance of Prometheus federation, and

ODH/OpenShift AI Architectural Design Document (ADD) Template

choosing to use federation to have Prometheus ingest metrics from the otel-collector

could have scalability issues.

● Very large deployments may require manually scaling the Prometheus instance,

OpenTelemetry collector deployment, or other deployments.

○ Future work in the Observability components may address this challenge (e.g.

work-in-progress autoscaling for Prometheus).

○ Scaling brings non-trivial considerations (see e.g. Scaling the Collector |

OpenTelemetry )

● Auto-instrumentation may not be viable for every workload; we may have to manually

configure some workload.

● Authentication could be hardened over time.

Dependencies

● Dependent products

○ OpenShift Container Platform

■ Image: openshift4/ose-prom-label-proxy

■ Image: openshift4/ose-kube-rbac-proxy

○ Cluster Observability Operator

○ Tempo Operator

○ Red Hat build of OpenTelemetry

● The centralized monitoring capabilities are intended to make the use of User Workload

Monitoring unnecessary. However, we will not actively prevent User Workload Monitoring

from working. Components using user workload monitoring should plan to migrate to

using new monitoring services as soon as feasible.

New feature/component considerations

● Code will land in the existing operator codebase.

● Because OpenShift AI will be dependent on the other operators (Tempo,

OpenTelemetry, Cluster Observability Operator), OpenShift AI customers will be

dependent on those Operators for security updates when necessary.

○ Disconnected installations should mirror these dependent operators if the user

wishes to use the out-of-the-box observability.

ODH/OpenShift AI Architectural Design Document (ADD) Template

User Workload Monitoring (UWM) Migration

Compon ents currently having metrics collected into UWM will need to apply the

monitoring.opendatahub.io/scrape: "true" label to PodMonitor/ServiceMonitor objects to

get their metrics into the platform-provided Prometheus instance.

Note: it is possible to omit a whole namespace from UWM by adding a label of

openshift.io/user-monitoring=false to the namespace, but this proposal doesn't advocate

for this.

Components that query prometheus will need to change their endpoints to the

platform-provided thanos querier endpoint, and will need to apply a RoleBinding to authorize

the component's service account to access metrics.

Notably, pending work (RHOAISTRAT-520) to enable autoscaling in Kserve Raw (used by

watsonx) will need to do both of the above.

References/Resources

● Observability primer | OpenTelemetry

● Miro - OpenShift AI Observability Architecture

● RHOAI Observability Jiras

● Cluster Observability Operator | Red Hat Product Documentation

● Distributed tracing | Red Hat Product Documentation

● Red Hat build of OpenTelemetry | Red Hat Product Documentation

● Auto-instrumentation in the Red Hat build of OpenTelemetry Operator

● Logging | OpenShift Container Platform | 4.18 | Red Hat Documentation

Review Status

Expected life cycle for each ADD will be as follows, no ADD once moved past Draft can be

‘deleted’ or ‘removed’. :

● Draft: An ADD in Draft indicates its a concept under consideration, being scoped, being

written up. No action is expected from anyone other than the author, and there are no

time pressures towards moving from the draft state.

● Proposed : ADDs can move to the proposed state once a sponsor has signed up, and

the ADD can then start to be socialized within the team / with PMs and with RFCs / RFEs

can be scoped in.

ODH/OpenShift AI Architectural Design Document (ADD) Template

● Reviewing : Once moved to Reviewing state, the ADD must be brought to the SIG-SRE

weekly call for consideration based on feedback already received. An ADD from this

stage can either be moved back to the Proposed state, if gaps are identified or

conversations need to be closed, or its Accepted / Rejected. And actions as follow-up

should be executed.

● Accepted : Once an ADD is ‘accepted’, it must be moved into read-only mode and no

further changes are allowed.

● Rejected : From the reviewing state, it's possible to move ADDs to a rejected state if

they do not align with goals, or if stakeholders have fundamental concerns.

● Superseded : ADDs once they move to Accepted state can no longer be changed, since

goals and execution is established. Changes would be implemented by replacement

ADDs with the previous ADD being moved to ‘superseded’ state, and a comment being

added into the ADD metadata itself indicating which ADD now supersedes this specific

one. The general pattern to follow is the IETF RFC flow.



