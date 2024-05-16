# discussions-comment-notify-action

## usege

### 1. Create Mention Map File

Create a file named `.github/slack-mention-map.json` in the repository.

e.g.

```json
{
  "github-username": "slack-user-id"
}
```

### 2. Create GitHub Actions Workflow

```yaml
name: discussions-comment-notify-action

on:
  discussion_comment:
    types: [created]

jobs:
  notice:
    runs-on: ubuntu-latest

    steps:
      - name: Notify Slack
        uses: yhorikawa/discussions-comment-notify-action@main
        with:
          slack-api-token: ${{ secrets.SLACK_API_TOKEN }}
          slack-channel:  ${{ secrets.SLACK_CHANNEL }}
          github-token: ${{ secrets.GITHUB_TOKEN }}
```
