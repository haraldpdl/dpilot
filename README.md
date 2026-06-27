# dpilot

dpilot orchestrates ordered groups of [ddev](https://github.com/ddev/ddev) projects.
A dpilot group is a named, ordered set of ddev projects defined at
`~/.dpilot/groups/<name>.yaml`. dpilot starts members in order (waiting for each
to be ready before the next) and stops them in reverse, driving ddev through its
CLI. It mirrors ddev's commands, aliases, flags, and output for familiarity.

```bash
dpilot create mystack
dpilot add mystack db-services
dpilot add mystack api
dpilot start mystack      # start each member in order, readiness-gated
dpilot describe mystack   # show members and live ddev state (alias: status)
dpilot stop mystack       # stop in reverse order
```

See `docs/specs/` for the design and `docs/plans/` for the implementation plan.
