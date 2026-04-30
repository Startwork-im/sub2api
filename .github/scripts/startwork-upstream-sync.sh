#!/usr/bin/env bash

set -euo pipefail

: "${GITHUB_OUTPUT:=/tmp/startwork-upstream-sync-output}"
: "${GITHUB_STEP_SUMMARY:=/tmp/startwork-upstream-sync-summary}"
: "${REQUESTED_UPSTREAM_TAG:=}"
: "${UPSTREAM_REMOTE_NAME:=upstream}"
: "${UPSTREAM_REMOTE_URL:=https://github.com/Wei-Shaw/sub2api.git}"
: "${ORIGIN_REMOTE_NAME:=origin}"
: "${DRY_RUN:=false}"

log() {
  printf '[%s] %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*"
}

append_output() {
  printf '%s=%s\n' "$1" "$2" >>"$GITHUB_OUTPUT"
}

version_lt() {
  local left="$1"
  local right="$2"
  [ "$left" != "$right" ] && [ "$(printf '%s\n%s\n' "$left" "$right" | sort -V | tail -n 1)" = "$right" ]
}

ensure_remote() {
  git remote add "$UPSTREAM_REMOTE_NAME" "$UPSTREAM_REMOTE_URL" 2>/dev/null || git remote set-url "$UPSTREAM_REMOTE_NAME" "$UPSTREAM_REMOTE_URL"
}

resolve_target_tag() {
  local tag="$REQUESTED_UPSTREAM_TAG"
  if [ -z "$tag" ]; then
    tag="$(git tag -l 'v*' | sort -V | tail -n 1)"
  fi
  if [ -z "$tag" ]; then
    log "No upstream tag found"
    exit 1
  fi
  printf '%s\n' "$tag"
}

list_maintained_branches() {
  git for-each-ref --format='%(refname:short)' "refs/remotes/${ORIGIN_REMOTE_NAME}/upstream-v*" \
    | sed "s#^${ORIGIN_REMOTE_NAME}/##" \
    | sort -V
}

select_patch_source_branch() {
  local target_tag="$1"
  local branch=""
  while IFS= read -r candidate; do
    [ -z "$candidate" ] && continue
    local candidate_tag="${candidate#upstream-}"
    if version_lt "$candidate_tag" "$target_tag"; then
      branch="$candidate"
    fi
  done < <(list_maintained_branches)
  printf '%s\n' "$branch"
}

main() {
  local conflict_file
  conflict_file="${RUNNER_TEMP:-/tmp}/sub2api-upstream-sync-conflicts.txt"
  : >"$conflict_file"

  git config user.name 'startwork-sub2api-sync[bot]'
  git config user.email 'startwork-sub2api-sync@users.noreply.github.com'

  ensure_remote
  git fetch "$ORIGIN_REMOTE_NAME" --prune
  git fetch "$UPSTREAM_REMOTE_NAME" --tags --force

  local target_tag target_branch patch_source_branch branch_head patch_source_tag patch_commits patch_commit_count
  target_tag="$(resolve_target_tag)"
  target_branch="upstream-${target_tag}"
  patch_source_branch="$(select_patch_source_branch "$target_tag")"

  append_output target_tag "$target_tag"
  append_output target_branch "$target_branch"
  append_output patch_source_branch "$patch_source_branch"
  append_output status "pending"

  if git show-ref --verify --quiet "refs/remotes/${ORIGIN_REMOTE_NAME}/${target_branch}"; then
    branch_head="$(git rev-parse "refs/remotes/${ORIGIN_REMOTE_NAME}/${target_branch}")"
    append_output status "noop_existing_branch"
    append_output branch_head "$branch_head"
    append_output patch_commit_count "0"
    {
      echo "## Startwork upstream sync"
      echo
      echo "- Result: existing branch reused"
      echo "- Upstream tag: \`${target_tag}\`"
      echo "- Branch: \`${target_branch}\`"
    } >>"$GITHUB_STEP_SUMMARY"
    exit 0
  fi

  git checkout -B "$target_branch" "refs/tags/${target_tag}"

  patch_commit_count=0
  if [ -n "$patch_source_branch" ]; then
    patch_source_tag="${patch_source_branch#upstream-}"
    mapfile -t patch_commits < <(git rev-list --reverse "refs/tags/${patch_source_tag}..refs/remotes/${ORIGIN_REMOTE_NAME}/${patch_source_branch}")
    patch_commit_count="${#patch_commits[@]}"
    for commit in "${patch_commits[@]}"; do
      if ! git cherry-pick "$commit"; then
        git diff --name-only --diff-filter=U >"$conflict_file" || true
        git cherry-pick --abort || true
        append_output status "conflict"
        append_output conflict_commit "$commit"
        append_output conflict_file "$conflict_file"
        append_output patch_commit_count "$patch_commit_count"
        {
          echo "## Startwork upstream sync"
          echo
          echo "- Result: conflict"
          echo "- Upstream tag: \`${target_tag}\`"
          echo "- Branch: \`${target_branch}\`"
          echo "- Patch source: \`${patch_source_branch}\`"
          echo "- Conflict commit: \`${commit}\`"
        } >>"$GITHUB_STEP_SUMMARY"
        exit 0
      fi
    done
  fi

  branch_head="$(git rev-parse HEAD)"
  append_output branch_head "$branch_head"
  append_output patch_commit_count "$patch_commit_count"

  if [ "$DRY_RUN" != "true" ]; then
    git push "$ORIGIN_REMOTE_NAME" "$target_branch"
  fi

  append_output status "created"
  {
    echo "## Startwork upstream sync"
    echo
    echo "- Result: branch created"
    echo "- Upstream tag: \`${target_tag}\`"
    echo "- Branch: \`${target_branch}\`"
    echo "- Patch source: \`${patch_source_branch:-none}\`"
    echo "- Patch commit count: \`${patch_commit_count}\`"
    echo "- Branch head: \`${branch_head}\`"
  } >>"$GITHUB_STEP_SUMMARY"
}

main "$@"
