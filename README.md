# KubeService

A simple CRD controller for micro-service management.

## What's KubeService

KubeService is a CRD controller build on KubeBuilder, 
and this define some CR in kubernetes like: `App`、`MicroService` for micro-service management easier .

The logical structure like:

![](https://github.com/Coderhypo/KubeService/blob/master/docs/img/logical_structure.jpg?raw=true)

## Feature

### Deploy management

User can define a `App` resource to manage multiple micro services, 
and use DeployVersion to making multiple versions coexist.

### Load balancing configuration

User use define `CurrentVersion` make the current version provide services, 
and define `Canary` to Canary Deploy other versions.

![](https://github.com/Coderhypo/KubeService/blob/master/docs/img/loadbalance.jpg?raw=true)
