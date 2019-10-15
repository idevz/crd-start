#!/usr/bin/env bash

### BEGIN ###
# Author: idevz
# Since: 10:23:35 2019/08/27
# Description:       running tools
# run          ./run.sh
#
# Environment variables that control this script:
#
### END ###

set -ex
BASE_DIR=$(dirname $(cd $(dirname "$0") && pwd -P)/$(basename "$0"))
CRD_START_ROOT=$BASE_DIR/..

DOCKER_IMAGE_TAG="zhoujing/kube-crdstart"
DEBUG_DOCKER_IMAGE_TAG="zhoujing/kube-crdstart:debug"

export GO111MODULE=on

function gen_crdstart_deepcopy() {
    export GOPATH=$(echo $GOPATH | awk -F: '{print $1}')
    local custom_resource_name="crdstart"
    local custom_resource_version="v1alpha1"
    local module_name="github.com/idevz/crd-start"
    # https://github.com/kubernetes/code-generator/blob/master/generate-groups.sh
    local gen_sh=${GENSH:-"$G1/src/k8s.io/code-generator/generate-groups.sh"}

    ${gen_sh} all \
        ${module_name}/pkg/client \
        ${module_name}/pkg/apis \
        "$custom_resource_name:$custom_resource_version" \
        --output-base ${BASE_DIR}
}

do_what=${1}
shift

case "${do_what}" in
init)
    # crd
    # rbac
    # pprof
    # Dcreater
    ;;
g)
    gen_crdstart_deepcopy
    ;;
bp | build_and_push_docker_images)
    if [[ -x $(which go 2>/dev/null) ]]; then
        GOLANG_GOARCH="amd64" GOOS="linux" go build \
            -o ${CRD_START_ROOT}/build/crdstart ${CRD_START_ROOT}
        GOGCFLAGS="-N -l" GOGCFLAGS="-e" GOLANG_GOARCH="amd64" GOOS="linux" go build \
            -o ${CRD_START_ROOT}/build/crdstart-debug ${CRD_START_ROOT}
    fi
    if [[ -x $(which docker 2>/dev/null) ]]; then
        docker build -t ${DOCKER_IMAGE_TAG} ${CRD_START_ROOT}/build
        docker build -t ${DEBUG_DOCKER_IMAGE_TAG} -f ${CRD_START_ROOT}/build/debug.Dockerfile ${CRD_START_ROOT}/build
        docker push ${DOCKER_IMAGE_TAG}
        docker push ${DEBUG_DOCKER_IMAGE_TAG}
    fi
    ;;
r | run_as_deployment)
    kubectl run crdstart --image=${DOCKER_IMAGE_TAG} --labels="app=de-crdstart" --replicas=1 -- --deploymentTpl=/artifacts/deployment-tpl.yaml --v=1
    ;;
rd | run_debug)
    kubectl run cdebug --image=${DEBUG_DOCKER_IMAGE_TAG} --replicas=1 -- --deploymentTpl=/artifacts/deployment-tpl.yaml --v=1
    ;;
d | debug)
    kubectl port-forward $(kubectl get pod -l run=cdebug -o jsonpath='{.items[0].metadata.name}') 40000:40000 --address=0.0.0.0 &
    ;;
*)
    echo "
usage:

    ./run.sh  [command]
    
    eg.
    ./run.sh          g         generate the kube deepcopy, clientset, infomers
    ./run.sh          bp        build and push the docker images, both common and debug
    ./run.sh          r         run crdstart as a deployment
    ./run.sh          rd        run crdstart as a deployment for debug
    ./run.sh          d         port-forward a running pod for remote debug
"
    ;;
esac

# make WHAT=cmd/kubectl KUBE_BUILD_PLATFORMS=linux/amd64 &&
# 	make WHAT=cmd/kube-apiserver KUBE_BUILD_PLATFORMS=linux/amd64 &&
# 	make WHAT=cmd/kube-controller-manager KUBE_BUILD_PLATFORMS=linux/amd64 &&
# 	make WHAT=cmd/kube-proxy KUBE_BUILD_PLATFORMS=linux/amd64 &&
# 	make WHAT=cmd/kube-scheduler KUBE_BUILD_PLATFORMS=linux/amd64 &&
# 	make WHAT=cmd/kubelet KUBE_BUILD_PLATFORMS=linux/amd64

# make GOGCFLAGS="-N -l" GOGCFLAGS="-e" WHAT=cmd/kubectl KUBE_BUILD_PLATFORMS=linux/amd64 &&
# make GOGCFLAGS="-N -l" GOGCFLAGS="-e" WHAT=cmd/kube-apiserver KUBE_BUILD_PLATFORMS=linux/amd64 &&
# make GOGCFLAGS="-N -l" GOGCFLAGS="-e" WHAT=cmd/kube-controller-manager KUBE_BUILD_PLATFORMS=linux/amd64 &&
# make GOGCFLAGS="-N -l" GOGCFLAGS="-e" WHAT=cmd/kube-proxy KUBE_BUILD_PLATFORMS=linux/amd64 &&
# make GOGCFLAGS="-N -l" GOGCFLAGS="-e" WHAT=cmd/kube-scheduler KUBE_BUILD_PLATFORMS=linux/amd64 &&
# make GOGCFLAGS="-N -l" GOGCFLAGS="-e" WHAT=cmd/kubelet KUBE_BUILD_PLATFORMS=linux/amd64
