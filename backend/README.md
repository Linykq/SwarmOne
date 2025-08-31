# SwarmOne Go Backend (no Docker)

Single binary backend for SwarmOne. Frontend stays unchanged.
- Fan-out to multiple LLM providers concurrently via goroutines.
- Majority vote consensus.
- No Docker, no Python.

## Run

```bash
cd SwarmOne-go-backend
export OPENAI_API_KEY=...
export GOOGLE_API_KEY=...
export ANTHROPIC_API_KEY=...

go mod tidy
go run ./cmd/swarmoned
# server on :8080
```

Frontend continues calling `http://localhost:8080/v1/ask`.

### Health
```bash
curl http://localhost:8080/health
```

### Ask (example)
```bash
curl -s http://localhost:8080/v1/ask -H "Content-Type: application/json" -d '{
  "template_id": "task.reply.email.v1",
  "instruction": "finish the task as following\n{\"Task\":\"reply email\", \"Content\":\"Please confirm one slot.\", \"Expections\":\"Professional; concise\", \"Source\":\"Meeting options: Tue 10:00 or Wed 14:00\", \"Language\":\"en-US\"}"
}'
```
