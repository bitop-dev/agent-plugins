# send-email

Draft and send emails via SMTP. Useful for agents that need to deliver
reports, summaries, or notifications.

## Tools

| Tool | Description |
|------|-------------|
| `email/draft` | Draft an email (returns preview without sending) |
| `email/send` | Send an email via configured SMTP server |

## Configuration

```bash
agent plugins config set send-email provider smtp
agent plugins config set send-email smtpHost smtp.gmail.com
agent plugins config set send-email smtpPort 587
agent plugins config set send-email username user@gmail.com
agent plugins config set send-email password app-password
agent plugins config set send-email fromAddress user@gmail.com
```
