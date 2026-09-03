#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

first_free_fork_migration=485
upstream_collision_globs=(
	server/migrations/43[7-9]_*.up.sql
	server/migrations/44[0-9]_*.up.sql
)

owned_by_fork() {
	git cat-file -e "HEAD~1:server/migrations/${1}.up.sql" 2>/dev/null
}

next="$first_free_fork_migration"
while ls server/migrations/${next}_*.up.sql >/dev/null 2>&1; do
	next=$((next + 1))
done

changed=0
for f in "${upstream_collision_globs[@]}"; do
	[ -f "$f" ] || continue
	base="$(basename "$f" .up.sql)"
	if owned_by_fork "$base"; then
		continue
	fi
	name="${base#*_}"
	new="${next}_${name}"
	git mv "server/migrations/${base}.up.sql" "server/migrations/${new}.up.sql"
	if [ -f "server/migrations/${base}.down.sql" ]; then
		git mv "server/migrations/${base}.down.sql" "server/migrations/${new}.down.sql"
	fi
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
