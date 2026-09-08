#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

list="${1:?usage: pick-wave.sh <sha-list-file>}"
log="${2:-$root_dir/.audit/upstream-v0438/pick-log.tsv}"
guard="$root_dir/scripts/check-fork-overlay.sh"

if ! git diff --quiet || ! git diff --cached --quiet; then
	printf 'dirty tracked tree; commit or stash before pick-wave\n' >&2
	exit 1
fi

if [ ! -f "$log" ]; then
	printf 'sha\tsubject\tresult\tdetail\n' > "$log"
fi

while read -r raw extra; do
	[ -z "${raw:-}" ] && continue
	case "$raw" in \#*) continue ;; esac
	sha="$(git rev-parse --verify "${raw}^{commit}")"
	subject="$(git log -1 --format='%s' "$sha")"
	if git merge-base --is-ancestor "$sha" HEAD; then
		printf '%s\t%s\tskipped\talready on HEAD\n' "${sha:0:9}" "$subject" >> "$log"
		continue
	fi
	if git cherry-pick -x --allow-empty "$sha"; then
		if "$guard" >/tmp/overlay-out 2>/tmp/overlay-err; then
			printf '%s\t%s\tpicked\t\n' "${sha:0:9}" "$subject" >> "$log"
		else
			git reset --hard HEAD~1
			printf '%s\t%s\treverted\toverlay: %s\n' "${sha:0:9}" "$subject" "$(tr '\n' ' ' </tmp/overlay-err)" >> "$log"
		fi
	else
		detail="$(git diff --name-only --diff-filter=U | tr '\n' ',')"
		git cherry-pick --abort >/dev/null 2>&1 || git reset --hard HEAD
		printf '%s\t%s\tconflict\t%s\n' "${sha:0:9}" "$subject" "$detail" >> "$log"
	fi
done < "$list"

printf 'wrote %s\n' "$log"
cut -f3 "$log" | sort | uniq -c
