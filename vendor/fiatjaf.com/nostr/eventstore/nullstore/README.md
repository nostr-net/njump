# NullStore

`nullstore` is a no-op implementation of the eventstore interface that doesn't actually store or retrieve any events.

All operations succeed without error but have no effect:
- `SaveEvent` and `ReplaceEvent` do nothing
- `QueryEvents` returns an empty iterator
- `DeleteEvent` does nothing
- `CountEvents` returns 0

This is useful for testing, development environments, or when event persistence is not required.
