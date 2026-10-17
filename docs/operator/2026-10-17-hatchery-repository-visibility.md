**Decide whether the Hatchery repository on GitHub goes private until the history rewrite has landed, and if so change the setting.** It waits on the owner because the visibility toggle is a setting in the GitHub console under the owner's own account, and because the choice is theirs. The reasoning is in `docs/findings/2026-10-17-personal-data-leak-protocol-gap.md` (Result 4, Next) and ADR 0018, whose paragraph on the owner's separate steps names "whether the Hatchery repository goes private until that rewrite has landed".

## The fact to check first

GitHub reported the repository public on 2026-10-17, with no forks, while a sibling's records say it is private. Confirm what is true today before acting:

```
gh repo view <owner>/<hatchery-repo> --json visibility,forkCount,stargazerCount
```

Account: the owner's GitHub login, any shell with `gh` authenticated. What the public state means: the 28 root commits carry a third party's name and email, and 25 messages carry session links. Anyone who cloned it while it was public holds them; making it private stops new copies, it does not recall old ones. Not determined: how long it has been public and whether anyone cloned it.

## Decide (the owner's)

1. Go private now, before the rewrite, or leave it public until the rewrite is pushed.
2. The order against `docs/operator/2026-10-17-rewrite-root-commit-identity.md`: private first is the safer order, because the force-push and the rewrite then happen out of public view.
3. Whether to make it public again afterwards. The siblings' clones and any project decanted from it use the remote's URL; check that they still reach it after the change (private repositories need their credentials).

## Sketch (untested, an agent can refine it; nothing here was run)

```
gh repo edit <owner>/<hatchery-repo> --visibility private --accept-visibility-change-consequences
# check it took
gh repo view <owner>/<hatchery-repo> --json visibility
```

The same switch is in the repository's Settings page, section Danger Zone, Change visibility. About two minutes.

## After it

Making the repository private does not remove the history. The rewrite entry above does that; the GitHub purge of cached views and old shas is the last item there.

## When done

Delete this file once the visibility is as decided and the finding's Next (the owner's visibility item) and `docs/HANDOFF.md`'s Waiting on user say so.
