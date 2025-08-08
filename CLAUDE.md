I will provide a product specification, and you may begin work. Please remember the development guidelines that were provided when working through process around beliefs, simplicity, process, technology and decision framework, tooling, and testing. Remember to document your work in the IMPLEMENTATION_PLAN.md as you go.


EmailRSS Product Specification

## Purpose

I need a tool which creates an RSS Feed for my email so that I can read the RSS feed in my rss feed reader, which is more convenient than using an email client.

## Features

The product will support these features.

1. Support for multiple folder paths
2. One feed per folder is fine.
3. Internal code hooks for future AI message summarization support; can be stubbed for now.
4. Simple web server which serves the RSS feed for the messages.
5. Ability to run in a container in a kubernetes cluster.
6. It needs to keep track of the messages it has placed into the feed so that it does not repeat previously-fed messages as new messages again.
7. Use IMAP to retrieve message lists and messages with configurable user and token / password for authentication.


## Implementation Specifications

1. Use the alecthomas/kong library for CLI operations, when needed.
2. Use the knadh/koanf library for configuration file settings, when needed.
3. If a database is used, a sqlite installation is fine. However, if we use a database, its storage placement must be configurable via koanf.
4. For performance, let's use async for processing messages into rss feeds.
5. The ultimate RSS feed file should be stored on the filesystem at a configurable location. 
6. The RSS feed can be built on the fly as the server scans the email or it can be constructed from the database once the scan is complete. 
7. It should be possible using a CLI switch to reset the history for a given folder such that it is rebuilt completely from scratch if required.


