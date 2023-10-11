# Bandits Schedule Twitter Notifier

This continually running script scrapes the Brookline Bandits 12U web page and sends out a tweet whenever it detects a difference in the schedule from the last time that it checked. The tweet will include a screenshot of the web page (via puppeteer).

## Prerequisites

1. Node.js 20.x. 
This was developed with NodeJS 20.x, but may work with other versions as well.
2. Twitter developer credentials with access to Twitter API (https://developer.twitter.com/en/docs/twitter-api/getting-started/getting-access-to-the-twitter-api)
   a. API Key
   b. API Secret
   c. Access Token Key
   d. Access Token Secret 

## Installation and Configuration

1. After cloning the repository, make sure that the node_modules are loaded.

```
  cd banditsNotification
  npm install
```

2. Create a `.env` file. This contains the environment variables for the Twitter API credentials. Copy the below into the file and update the values.
```
TWITTER_CONSUMER_KEY=<API Key>
TWITTER_CONSUMER_SECRET=<API Key Secret>
TWITTER_ACCESS_TOKEN_KEY=<Access Token>
TWITTER_ACCESS_TOKEN_SECRET=<Access Token Secret>
RUN_INTERVAL=300
TWITTER_USER_HANDLE=BlineBanditsBot
```
3. Run the script - it will post a tweet the first time because it doesn't detect any previous data in the `archive/` or local folder.

## Setting up `launchd` on a Mac
To use on a Mac system, do the following:

1. Create a new file `~/Library/LaunchAgents/com.harvardpan.banditsNotifications.plist`. Within the file, put this code:
```
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>Label</key>
        <string>com.harvardpan.banditsNotifications</string>
        <key>ProgramArguments</key>
        <array>
            <string>[FULL PATH OF node BINARY]</string>
            <string>[FULL PATH OF GIT REPOSITORY]/index.js</string>
        </array>
        <key>WorkingDirectory</key>
        <string>[FULL PATH OF GIT REPOSITORY]</string>
        <key>RunAtLoad</key>
        <true/>
        <key>StandardOutPath</key>
        <string>/tmp/com.harvardpan.banditsNotifications.out</string>
        <key>StandardErrorPath</key>
        <string>/tmp/com.harvardpan.banditsNotifications.err</string>
    </dict>
</plist>
```
Replace the bracketed full paths with the appropriate paths. 
2. Add the plist to be loaded by `launchctl`
```
sudo launchctl bootstrap gui/501 ~/Library/LaunchAgents/com.harvardpan.banditsNotifications.plist
```
To remove it from `launchctl`, use the `bootout` command.
```
sudo launchctl bootout gui/501 ~/Library/LaunchAgents/com.harvardpan.banditsNotifications.plist
```
Note that the `gui/501` should be the user id.
