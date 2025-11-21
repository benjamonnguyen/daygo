# Guidelines
- i/o operations should be wrapped in tea.Cmd to be handled asynchronously by tea.Program.
- perform model updates optimistically. Assume i/o operations were successful.
