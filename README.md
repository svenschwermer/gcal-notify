# gcal-notify

Google Calendar Notifier service. This service runs in the user session and
periodically checks a Google Calendar. Once it finds a notification that is
ready to be fired, it will do so via the [D-BUS API][1]. This requires a
notification daemon, like [mako][2].

## Setup
1. Download [API client credentials][3] and copy them to
    `~/.config/gcal-notify/client-secret.json`
2. Run `gcal-notify auth`
3. Configure the calendar ID, see [Configuration](#configuration)

## Configuration
The location of the configuration file is `~/.config/gcal-notify/config.toml` by
default. This can be changed via the command line parameter `-config`.

- `CalendarID`: Calendar identifier, typically an email address (required)
- `ClientSecretPath`: API credentials file (optional,
    default=`~/.config/gcal-notify/client-secret.json`)
- `TokenPath`: OAuth2 token file path (optional,
    default=`~/.cache/gcal-notify/token.json`)
- `PollInterval`: Interval at which the Google Calendar API is polled (optional,
    default=`3m`)
- `LookaheadInterval`: Longest possible notification duration (optional,
    default=`24h`)
- `LocationPollInterval`: Interval at which the Google Calendar API is polled
    for working location events (optional, default=`15m`)
- `SlackTokenFile`: Slack token file (optional,
    default=`~/.config/gcal-notify/slack-token`)

[1]:https://specifications.freedesktop.org/notification-spec/latest/ar01s09.html
[2]:https://wayland.emersion.fr/mako/
[3]:https://console.cloud.google.com/apis/api/calendar-json.googleapis.com/credentials
