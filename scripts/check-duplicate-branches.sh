#!/bin/bash
# Check for duplicate branches with similar functionality
# Usage: bash scripts/check-duplicate-branches.sh

set -euo pipefail

git fetch origin

echo "🔍 Checking for potential duplicate branches..."
echo ""

# Get list of feature branches (exclude main and HEAD)
BRANCHES=$(git branch -r | grep -v main | grep -v HEAD | sed 's|origin/||')

if [ -z "$BRANCHES" ]; then
  echo "✅ No feature branches found"
  exit 0
fi

DUPLICATES_FOUND=0

for branch in $BRANCHES; do
  # Skip if already merged to main
  if git merge-base --is-ancestor origin/$branch origin/main 2>/dev/null; then
    continue
  fi

  COMMIT_MSG=$(git log -1 --format=%B origin/$branch 2>/dev/null | head -1 || echo "")
  BRANCH_AGE=$(git log -1 --format=%ai origin/$branch 2>/dev/null | cut -d' ' -f1 || echo "")

  # Compare with other branches for similar keywords
  for other in $BRANCHES; do
    if [ "$branch" != "$other" ] && [ "$branch" \< "$other" ]; then
      # Skip if other already merged to main
      if git merge-base --is-ancestor origin/$other origin/main 2>/dev/null; then
        continue
      fi

      OTHER_MSG=$(git log -1 --format=%B origin/$other 2>/dev/null | head -1 || echo "")
      OTHER_AGE=$(git log -1 --format=%ai origin/$other 2>/dev/null | cut -d' ' -f1 || echo "")

      # Check for similar keywords in commit messages
      # Look for common architecture/infrastructure keywords
      for keyword in "feat" "fix" "structurizr" "import" "export" "sync" "diagram" "c4" "mermaid"; do
        if echo "$COMMIT_MSG" | grep -qi "$keyword" && \
           echo "$OTHER_MSG" | grep -qi "$keyword"; then

          # Additional check: if both mention same package/file pattern
          BRANCH_FILES=$(git diff --name-only origin/main...origin/$branch 2>/dev/null | cut -d'/' -f1,2 | sort -u | tr '\n' ' ')
          OTHER_FILES=$(git diff --name-only origin/main...origin/$other 2>/dev/null | cut -d'/' -f1,2 | sort -u | tr '\n' ' ')

          # Count overlaps
          OVERLAPS=$(comm -12 <(echo "$BRANCH_FILES" | tr ' ' '\n' | sort -u) <(echo "$OTHER_FILES" | tr ' ' '\n' | sort -u) | wc -l)

          if [ "$OVERLAPS" -gt 0 ]; then
            if [ "$DUPLICATES_FOUND" -eq 0 ]; then
              echo "⚠️  Potential Duplicates Found:"
              echo ""
              DUPLICATES_FOUND=1
            fi

            echo "📌 Similar work detected:"
            echo "   Branch 1: $branch ($BRANCH_AGE)"
            echo "             Commit: $COMMIT_MSG"
            echo "   Branch 2: $other ($OTHER_AGE)"
            echo "             Commit: $OTHER_MSG"
            echo "   Overlapping packages: $OVERLAPS"
            echo ""
          fi
        fi
      done
    fi
  done
done

if [ "$DUPLICATES_FOUND" -eq 0 ]; then
  echo "✅ No duplicate branches detected"
  exit 0
else
  echo "ℹ️  Review the branches above for potential duplication"
  echo "   Consider consolidating work or closing redundant branches"
  exit 0  # Not a hard error, just informational
fi
