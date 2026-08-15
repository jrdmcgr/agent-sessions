# Improve Multi-agent Flow

This project was the first where I used Fable to plan and Sonnet (no thinking) to build. 
It used a top-level Sonnet agent to run certain tasks in parallel. 

Shelling out to subagents was done similar to the following by the agent managing the process.
```
pi -p --no-session \
      --model claude-sonnet-5 \
      --thinking off \
"Read @tasks/01-xxx.md and follow the instructions. Make changes to code in a worktree. Commit your work when done. Do not work on anything else."
```

I would like to explore some improvements with this approach.

1. Because this was worked on in a subprocess with `--no-session` the cost / duration etc wasn't included.

- Can I reuse the current session for this?
- Pi supports session trees or threaded sessions. Can I take advantage or this?
- Would forking the session be practical, or would this double the token overhead for each subagent?

2. Can we wrap up the subagent task runner in a skill, so that I can just give a task as an argument and it will acheive the right thing.
3. Can we take advantage of `herder`'s cli to open a new tab or pane in the current session?

