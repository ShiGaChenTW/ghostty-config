#!/bin/sh
# Moves ghostty-config to a new version: writes it into the binary's default
# version, commits everything in the working tree as "vX.Y.Z" and tags X.Y.Z
# (no "v", like the earlier tags). With --push it also pushes the commit and the tag; the tag starts
# the release workflow, which builds the binaries for macOS and Linux and
# publishes the GitHub release that install.sh and the update check read.
#
#   ./bump_version.sh 1.0.0           that exact version
#   ./bump_version.sh patch           0.4.0 → 0.4.1 (the default)
#   ./bump_version.sh minor | major   0.4.0 → 0.5.0 | 1.0.0
#   ./bump_version.sh 1.0.0 --push    and publish it
set -eu

cd "$(dirname "$0")"
MAIN=cmd/ghostty-config/main.go

say() { printf '%s\n' "$*"; }
die() { printf '✗ %s\n' "$*" >&2; exit 1; }

usage() { sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; }

want="" push=0
for arg in "$@"; do
	case "$arg" in
	--push) push=1 ;;
	-h | --help) usage; exit 0 ;;
	-*) die "unknown option $arg (see --help)" ;;
	*) [ -z "$want" ] || die "give one version only"; want="$arg" ;;
	esac
done
[ -n "$want" ] || want=patch

semver='^[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*$'

# The released versions are the tags on GitHub; fetch them so the newest one
# is known even in a clone that never pulled them.
git fetch --quiet --tags origin 2>/dev/null || say "(could not reach origin; using the tags already here)"
current="$(git tag --list | grep -E "$semver" | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1 || true)"
if [ -z "$current" ]; then
	current="$(sed -n 's/^var version = "\(.*\)"$/\1/p' "$MAIN")"
fi
printf '%s\n' "$current" | grep -Eq "$semver" || die "cannot read the current version (got \"$current\")"

major="${current%%.*}"
rest="${current#*.}"
minor="${rest%%.*}"
patch="${rest#*.}"

case "$want" in
major) next="$((major + 1)).0.0" ;;
minor) next="$major.$((minor + 1)).0" ;;
patch) next="$major.$minor.$((patch + 1))" ;;
*)
	printf '%s\n' "$want" | grep -Eq "$semver" || die "\"$want\" is not a version like 1.0.0, nor major, minor or patch"
	next="$want"
	;;
esac

highest="$(printf '%s\n%s\n' "$current" "$next" | sort -t. -k1,1n -k2,2n -k3,3n | tail -n 1)"
[ "$next" != "$current" ] && [ "$highest" = "$next" ] || die "$next is not above the current version $current"
git rev-parse -q --verify "refs/tags/$next" >/dev/null && die "tag $next already exists"

branch="$(git rev-parse --abbrev-ref HEAD)"
[ "$branch" = "main" ] || say "(note: on branch $branch, not main)"

say "Checking the build and the tests…"
go build ./... || die "the build fails"
go test ./... >/dev/null || die "the tests fail: go test ./..."

sed "s/^var version = \".*\"$/var version = \"$next\"/" "$MAIN" >"$MAIN.tmp" && mv "$MAIN.tmp" "$MAIN"
grep -q "^var version = \"$next\"$" "$MAIN" || die "could not write the version into $MAIN"

# The tag points at the code it names: everything goes into this commit.
git add -A
git commit --quiet -m "v$next"
git tag -a "$next" -m "v$next"
say "Version $current → $next: everything committed as v$next, tagged $next."
git show --stat --format= HEAD | tail -n 1

if [ "$push" -eq 0 ]; then
	say "Nothing is pushed yet. To publish, push the commit and the tag:"
	say "  git push origin $branch $next"
	exit 0
fi

git push --quiet origin "$branch" "$next"
say "Pushed $branch and $next. GitHub Actions now builds the binaries and publishes the release:"
say "  https://github.com/vitruves/ghostty-config/actions"
