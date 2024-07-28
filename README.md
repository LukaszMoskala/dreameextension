# Dreame extensions (for valetudo)

This implements dreame-specific features that I think are cool. It's designed to work with homeassistant and valetudo.

**This project is not related to valetudo in any way.**

So far only tested with my two dreame L10S ultra robots. Let me know if you tested it with another robot and it works.
Or if it doesn't, but it's possible to make it work with small changes.

## Supported functions
 - Send images of obstacles to homeassistant
   - apparently dreame saves pictures of obstacles to `/data/record/*.jpg`.
     I watch this directory for changes and publish contents of said files to mqtt.
   - autodiscovery is supported
 - Play sound when obstacle is detected
   - kind of inspired by roborock-oucher, but since I didn't find any log file that has entries when obstacle is hit,
     I decided to play sounds when obstacle is detected.
   - Put WAV files to `/data/dreameextension/obstacleVoices`, one will be picked at random every time new obstacle is detected.
   - Johnny Silverhand's voiceover lines are great for that
 - Play sound from mqtt
   - send zstd-compressed WAV file via mqtt
   - `zstd < filename.wav | mosquitto_pub -d -h mqtt.server.ip -u valetudo -P password -t valetudo/identifier/dreameextension/play -s`
 - No configuration needed (mqtt configuration is read from `/data/valetudo_config.json`)

## TODO
 - TLS certificate authentication to mqtt
 - Provide pre-built binaries for all architectures supported by valetudo
    - aarch64
    - armv7
 - Send command over mqtt to play local (local to robot) file
 - Send command over mqtt to play sound from URL
 - configurable autodiscovery prefix
 - Accept environment variables for things
   - `aplay` arguments
   - valetudo config location
   - override options from valetudo config with env vars

I may or may not be motivated enough to do anything from this list. If you are interested in some of said features,
please open issue for that. And preferably also a pull request :)

## Disclaimer
I put this together in less than 6 hours, I don't guarantee that it will work for you at all.

Again, **This project is not related to valetudo in any way.** I take no responsibility for what you do with this
software, broken robots or anything. Use at your own risk.

## Installation

Basically it comes down to: build, copy to robot, run.

After you enter cloned repo and installed go:
```shell
make #compile
ssh root@vacuum.ip 'mkdir -p /data/dreameextension/obstacleVoices ; cat - > /data/dreameextension/dreameextension; chmod +x /data/dreameextension/dreameextension' < dreameextension_linux_arm64 #copy to robot
```

Now, ssh to your robot and run:
```shell
nohup /data/dreameextension/dreameextension &
```

To make it auto start, edit `/data/_root_postboot.sh` and add

```shell
if [[ -f /data/dreameextension/dreameextension ]]; then
	/data/dreameextension/dreameextension > /tmp/dreameextension.log 2>&1 &
fi
```
At the end of file.

To stop, just run `killall dreameextension`.