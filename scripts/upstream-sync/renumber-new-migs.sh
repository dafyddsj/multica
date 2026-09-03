#!/usr/bin/env bash
# After a cherry-pick that dropped upstream 437-449 migration files,
# rename each NEW file (not already in HEAD^) to the next free number >= 485
# and rewrite the basename in migrate/main.go.
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"
next=485
while ls server/migrations/${next}_*.up.sql >/dev/null 2>&1; do
	next=$((next + 1))
done

changed=0
for f in server/migrations/43[7-9]_*.up.sql server/migrations/44[0-9]_*.up.sql; do
	[ -f "$f" ] || continue
	base="$(basename "$f" .up.sql)"
	# skip if this basename existed before the pick
	if git cat-file -e "HEAD~1:server/migrations/${base}.up.sql" 2>/dev/null; then
		continue
	fi
	name="${base#*_}"
	old_num="${base%%_*}"
	new="${next}_${name}"
	git mv "server/migrations/${base}.up.sql" "server/migrations/${new}.up.sql"
	if [ -f "server/migrations/${base}.down.sql" ]; then
		git mv "server/migrations/${base}.down.sql" "server/migrations/${new}.down.sql"
	fi
	# rewrite references in staged/untracked tree
	rg -l --fixed-strings "$base" server || true
	if grep -Rql --fixed-strings "$base" server; then
		grep -RIl --fixed-strings "$base" server | while read -r path; do
			sed -i "s/${base}/${new}/g" "$path"
		done
	fi
	printf 'renumbered %s -> %s\n' "$base" "$new" >&2
	next=$((next + 1))
	changed=1
done
if [ "$changed" -eq 0 ]; then
	printf 'no new 437-449 migrations to renumber\n' >&2
fi
