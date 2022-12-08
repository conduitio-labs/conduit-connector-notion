# Conduit Connector for Notion
A [Conduit](https://conduit.io) connector for [Notion](https://www.notion.so).

## How to build?
Run `make build` to build the connector.

## Testing
Run `make test` to run all the unit tests. Run `make test-integration` to run the integration tests.

## Source
The source connector is able to read new and updated pages in a Notion workspace. Note that this works only for pages
that are accessible to the Notion integration used with this connector. 

The records produced by this connector will contain a plain text representation of pages read.

### Configuration

| name           | description                                                                                                     | required | default value |
|----------------|-----------------------------------------------------------------------------------------------------------------|----------|---------------|
| `token`        | A token to be used for authorizing requests to Notion. Can be an internal integration or an OAuth access token. | true     | ""            |
| `pollInterval` | Interval at which we poll Notion for changes. A Go duration string. Cannot be shorter than 1 minute.            | false    | 1 minute      |

## Known Issues & Limitations
* Currently, only pages are supported.

## Planned work
- [ ] Support databases
- [ ] Support comments
