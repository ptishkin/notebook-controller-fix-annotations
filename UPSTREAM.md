# Upstream provenance

- Source repository: https://github.com/kubeflow/kubeflow
- Source branch: v1.10-branch
- Import commit: 90e987bf87d3e7c900926310b00bfa16b59e41eb
- Imported paths:
  - components/notebook-controller -> notebook-controller/
  - components/common              -> common/
- go.mod of notebook-controller pins common as
  github.com/kubeflow/kubeflow/components/common v0.0.0-20220218084159-4ad0158e955e
  (commit 4ad0158e955e). A replace directive to ../common is added in this repo;
  verified that reconcilehelper code is identical between 4ad0158e955e and
  v1.10-branch HEAD (only centraldashboard files differ).
- License: Apache-2.0 (see notebook-controller/third_party/license.txt)
