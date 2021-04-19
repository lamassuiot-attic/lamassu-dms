<a href="https://www.lamassu.io/">
  <img src="logo.png" alt="Lamassu logo" title="Lamassu" align="right" height="80" />
</a>

Lamassu
=======
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-blue.svg)](http://www.mozilla.org/MPL/2.0/index.txt)

[Lamassu](https://www.lamassu.io) project is a Public Key Infrastructure (PKI) for the Internet of Things (IoT).

## Device Manufacturing System
This repository contains an enrolling agent for manufacturing systems to provision their devices with a certificate obtained from Lamassu PKI.

### Project Structure
The Device Manufacturing System is composed of two services:
1. Enroller: This service performs basic operations to manage the CSRs (Certificate Signing Request) submitted by the manufacturing system to the [Enroller](https://github.com/lamassuiot/enroller). Those operations that need information about the status about the CSRs are proxied to the [Enroller](https://github.com/lamassuiot/enroller).
2. Manufacturing: This service performs the operations to build and send a CSR for a device to the [Enroller](https://github.com/lamassuiot/enroller). This operation is authenticated with a mTLS connection with the SCEP proxy server. 

Each service has its own application directory in `cmd/` and libraries in `pkg/`.

## Installation
To compile the Device Manufacturing System follow the next steps:
1. Clone the repository: `go get github.com/lamassuiot/device-manufacturing-system`
2. Run the Enroller service compilation script: `cd src/github.com/lamassuiot/device-manufacturing-system/cmd/enroller && ./release.sh`
3. Run the Manufacturing service compilation script: `cd src/github.com/lamassuiot/device-manufacturing-system/cmd/manufacturing && ./release.sh`

The binaries will be compiled in the `/build` directory.

## Usage
Each service of the Device Manufacturing System should be configured with some environment variables.

**Enroller service**
```
ENROLLER_PORT=8889 //Enroller service API port.
ENROLLER_UIHOST=manufacturingui //UI host (for CORS 'Access-Control-Allow-Origin' header).
ENROLLER_UIPROTOCOL=https //UI protocol (for CORS 'Access-Control-Allow-Origin' header).
ENROLLER_UIPORT=443 //UI port (for CORS 'Access-Control-Allow-Origin' header).
ENROLLER_CONSULPROTOCOL=https //Consul server protocol.
ENROLLER_CONSULHOST=consul //Consul server host.
ENROLLER_CONSULPORT=8501 //Consul server port.
ENROLLER_CONSULCA=consul.crt //Consul server certificate CA to trust it.
ENROLLER_KEYCLOAKREALM=<KEYCLOAK_REALM> //Keycloak realm configured.
ENROLLER_KEYCLOAKHOSTNAME=keycloak //Keycloak server hostname.
ENROLLER_KEYCLOAKPORT=8443 //Keycloak server port.
ENROLLER_KEYCLOAKPROTOCOL=https //Keycloak server protocol.
ENROLLER_KEYCLOAKCA=keycloak.crt //Keycloak server certificate CA to trust it.
ENROLLER_CERTFILE=enroller.crt //Enroller service API certificate.
ENROLLER_KEYFILE=enroller.key //Enroller service API key.
ENROLLER_PROXYADDRESS=https://enroller:8085 //Lamassu Enroller address to proxy requests that need information about CSR status.
ENROLLER_PROXYCA=enroller.crt //Lamassu Enroller certificate CA to trust it.
JAEGER_SERVICE_NAME=dms-enroller //Jaeger tracing service name.
JAEGER_AGENT_HOST=jaeger //Jaeger agent host.
JAEGER_AGENT_PORT=6831 //Jaeger agent port.
```

**Manufacturing service**
```
MANUFACTURING_PORT=8888 //Manufacturing service port.
MANUFACTURNG_UIHOST=manufacturingui //UI host (for CORS 'Access-Control-Allow-Origin' header).
MANUFACTURING_UIPORT=443 //UI port (for CORS 'Access-Control-Allow-Origin' header).
MANUFACTURING_UIPROTOCOL=https //UI protocol (for CORS 'Access-Control-Allow-Origin' header).
MANUFACTURING_KEYCLOAKHOSTNAME=keycloak //Keycloak server hostname.
MANUFACTURING_KEYCLOAKPORT=8443 //Keycloak server port.
MANUFACTURING_KEYCLOAKPROTOCOL=https //Keycloak server protocol.
MANUFACTURING_KEYCLOAKREALM=<KEYCLOAK_REALM> //Keycloak realm configured.
MANUFACTURING_KEYCLOAKCA=keycloak.crt //Keycloak server certificate CA to trust it.
MANUFACTURING_CONSULPROTOCOL=https //Keycloak server protocol.
MANUFACTURING_CONSULHOST=consul //Consul server host.
MANUFACTURING_CONSULPORT=8443 //Keycloak server port.
MANUFACTURING_CONSULCA=consul.crt //Consul server certificate CA to trust it.
MANUFACTURING_CERTFILE=manufacturing.crt //Manufacturing service API certificate.
MANUFACTURING_KEYFILE=manufacturing.key //Manufacturing service API key.
MANUFACTURING_AUTHKEYFILE=manufacturing_system.key //Device Manufacturing System private key. Used to establish mTLS with SCEP proxy server.
MANUFACTURING_PROXYADDRESS=https://scepproxy //SCEP proxy server address.
MANUFACTURING_PROXYCA=scepproxy.crt //SCEP proxy server certificate CA to trust it.
JAEGER_SERVICE_NAME=dms-manufacturing //Jaeger tracing service name.
JAEGER_AGENT_HOST=jaeger //Jaeger agent host.
JAEGER_AGENT_PORT=6831 //Jaeger agent port.
```
The prefixes `(ENROLLER_)` and `(MANUFACTURING_)` used to declare the environment variables can be changed in `cmd/enroller/main.go` and `cmd/manufacturing/main.go`:

```	
cfg, err := configs.NewConfig("enroller")
cfg, err := configs.NewConfig("manufacturing")
```

For more information about the environment variables declaration check `pkg/enroller/configs` and `pkg/manufacturing/configs`.

## Docker
The recommended way to run [Lamassu](https://www.lamassu.io) is following the steps explained in [lamassu-compose](https://github.com/lamassuiot/lamassu-compose) repository. However, each component can be run separately in Docker following the next steps.

**Enroller service**
```
docker image build -t lamassuiot/lamassu-dms-enroller:latest -f Dockerfile.manufacturingenroll .
docker run -p 8889:8889
  --env ENROLLER_PORT=8889
  --env ENROLLER_UIHOST=manufacturingui
  --env ENROLLER_UIPROTOCOL=https
  --env ENROLLER_UIPORT=443
  --env ENROLLER_CONSULPROTOCOL=https
  --env ENROLLER_CONSULHOST=consul
  --env ENROLLER_CONSULPORT=8501
  --env ENROLLER_CONSULCA=consul.crt
  --env ENROLLER_KEYCLOAKREALM=<KEYCLOAK_REALM>
  --env ENROLLER_KEYCLOAKHOSTNAME=keycloak
  --env ENROLLER_KEYCLOAKPORT=8443 
  --env ENROLLER_KEYCLOAKPROTOCOL=https
  --env ENROLLER_KEYCLOAKCA=keycloak.crt
  --env ENROLLER_CERTFILE=enroller.crt
  --env ENROLLER_KEYFILE=enroller.key
  --env ENROLLER_PROXYADDRESS=https://enroller:8085
  --env ENROLLER_PROXYCA=enroller.crt
  --env JAEGER_SERVICE_NAME=dms-enroller
  --env JAEGER_AGENT_HOST=jaeger
  --env JAEGER_AGENT_PORT=6831
  lamassuiot/lamassu-dms-enroller:latest
```
**Manufacturing service**
```
docker image build -t lamassuiot/lamassu-dms-manufacturing:latest -f Dockerfile.manufacturing .
docker run -p 8888:8888
  --env MANUFACTURING_PORT=8888 
  --env MANUFACTURNG_UIHOST=manufacturingui 
  --env MANUFACTURING_UIPORT=443
  --env MANUFACTURING_UIPROTOCOL=https
  --env MANUFACTURING_KEYCLOAKHOSTNAME=keycloak
  --env MANUFACTURING_KEYCLOAKPORT=8443
  --env MANUFACTURING_KEYCLOAKPROTOCOL=https
  --env MANUFACTURING_KEYCLOAKREALM=<KEYCLOAK_REALM>
  --env MANUFACTURING_KEYCLOAKCA=keycloak.crt
  --env MANUFACTURING_CONSULPROTOCOL=https
  --env MANUFACTURING_CONSULHOST=consul
  --env MANUFACTURING_CONSULPORT=8443
  --env MANUFACTURING_CONSULCA=consul.crt
  --env MANUFACTURING_CERTFILE=manufacturing.crt
  --env MANUFACTURING_KEYFILE=manufacturing.key
  --env MANUFACTURING_AUTHKEYFILE=manufacturing_system.key
  --env MANUFACTURING_PROXYADDRESS=https://scepproxy
  --env MANUFACTURING_PROXYCA=scepproxy.crt
  --env JAEGER_SERVICE_NAME=dms-manufacturing
  --env JAEGER_AGENT_HOST=jaeger
  --env JAEGER_AGENT_PORT=6831
  lamassuiot/lamassu-dms-manufacturing:latest
```
## Kubernetes
[Lamassu](https://www.lamassu.io) can be run in Kubernetes deploying the objects defined in `k8s/` directory. `provision-k8s.sh` script provides some useful guidelines and commands to deploy the objects in a local [Minikube](https://github.com/kubernetes/minikube) Kubernetes cluster.
