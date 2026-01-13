#!/bin/bash
# Reset main branch to exactly match card branch

# Discard all local changes
git reset --hard HEAD

# Fetch latest from remote
git fetch origin

# Reset main to match card exactly
git reset --hard origin/card

# Show status
echo "Main branch has been reset to match card branch"
git log --oneline -3

