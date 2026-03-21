# slack

Post messages to Slack channels via Incoming Webhooks or the Slack Web API.

## Tools

| Tool | Description |
|------|-------------|
| `slack/post-message` | Post a plain text message to a channel |
| `slack/post-blocks` | Post a rich Block Kit message |

## Configuration

```bash
agent plugins config set slack webhookURL https://hooks.slack.com/services/...
# or
agent plugins config set slack apiToken xoxb-...
```
