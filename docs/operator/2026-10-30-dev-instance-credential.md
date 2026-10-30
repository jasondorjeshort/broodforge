# The development instance's credential

**Register yourself on the PC development instance (`dev`) and give it your subscription as the
credential labelled `dev`, logged in once in its own harness directory, so a later brief can run a real turn
there on your word.** It waits on you because the login is your subscription's, signed in in your
browser, and the Discord user id is yours. The step and its reasoning are owned by
`docs/design/05-debug-interface-dev-instance.md` (The owner's one step: the dev credential); the
instance itself by the same doc (The instance).

Runs as: you, under your own Windows account (no elevation), in Git Bash in `<repo>`, the main
checkout, with `bin/broodforge.exe` built. Time: about five minutes.

First: the dev root exists (an agent made it with the `run` skill's setup line; `ls
"$LOCALAPPDATA/Broodforge/dev/config"` lists `instance.json`) and the dev bot is running. When no
agent left it running, start it in a second Git Bash window in `<repo>` and leave that window open:

```
export BROODFORGE_ROOT="$LOCALAPPDATA/Broodforge/dev"; unset BROODFORGE_KEY_FILE
bin/broodforge.exe serve --discord dry-run
```

Then in the first window, the same `export` line first, and the four steps in order. `<id>` is
the user id Discord's "Copy User ID" gives you (Developer Mode on), so the registration carries
into the live form later; `<git name>` and `<git email>` are the name and email your commits carry:

```
export BROODFORGE_ROOT="$LOCALAPPDATA/Broodforge/dev"; unset BROODFORGE_KEY_FILE
bin/broodforge.exe status --expect ok
bin/broodforge.exe register <id> --name "<git name>" --email <git email>
bin/broodforge.exe credential add --kind subscription --user <id> --label dev
```

The last line prints the new id, `cred-<x>`. Then:

```
bin/broodforge.exe credential login cred-<x>
```

It prints one line, `CLAUDE_CONFIG_DIR=<root>/harness/cred-<x> <engine> auth login`. Paste that
line into the same window and sign in in the browser with the subscription (this is the one extra
login: your own `~/.claude` login is not reused); if the browser's callback fails, paste the code it
shows back into the window. Then:

```
bin/broodforge.exe credential login cred-<x> --done
bin/broodforge.exe register set-default <id> cred-<x>
bin/broodforge.exe credential list
bin/broodforge.exe stop
```

How to check it took: `credential list` shows `cred-<x>` as `subscription`, user `<id>`, `usable`;
`stop` prints `stopped  <n>` (the window running `serve` returns to its prompt), so the next agent's
first `status` finds no bot.

The dev store key `setup` printed guards no secret the harness directory does not already hold: it
goes in no password manager, and a lost dev root is remade by the `run` skill's setup line (the
login is then repeated).

When done: delete this file and its line in `README.md`; the coordinator notes in `docs/HANDOFF.md`
that the dev credential is `usable`.
