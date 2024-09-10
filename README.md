# k8s-function
This repo contains a K8s Operator to execute a python function using custom resource. Provide basic functionality to showcase the power k8s as a framework to customize to your needs.
It defines a CRD to specify a Python Function to be executes. It also provides a `/scale` sub-resources, similar to a Deployment which enables this CR to be used with autoscaling frameworks like KEDA 

The functionality is intentionally kept very basic to better understand the workflow of custom resources in k8s. The code is fairly simple to understand but at a high level its a controller like a Deployment but without all the features. 
It creates a Pod with a Python runtime specified and executes the code given in the CRD. It specifies a few parameters like `replicas` for scaling and some cleanup functionality. It doesnt do any of retry and advanced logic as a Deployment.
