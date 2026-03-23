# Notion Tools

Three tools for reading, creating, and updating Notion pages and databases. Authentication uses a Notion internal integration token — no OAuth redirect flow required.

## Creating an integration

1. Go to [notion.so](https://www.notion.so) → **Settings & members → Integrations**.
2. Click **New integration**.
3. Give it a name (e.g. "Bruce"), select the associated workspace, and set **Content capabilities** to **Read content**, **Update content**, and **Insert content**.
4. Click **Submit** and copy the **Internal Integration Token** (starts with `secret_`).

## Config snippet

```yaml
tools:
  notion:
    enabled: true
    api_token: "secret_your_integration_token_here"
```

## Sharing pages with Bruce

Notion pages and databases are private by default. You must explicitly share each page or database with the integration before Bruce can access it:

1. Open the page or database in Notion.
2. Click **Share** (top-right corner).
3. Under **Connections**, search for and select your integration (the name you gave it in step 2 above).
4. Click **Invite**.

Any page that is not shared with the integration will return an "object not found" error — Notion returns 404 rather than 403 for unshared resources.

## Tools reference

| Tool | Description |
|---|---|
| `notion_read` | Read a page or query a database |
| `notion_create` | Create a new page |
| `notion_update` | Update properties on an existing page |

**`notion_read` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `page_id` | string (UUID) | Yes | Notion page or database ID |
| `query` | string | No | Filter query for database reads |

Returns: `page_id`, `title`, `url`, `properties`, `content`.

Note: `notion_read` first attempts to read the object as a page. If that fails, it automatically falls back to a database query. You do not need to specify whether the ID is a page or a database.

**`notion_create` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `parent_id` | string | Yes | ID of the parent page or database |
| `title` | string | Yes | Page title |
| `properties` | object | No | Additional Notion properties in Notion API format |

Returns: `page_id`, `title`, `url`, `created_at`.

**`notion_update` parameters**

| Parameter | Type | Required | Description |
|---|---|---|---|
| `page_id` | string | Yes | ID of the page to update |
| `properties` | object | Yes (non-empty) | Properties to update in Notion API format |

Returns: `page_id`, `updated_at`.

## Finding page IDs

The page ID is the last 32-character hexadecimal string in a Notion page URL:

```text
https://www.notion.so/My-Page-Title-1234567890abcdef1234567890abcdef
                                     ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                                     This is the page ID (with hyphens in UUID form)
```

You can also use the page ID with hyphens as a UUID: `12345678-90ab-cdef-1234-567890abcdef`.

## Example prompts

- "Read my project notes page at ID 1234567890abcdef1234567890abcdef"
- "Create a new page called 'Meeting Notes' in my workspace database"
- "Update the Status property on page abc123 to 'Done'"

## Limitations

- `notion_update` updates page **properties** only — it does not write block content (paragraphs, bullets, etc.). For appending content, use `notion_create` to create a sub-page.
- Each page must be individually shared with the integration — sharing a parent page does not grant access to child pages.
- `notion_create` creates a page with title and optional properties; block-level content must be added via the Notion UI or additional API calls not covered by these tools.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| "Could not find object" (404) | Page not shared with the integration | Open the page in Notion → Share → Connections → add the integration |
| 401 Unauthorized | Invalid integration token | Re-copy the token from Notion → Settings → Integrations |
| "properties must be non-empty" | `notion_update` called without properties | Include at least one property key in the `properties` object |
| Property type error | Properties object uses wrong Notion type format | Check [Notion API docs](https://developers.notion.com/reference/property-value-object) for the correct property value format |
| Database query returns empty | No pages match filter or database not shared | Verify the database is shared with the integration and the filter syntax is correct |
