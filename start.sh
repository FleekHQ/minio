#!/bin/bash

export MINIO_ACCESS_KEY=minio
export MINIO_SECRET_KEY=miniostorage
export MINIO_AUDIT_WEBHOOK_ENABLE_target1="on"
export MINIO_AUDIT_WEBHOOK_ENDPOINT_target1=https://webhook.site/227bab3d-aacd-4cff-a087-9beee644e7ef
# iam user: s3x-lambda-caller
export AWS_ACCESS_KEY_ID="AKIA3FV76Z24YZB67Z7G"
export AWS_SECRET_ACCESS_KEY="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"
export CRUD_HANDLER_FUNCTION="httpLogger"

./minio gateway s3x --temporalx.endpoint=52.43.173.21:9090 --temporalx.insecure
