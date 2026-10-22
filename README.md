# notebook-controller-fix-annotations

Patched build of the Kubeflow **notebook-controller** (Kubeflow 1.10,
`ghcr.io/kubeflow/kubeflow/notebook-controller:v1.10.0`) that **stops wiping
foreign labels/annotations** on the Services and StatefulSets it manages.

## The problem

Upstream `reconcilehelper.CopyServiceFields` (and `CopyStatefulSetFields`,
`CopyDeploymentSetFields`) did:

```go
to.Annotations = from.Annotations
```

Since `generateService()` builds the desired Service **without any
annotations**, every reconcile replaced the live Service annotations with an
empty map and pushed the wipe via `r.Update()`. Any annotation set by another
controller (OVN CNI, istio, monitoring, reloader, ...) was deleted, that
controller patched it back, the change re-triggered the notebook-controller
(`Owns(&corev1.Service{})`) — an endless patch/update fight, clearly visible
in the kube-apiserver audit log.

## The fix

`mergeStringMap()` merges desired keys into the live object **without
deleting keys that exist only on the live object**:

- owned keys are still added/updated (behavior for vanilla Kubeflow is
  unchanged — the controller generates no Service annotations at all);
- foreign keys survive every reconcile;
- the update-fight loop disappears.

Covered by unit tests in `notebook-controller/tests/`.

## Provenance

- Source: https://github.com/kubeflow/kubeflow, branch `v1.10-branch`,
  import commit `90e987bf87d3e7c900926310b00bfa16b59e41eb` (see `UPSTREAM.md`).
- Layout: `notebook-controller/` + `common/` side by side, matching the
  upstream Docker build context (`components/`).
- `notebook-controller/go.mod` got a `replace` directive to `../common`:
  upstream pinned `common` as a module-proxy pseudo-version, which would have
  silently ignored the local fix. Verified `reconcilehelper` is identical
  between the pinned pseudo-version and `v1.10-branch` HEAD.

## Image

```
ghcr.io/ptishkin/notebook-controller:v1.10.0-patch.1
```

Deploy:

```bash
kubectl -n kubeflow set image deploy/notebook-controller-deployment \
  manager=ghcr.io/ptishkin/notebook-controller:v1.10.0-patch.1
```

Verify the fix:

```bash
kubectl -n <ns> annotate svc <notebook> test-annotation=1 --overwrite
sleep 60   # let 1-2 reconcile cycles pass
kubectl -n <ns> get svc <notebook> -o jsonpath='{.metadata.annotations}'
# test-annotation must still be there
```

Rollback: set the image back to `ghcr.io/kubeflow/kubeflow/notebook-controller:v1.10.0`.

## Syncing with upstream

`scripts/sync-upstream.sh` pulls changes from `kubeflow/kubeflow@v1.10-branch`
limited to `components/notebook-controller/**` and `components/common/**`,
commits them to `main`, and rebases the patch branch. The GitHub workflow
`.github/workflows/sync-upstream.yml` runs it weekly and on demand
(`workflow_dispatch`) and opens a PR when upstream moved.

The patch itself is also exported as `patches/0001-*.patch` for easy
porting to other versions (the same code exists in kubeflow/notebooks
`notebooks-v1` branch, used by Kubeflow 1.11+).

## License

Apache-2.0 (upstream). See `notebook-controller/third_party/license.txt`.
