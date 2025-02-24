<!--BEGIN_BANNER_IMAGE-->

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="/.github/banner_dark.png">
  <source media="(prefers-color-scheme: light)" srcset="/.github/banner_light.png">
  <img style="width:100%;" alt="The LiveKit icon, the name of the repository and some sample code in the background." src="https://raw.githubusercontent.com/conkit/conkit/main/.github/banner_light.png">
</picture>

<!--END_BANNER_IMAGE-->

# LiveKit: Real-time video, audio and data for developers

[LiveKit](https://conkit.io) is an open source project that provides scalable, multi-user conferencing based on WebRTC.
It's designed to provide everything you need to build real-time video audio data capabilities in your applications.

LiveKit's server is written in Go, using the awesome [Pion WebRTC](https://github.com/pion/webrtc) implementation.

[![GitHub stars](https://img.shields.io/github/stars/conkit/conkit?style=social&label=Star&maxAge=2592000)](https://github.com/a4to/conkit/stargazers/)
[![Slack community](https://img.shields.io/endpoint?url=https%3A%2F%2Fconkit.io%2Fbadges%2Fslack)](https://conkit.io/join-slack)
[![Twitter Follow](https://img.shields.io/twitter/follow/conkit)](https://twitter.com/conkit)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/conkit/conkit)](https://github.com/a4to/conkit/releases/latest)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/conkit/conkit/buildtest.yaml?branch=master)](https://github.com/a4to/conkit/actions/workflows/buildtest.yaml)
[![License](https://img.shields.io/github/license/conkit/conkit)](https://github.com/a4to/conkit/blob/master/LICENSE)

## Features

-   Scalable, distributed WebRTC SFU (Selective Forwarding Unit)
-   Modern, full-featured client SDKs
-   Built for production, supports JWT authentication
-   Robust networking and connectivity, UDP/TCP/TURN
-   Easy to deploy: single binary, Docker or Kubernetes
-   Advanced features including:
    -   [speaker detection](https://docs.conkit.io/home/client/tracks/subscribe/#speaker-detection)
    -   [simulcast](https://docs.conkit.io/home/client/tracks/publish/#video-simulcast)
    -   [end-to-end optimizations](https://blog.conkit.io/conkit-one-dot-zero/)
    -   [selective subscription](https://docs.conkit.io/home/client/tracks/subscribe/#selective-subscription)
    -   [moderation APIs](https://docs.conkit.io/home/server/managing-participants/)
    -   end-to-end encryption
    -   SVC codecs (VP9, AV1)
    -   [webhooks](https://docs.conkit.io/home/server/webhooks/)
    -   [distributed and multi-region](https://docs.conkit.io/home/self-hosting/distributed/)

## Documentation & Guides

https://docs.conkit.io

## Live Demos

-   [LiveKit Meet](https://meet.conkit.io) ([source](https://github.com/a4to-examples/meet))
-   [Spatial Audio](https://spatial-audio-demo.conkit.io/) ([source](https://github.com/a4to-examples/spatial-audio))
-   Livestreaming from OBS Studio ([source](https://github.com/a4to-examples/livestream))
-   [AI voice assistant using ChatGPT](https://conkit.io/kitt) ([source](https://github.com/a4to-examples/kitt))

## Ecosystem

-   [Agents](https://github.com/a4to/agents): build real-time multimodal AI applications with programmable backend participants
-   [Egress](https://github.com/a4to/egress): record or multi-stream rooms and export individual tracks
-   [Ingress](https://github.com/a4to/ingress): ingest streams from external sources like RTMP, WHIP, HLS, or OBS Studio

## SDKs & Tools

### Client SDKs

Client SDKs enable your frontend to include interactive, multi-user experiences.

<table>
  <tr>
    <th>Language</th>
    <th>Repo</th>
    <th>
        <a href="https://docs.conkit.io/home/client/events/#declarative-ui" target="_blank" rel="noopener noreferrer">Declarative UI</a>
    </th>
    <th>Links</th>
  </tr>
  <!-- BEGIN Template
  <tr>
    <td>Language</td>
    <td>
      <a href="" target="_blank" rel="noopener noreferrer"></a>
    </td>
    <td></td>
    <td></td>
  </tr>
  END -->
  <!-- JavaScript -->
  <tr>
    <td>JavaScript (TypeScript)</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-js" target="_blank" rel="noopener noreferrer">client-sdk-js</a>
    </td>
    <td>
      <a href="https://github.com/a4to/conkit-react" target="_blank" rel="noopener noreferrer">React</a>
    </td>
    <td>
      <a href="https://docs.conkit.io/client-sdk-js/" target="_blank" rel="noopener noreferrer">docs</a>
      |
      <a href="https://github.com/a4to/client-sdk-js/tree/main/example" target="_blank" rel="noopener noreferrer">JS example</a>
      |
      <a href="https://github.com/a4to/client-sdk-js/tree/main/example" target="_blank" rel="noopener noreferrer">React example</a>
    </td>
  </tr>
  <!-- Swift -->
  <tr>
    <td>Swift (iOS / MacOS)</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-swift" target="_blank" rel="noopener noreferrer">client-sdk-swift</a>
    </td>
    <td>Swift UI</td>
    <td>
      <a href="https://docs.conkit.io/client-sdk-swift/" target="_blank" rel="noopener noreferrer">docs</a>
      |
      <a href="https://github.com/a4to/client-example-swift" target="_blank" rel="noopener noreferrer">example</a>
    </td>
  </tr>
  <!-- Kotlin -->
  <tr>
    <td>Kotlin (Android)</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-android" target="_blank" rel="noopener noreferrer">client-sdk-android</a>
    </td>
    <td>Compose</td>
    <td>
      <a href="https://docs.conkit.io/client-sdk-android/index.html" target="_blank" rel="noopener noreferrer">docs</a>
      |
      <a href="https://github.com/a4to/client-sdk-android/tree/main/sample-app/src/main/java/io/conkit/android/sample" target="_blank" rel="noopener noreferrer">example</a>
      |
      <a href="https://github.com/a4to/client-sdk-android/tree/main/sample-app-compose/src/main/java/io/conkit/android/composesample" target="_blank" rel="noopener noreferrer">Compose example</a>
    </td>
  </tr>
<!-- Flutter -->
  <tr>
    <td>Flutter (all platforms)</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-flutter" target="_blank" rel="noopener noreferrer">client-sdk-flutter</a>
    </td>
    <td>native</td>
    <td>
      <a href="https://docs.conkit.io/client-sdk-flutter/" target="_blank" rel="noopener noreferrer">docs</a>
      |
      <a href="https://github.com/a4to/client-sdk-flutter/tree/main/example" target="_blank" rel="noopener noreferrer">example</a>
    </td>
  </tr>
  <!-- Unity -->
  <tr>
    <td>Unity WebGL</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-unity-web" target="_blank" rel="noopener noreferrer">client-sdk-unity-web</a>
    </td>
    <td></td>
    <td>
      <a href="https://conkit.github.io/client-sdk-unity-web/" target="_blank" rel="noopener noreferrer">docs</a>
    </td>
  </tr>
  <!-- React Native -->
  <tr>
    <td>React Native (beta)</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-react-native" target="_blank" rel="noopener noreferrer">client-sdk-react-native</a>
    </td>
    <td>native</td>
    <td></td>
  </tr>
  <!-- Rust -->
  <tr>
    <td>Rust</td>
    <td>
      <a href="https://github.com/a4to/client-sdk-rust" target="_blank" rel="noopener noreferrer">client-sdk-rust</a>
    </td>
    <td></td>
    <td></td>
  </tr>
</table>

### Server SDKs

Server SDKs enable your backend to generate [access tokens](https://docs.conkit.io/home/get-started/authentication/),
call [server APIs](https://docs.conkit.io/reference/server/server-apis/), and
receive [webhooks](https://docs.conkit.io/home/server/webhooks/). In addition, the Go SDK includes client capabilities,
enabling you to build automations that behave like end-users.

| Language                | Repo                                                                                    | Docs                                                        |
| :---------------------- | :-------------------------------------------------------------------------------------- | :---------------------------------------------------------- |
| Go                      | [server-sdk-go](https://github.com/a4to/server-sdk-go)                               | [docs](https://pkg.go.dev/github.com/a4to/server-sdk-go) |
| JavaScript (TypeScript) | [server-sdk-js](https://github.com/a4to/server-sdk-js)                               | [docs](https://docs.conkit.io/server-sdk-js/)              |
| Ruby                    | [server-sdk-ruby](https://github.com/a4to/server-sdk-ruby)                           |                                                             |
| Java (Kotlin)           | [server-sdk-kotlin](https://github.com/a4to/server-sdk-kotlin)                       |                                                             |
| Python (community)      | [python-sdks](https://github.com/a4to/python-sdks)                                   |                                                             |
| PHP (community)         | [agence104/conkit-server-sdk-php](https://github.com/agence104/conkit-server-sdk-php) |                                                             |

### Tools

-   [CLI](https://github.com/a4to/conkit-cli) - command line interface & load tester
-   [Docker image](https://hub.docker.com/r/conkit/conkit-server)
-   [Helm charts](https://github.com/a4to/conkit-helm)

## Install

> [!TIP]
> We recommend installing [LiveKit CLI](https://github.com/a4to/conkit-cli) along with the server. It lets you access
> server APIs, create tokens, and generate test traffic.

The following will install LiveKit's media server:

### MacOS

```shell
brew install conkit
```

### Linux

```shell
curl -sSL https://get.conkit.io | bash
```

### Windows

Download the [latest release here](https://github.com/a4to/conkit/releases/latest)

## Getting Started

### Starting LiveKit

Start LiveKit in development mode by running `conkit-server --dev`. It'll use a placeholder API key/secret pair.

```
API Key: devkey
API Secret: secret
```

To customize your setup for production, refer to our [deployment docs](https://docs.conkit.io/deploy/)

### Creating access token

A user connecting to a LiveKit room requires an [access token](https://docs.conkit.io/home/get-started/authentication/#creating-a-token). Access
tokens (JWT) encode the user's identity and the room permissions they've been granted. You can generate a token with our
CLI:

```shell
lk token create \
    --api-key devkey --api-secret secret \
    --join --room my-first-room --identity user1 \
    --valid-for 24h
```

### Test with example app

Head over to our [example app](https://example.conkit.io) and enter a generated token to connect to your LiveKit
server. This app is built with our [React SDK](https://github.com/a4to/conkit-react).

Once connected, your video and audio are now being published to your new LiveKit instance!

### Simulating a test publisher

```shell
lk room join \
    --url ws://localhost:7880 \
    --api-key devkey --api-secret secret \
    --identity bot-user1 \
    --publish-demo \
    my-first-room
```

This command publishes a looped demo video to a room. Due to how the video clip was encoded (keyframes every 3s),
there's a slight delay before the browser has sufficient data to begin rendering frames. This is an artifact of the
simulation.

## Deployment

### Use LiveKit Cloud

LiveKit Cloud is the fastest and most reliable way to run LiveKit. Every project gets free monthly bandwidth and
transcoding credits.

Sign up for [LiveKit Cloud](https://cloud.conkit.io/).

### Self-host

Read our [deployment docs](https://docs.conkit.io/deploy/) for more information.

## Building from source

Pre-requisites:

-   Go 1.23+ is installed
-   GOPATH/bin is in your PATH

Then run

```shell
git clone https://github.com/a4to/conkit
cd conkit
./bootstrap.sh
mage
```

## Contributing

We welcome your contributions toward improving LiveKit! Please join us
[on Slack](http://conkit.io/join-slack) to discuss your ideas and/or PRs.

## License

LiveKit server is licensed under Apache License v2.0.

<!--BEGIN_REPO_NAV-->
<br/><table>
<thead><tr><th colspan="2">LiveKit Ecosystem</th></tr></thead>
<tbody>
<tr><td>LiveKit SDKs</td><td><a href="https://github.com/a4to/client-sdk-js">Browser</a> · <a href="https://github.com/a4to/client-sdk-swift">iOS/macOS/visionOS</a> · <a href="https://github.com/a4to/client-sdk-android">Android</a> · <a href="https://github.com/a4to/client-sdk-flutter">Flutter</a> · <a href="https://github.com/a4to/client-sdk-react-native">React Native</a> · <a href="https://github.com/a4to/rust-sdks">Rust</a> · <a href="https://github.com/a4to/node-sdks">Node.js</a> · <a href="https://github.com/a4to/python-sdks">Python</a> · <a href="https://github.com/a4to/client-sdk-unity">Unity</a> · <a href="https://github.com/a4to/client-sdk-unity-web">Unity (WebGL)</a></td></tr><tr></tr>
<tr><td>Server APIs</td><td><a href="https://github.com/a4to/node-sdks">Node.js</a> · <a href="https://github.com/a4to/server-sdk-go">Golang</a> · <a href="https://github.com/a4to/server-sdk-ruby">Ruby</a> · <a href="https://github.com/a4to/server-sdk-kotlin">Java/Kotlin</a> · <a href="https://github.com/a4to/python-sdks">Python</a> · <a href="https://github.com/a4to/rust-sdks">Rust</a> · <a href="https://github.com/agence104/conkit-server-sdk-php">PHP (community)</a> · <a href="https://github.com/pabloFuente/conkit-server-sdk-dotnet">.NET (community)</a></td></tr><tr></tr>
<tr><td>UI Components</td><td><a href="https://github.com/a4to/components-js">React</a> · <a href="https://github.com/a4to/components-android">Android Compose</a> · <a href="https://github.com/a4to/components-swift">SwiftUI</a></td></tr><tr></tr>
<tr><td>Agents Frameworks</td><td><a href="https://github.com/a4to/agents">Python</a> · <a href="https://github.com/a4to/agents-js">Node.js</a> · <a href="https://github.com/a4to/agent-playground">Playground</a></td></tr><tr></tr>
<tr><td>Services</td><td><b>LiveKit server</b> · <a href="https://github.com/a4to/egress">Egress</a> · <a href="https://github.com/a4to/ingress">Ingress</a> · <a href="https://github.com/a4to/sip">SIP</a></td></tr><tr></tr>
<tr><td>Resources</td><td><a href="https://docs.conkit.io">Docs</a> · <a href="https://github.com/a4to-examples">Example apps</a> · <a href="https://conkit.io/cloud">Cloud</a> · <a href="https://docs.conkit.io/home/self-hosting/deployment">Self-hosting</a> · <a href="https://github.com/a4to/conkit-cli">CLI</a></td></tr>
</tbody>
</table>
<!--END_REPO_NAV-->
