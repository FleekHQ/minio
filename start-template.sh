#!/bin/bash

export MINIO_ACCESS_KEY=minio
export MINIO_SECRET_KEY=miniostorage
export MINIO_AUDIT_WEBHOOK_ENABLE_target1="on"
export MINIO_AUDIT_WEBHOOK_ENDPOINT_target1=https://webhook.site/227bab3d-aacd-4cff-a087-9beee644e7ef
export AWS_ACCESS_KEY_ID="<access key id>"
export AWS_SECRET_ACCESS_KEY="<secret>"

./minio gateway s3x --temporalx.endpoint=52.43.173.21:9090 --temporalx.insecure
