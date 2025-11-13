# DayGo
...as in "Where did my day go?"

# Why?
I need a system to...
  1. offload TODOs from my brain onto "paper"
  2. minimize decision points so I can just focus on completing TODOs one after another

In other words, "Load 'em up and knock 'em out!"

After several attempts at bespoke TODO apps, I think the main failing is burdening the user (aka me) with too many decisions.

The ability to set priority levels and tags requires the user to make additional decisions when adding tasks and again when selecting their next task. This takes up a significant amount of mental energy, making these features counterproductive toward the ultimate goal of getting things done.

DayGo is yet another experiment: a first-in-first-out task list.

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
- An exception to the goal of minimizing features is the `/t <end_time>` command. I've found time blocking an effective countermeasure to my tendency to hyper-focus on a task at the expense of other priorities. This also allows ending a task at a predetermined time while AFK.
