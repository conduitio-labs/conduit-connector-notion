# Change data capture (CDC)

This document is about capturing changes in a Notion source. 

## Problem statement
We need a way to fetch changes in a Notion source. That can include:
* new pages and databases
* updated pages and databases

## Background
To detect changes in Notion API, we rely on the `last_edited_time` property which can be found on `page`, `database`,
and `block` objects. `last_edited_time` is the only property and the only feature in the Notion API which deals with
changes on objects. `last_edited_time` is rounded down to the closest minute ([reference](https://developers.notion.com/changelog/last-edited-time-is-now-rounded-to-the-nearest-minute)).

Furthermore, the search API doesn't allow for filtering by this property, so filtering needs to happen in the connector. 
The connector gets a list of pages and filters out those which have the `last_edited_time` property **after** the last saved
position. 

##  Problem
In certain cases, detecting changes becomes a challenge. Let's assume a following timeline of events:

| Time     | Where     | Event                                                            |
|----------|-----------|------------------------------------------------------------------|
| 09:00:05 | Notion    | Page is created                                                  |
| 09:00:10 | Connector | Page is read, saved with position (`last_edited_time`) 09:00     |
| 09:00:20 | Notion    | Page is updated                                                  |
| 09:01:10 | Connector | Page is read, `last_edited_time` is still 09:00, so it's skipped |

In other words, because of the rounding down happening in Notion, a `hh:mm` time includes everything between 
`hh:mm:00` and `hh:mm:59`. If the connector is requesting changes between `hh:mm:00` and `hh:mm:59`, finds some, and saves 
the position as `hh:mm` that can obviously include future changes too.


## Option 1
The connector filters out those Notion objects which have the `last_edited_time` property **after or equal to** the last saved
position. E.g. let's assume that the connector is getting the pages at 09:01:10. It would look for objects which changed 
until 09:01 (and including it).

The consequence is that we might get duplicates. We could handle this, if the connector had a way to store the check sums
of objects it already read. Given that it has no access to a store where it can persist that information, it would need
to store that information in the position, making it potentially big.

## Option 2
The connector ignores changes which happened at `hh:mm`, until it's certain that `hh:mm:59` passed.

The consequence is that we might be waiting for changes longer than in option 1. We could limit this effect by checking
for changes on the minute (or slightly after that). E.g. let's assume that the connector is getting the pages at 09:01:10. 
It would look for objects which changed until 09:00 (and including it), but it would ignore objects which have `last_edited_time`
set to 09:01:00.

## Preferred option
The preferred option is option 2. The only disadvantage of it mentioned can be handled relatively easily and the effect 
of it is even less if the `poll_interval` is configured and set to something longer than a minute.
