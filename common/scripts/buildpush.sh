#!/bin/bash
# Licensed Materials - Property of IBM
# (C) Copyright IBM Corporation 2016, 2020. All Rights Reserved.
# US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#

SCRIPT_DIR=$(dirname $0)

IMGNAME=quay.io/horis233/ibm-cs-webhook

docker build -t $IMGNAME -f ${SCRIPT_DIR}/../../build/Dockerfile .
docker push  $IMGNAME