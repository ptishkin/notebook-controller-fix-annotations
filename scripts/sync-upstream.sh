#!/usr/bin/env bash
# Sync this repository with upstream kubeflow/kubeflow v1.10-branch,
# limited to the paths this repo mirrors.
#
# Usage:
#   scripts/sync-upstream.sh            # check + apply + commit on main
#   scripts/sync-upstream.sh --check    # only report whether upstream moved
set -euo pipefail

UPSTREAM_REPO="https://github.com/kubeflow/kubeflow"
UPSTREAM_BRANCH="v1.10-branch"
# Paths in the upstream monorepo this repo mirrors:
UPSTREAM_PATHS=("components/notebook-controller" "components/common")
# Where they live in this repo (same order as UPSTREAM_PATHS):
LOCAL_PATHS=("notebook-controller" "common")

cd "$(git rev-parse --show-toplevel)"

echo ">> Fetching ${UPSTREAM_REPO} ${UPSTREAM_BRANCH}..."
git fetch "${UPSTREAM_REPO}" "${UPSTREAM_BRANCH}" --no-tags
UP_SHA="$(git rev-parse FETCH_HEAD)"
echo ">> Upstream HEAD: ${UP_SHA}"

LAST_IMPORT="$(grep -m1 '^Commit:' UPSTREAM.md | awk '{print $2}')"
echo ">> Last imported upstream commit: ${LAST_IMPORT}"

if [ "${LAST_IMPORT}" = "${UP_SHA}" ]; then
    echo ">> Already up to date."
    exit 0
fi

CHANGED=0
for i in "${!UPSTREAM_PATHS[@]}"; do
    up="${UPSTREAM_PATHS[$i]}"
    local_path="${LOCAL_PATHS[$i]}"
    if git diff --quiet "${LAST_IMPORT}" "${UP_SHA}" -- "${up}"; then
        echo ">> ${up}: no changes"
    else
        echo ">> ${up}: has changes, copying..."
        rm -rf "${local_path}"
        # git archive from the fetched commit keeps it exact and blob-free
        git archive "${UP_SHA}" "${up}" | tar -x -C /tmp
        cp -R "/tmp/${up}" "${local_path}"
        rm -rf "/tmp/$(basename "${up}")"
        CHANGED=1
    fi
done

if [ "${CHANGED}" -eq 0 ]; then
    echo ">> Upstream moved but none of the mirrored paths changed. Updating UPSTREAM.md only."
fi

# Refresh provenance marker
sed -i.bak "s/^Commit: .*/Commit: ${UP_SHA}/" UPSTREAM.md && rm -f UPSTREAM.md.bak

if [ "${CHANGED}" -eq 0 ] && [ "${1:-}" != "--check" ]; then
    git add UPSTREAM.md
    git -c user.name="sync-bot" -c user.email="sync@noreply" \
        commit -m "sync: upstream pointer update ${UP_SHA}" --quiet
    echo ">> UPSTREAM.md updated."
    exit 0
fi

if [ "${1:-}" = "--check" ]; then
    echo ">> Upstream has changes in mirrored paths (not applied, ran with --check)."
    exit 1
fi

git add -A
git -c user.name="sync-bot" -c user.email="sync@noreply" \
    commit -m "sync: upstream ${UPSTREAM_BRANCH} ${UP_SHA}

Mirrored paths updated from kubeflow/kubeflow ${UPSTREAM_BRANCH}.
Next: rebase patch/merge-annotations onto this commit and re-run tests." --quiet
echo ">> Sync commit created on $(git rev-parse --abbrev-ref HEAD)."
echo ">> Now rebase the patch branch:"
echo "     git checkout patch/merge-annotations && git rebase main"
echo "     docker run --rm -v \$PWD:/src -w /src/notebook-controller golang:1.17 sh -c 'go test ./tests/ && go vet ./...'"
