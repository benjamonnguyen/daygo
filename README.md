# DayGo
...as in "Where did my day go?"

# Why?
I need a system to...
  1. offload thoughts onto "paper" to reduce cognitive load
  2. keep me on track rather than bouncing from one unfinished task to another
  3. block out time to prevent myself from hyper-focusing on one task at the expense of other priorities
    - still unsolved... /t <duration> to set an auto-end timer?

After several attempts at bespoke TODO apps, I think the crux of the problem is minimizing the number of decisions to be made.

Priority levels, tags, and most features introduce more decision points and end up being counterproductive toward the goal of crossing tasks off of an ever-growing list.

This is yet another experiment: a first-in-first-out task list.

Rather than being presented with a big list, queued tasks are doled out one at a time in the order they were added.

Your only decisions are to do, skip, or discard the task.

Skipped tasks go to the end of the queue, effectively de-prioritizing them. And discards naturally prune the list.

This is NOT a planner nor a calendar and should not be used for tasks that have a definitive deadline or scheduled time.

# Mockup
```bash
[06:00] morning routine ───┐
[06:15] water plants
[06:27] did yoga
[07:15] ───────────────────┘
[08:20] work on daygo ─────────┐
[08:20] implement new feature
[11:00] fix bugs
[11:50] stuck on db bug
[12:00] ───────────────────────┘

> |
(ctrl+c to quit)
```

# Usage
`daygo`: start next queued task\
`daygo <task>`: start new task\
`daygo /a <task>`: add task to the queue\
`daygo /review \[date\]`: review completed tasks for target date; accepts date format "DD-MM\[-YYYY\]" or number of days ago.\

# Personal Notes
- Phrase tasks to have a clear stopping point and limited scope
- Make the last note a status (ex. "done and submitted hw!", "still need to clean the bathroom")
