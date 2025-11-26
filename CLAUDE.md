# daygo
- i/o operations should be wrapped in tea.Cmd to be handled asynchronously by tea.Program.
- perform model updates optimistically - assume i/o operations were successful.
- new tasks are created and updated in memory and only persisted to db when ended.

# General architecture
- Repository layer interfaces with storage layer for data access
- Service layer interfaces with repository layer with added business logic

# Repository layer
- Record contains fields for initial creation of entity
- ExistingRecord contains auto-populated fields like ID and CreatedAt
