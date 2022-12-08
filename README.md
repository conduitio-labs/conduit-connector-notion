# Conduit Connector for Notion
A [Conduit](https://conduit.io) connector for [Notion](https://www.notion.so).

## How to build?
Run `make build` to build the connector.

## Testing
Run `make test` to run all the unit tests. Run `make test-integration` to run the integration tests.

## Source
TBD

### Configuration

| name           | description                                                         | required | default value |
|----------------|---------------------------------------------------------------------|----------|---------------|
| `token`        | Internal integration token.                                         | true     | ""            |
| `pollInterval` | Interval at which we poll Notion for changes. A Go duration string. | false    | 1 minute      |

## Known Issues & Limitations
* Known issue A
* Limitation A

## Planned work
- [ ] Item A
- [ ] Item B
