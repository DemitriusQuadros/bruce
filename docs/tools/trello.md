# Trello Tools

Four tools for board and card management. Authentication uses a Trello API key plus a user token — no OAuth redirect flow required.

## Getting API credentials

### Step 1 — Get your API key

1. Go to the [Trello Power-Up Admin portal](https://trello.com/power-ups/admin).
2. Click **New** to create a Power-Up for your workspace.
3. Fill in the required fields (name, workspace, iframe URL can be a placeholder).
4. After creation, the **API key** is shown on the Power-Up's API key tab.

### Step 2 — Generate a user token

Open the following URL in your browser, replacing `YOUR_KEY` with your API key:

```text
https://trello.com/1/authorize?expiration=never&scope=read,write&response_type=token&key=YOUR_KEY
```

Click **Allow** and copy the token from the next page.

## Config snippet

```yaml
tools:
  trello:
    enabled: true
    api_key: "your-trello-api-key"
    api_token: "your-trello-user-token"
```

## Tools reference

| Tool | Description |
|---|---|
| `trello_board_list` | List all boards accessible to the token |
| `trello_card_create` | Create a card in a specific list |
| `trello_card_move` | Move a card to a different list |
| `trello_card_get` | Get details of a specific card |

**`trello_board_list` parameters**

No parameters required.

Returns: array of board objects with `id`, `name`, `url`, `closed`.

**`trello_card_create` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `list_id` | string | Yes | ID of the list to create the card in |
| `name` | string | Yes | Card name |
| `board_id` | string | No | Informational only — not used in the API call |
| `description` | string | No | Card description |
| `due_date` | string | No | Due date in date format (e.g. `2026-04-01`) |
| `labels` | array of strings | No | Label IDs to apply |

Returns: `card_id`, `url`, `created_at`.

**`trello_card_move` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `card_id` | string | Yes | ID of the card to move |
| `list_id` | string | Yes | ID of the destination list |

Returns: `card_id`, `list_id`, `moved_at`.

**`trello_card_get` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `card_id` | string | Yes | ID of the card to retrieve |

Returns: `name`, `description`, `due_date`, `labels`, `list_id`, `list_name`, `board_id`, `url`.

## Finding list and card IDs

**From the board URL:** Add `.json` to the end of any Trello board URL to get a JSON dump that includes all list IDs and card IDs.

```text
https://trello.com/b/BOARD_SHORT_ID/board-name.json
```

**Using `trello_board_list`:** Ask Bruce "List my Trello boards" — the response includes board IDs. From there you can navigate to `.json` for the specific board.

**From a card URL:** The short card ID appears in the URL:
```text
https://trello.com/c/CARD_SHORT_ID/card-title
```
Use this short ID with `trello_card_get`.

## Example prompts

- "List my Trello boards"
- "Create a card called 'Fix login page' in list 5e9f0a1b2c3d4e5f6a7b8c9d"
- "Move card abc123 to the Done list def456"
- "Get details for Trello card xyz789"

## Limitations

- Checklist items are not supported (read or write).
- Attachment upload is not supported.
- Boards and lists must be accessible to the token owner — private boards on workspaces not linked to the token will return errors.
- `trello_board_list` returns all boards accessible to the user; there is no filter by workspace.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| 401 Unauthorized | API key or token invalid | Re-generate the token via the authorize URL; verify the API key in Power-Up Admin |
| "list not found" / 404 on card create | `list_id` is incorrect | Use the board `.json` endpoint to find the correct list ID |
| "board not found" | Board is not accessible to the token | Confirm the board belongs to the workspace linked to the API key |
| Token expired | Token generated with `expiration=1hour` | Re-generate using `expiration=never` in the authorize URL |
| Empty board list | Token owner has no boards | Verify you authorized with the correct Trello account |
