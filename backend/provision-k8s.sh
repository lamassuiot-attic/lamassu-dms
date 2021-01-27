#! /bin/sh
# ==================================================================
#  _                                         
# | |                                        
# | |     __ _ _ __ ___   __ _ ___ ___ _   _ 
# | |    / _` | '_ ` _ \ / _` / __/ __| | | |
# | |___| (_| | | | | | | (_| \__ \__ \ |_| |
# |______\__,_|_| |_| |_|\__,_|___/___/\__,_|
#                                            
#                                            
# ==================================================================

minikube kubectl -- create secret generic dms-manufacturing-certs --from-file=./certs/consul.crt --from-file=./certs/enroller.crt --from-file=./certs/keycloak.crt --from-file=./certs/manufacturing.crt --from-file=./certs/manufacturing.key --from-file=./certs/operator.key --from-file=./certs/scepproxy.crt
minikube kubectl -- create secret generic dms-enroller-certs --from-file=./certs/keycloak.crt --from-file=./certs/manufacturing.crt --from-file=./certs/manufacturing.key --from-file=./certs/enroller.crt --from-file=./certs/consul.crt

minikube kubectl -- apply -f k8s/manufacturing-deployment.yml
minikube kubectl -- apply -f k8s/manufacturing-service.yml

minikube kubectl -- apply -f k8s/enroller-deployment.yml
minikube kubectl -- apply -f k8s/enroller-service.yml