# Comments

I tried to do it in a way that focuses on dependency injection, rather than singleton.

There is no error handling (visual UI) if the connection to the Kubernetes cluster is not correct.

There is currently no (visual) distinction between making a reference to parent, and referencing a parent that doesn't exist.
If we want this, then based on if the parent exist, we need to mark it and visualize it differently.