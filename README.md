# DayGo
...as in "Where did my day go?"

# Why?
I need a system to...
  1. offload TODOs onto "paper" to reduce cognitive load
  2. keep me on track rather than bouncing from one unfinished TODO to another
  3. block out time to prevent myself from hyper-focusing on one task at the expense of other priorities

After several attempts at bespoke TODO apps, I think the crux of the problem is minimizing the number of decisions to be made.

Priority levels, tags, and most features introduce more decision points and end up being counterproductive toward the ultimate goal of crossing tasks off of a list (ideally faster than it can grow).

This is yet another experiment: a first-in-first-out task list.

Rather than being presented with the list, queued tasks are doled out one at a time in the order they were added.

Your only decisions are to do, skip, or discard the task.

Prioritization and pruning are therefore naturally baked into this simple decision flow.

What this tool is NOT meant to be is a planner or calendar and should not be used for tasks that have a definitive deadline or scheduled time.

# Mockup
```bash
[06:00] morning routine ───┐
[06:15] watered plants
[06:27] yoga
───────────────────────────┘
[07:10] daygo: implement new feature ───┐
[07:30] pushed!
[07:30] ────────────────────────────────┘

> |
(ctrl+c to quit)
```

# Usage
`daygo`: start next queued task\
`daygo <task>`: start new task\
`daygo /a <task>`: add task to the queue\
`daygo /r [days_ago]`: review tasks for date some number of days ago (default 0)`

# Personal Notes
- Phrase tasks to have a clear stopping point and limited scope
- End tasks with status note (ex. "submitted assignment", "blocked on concurrency bug")
