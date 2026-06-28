# Release Distribution

AgentMeter releases are portable archives. They include the executable, built
Web assets, README files, license, and changelog.

For install instructions, see [Install And Run AgentMeter](install.md).

## Current Release Assets

The release workflow publishes these archive names for tags that start with
`v`:

```text
AgentMeter-windows-amd64.zip
AgentMeter-windows-arm64.zip
AgentMeter-linux-amd64.tar.gz
AgentMeter-linux-arm64.tar.gz
AgentMeter-darwin-amd64.tar.gz
AgentMeter-darwin-arm64.tar.gz
checksums.txt
```

The packaged app runs locally and serves the Web UI on loopback by default:

```text
http://127.0.0.1:34115
```

## Distribution Channels To Add

Package-manager distribution should reuse the same release archives and
checksums instead of building a different app shape.

| Channel | Recommended path | Notes |
| --- | --- | --- |
| Homebrew | Tap formula that downloads the macOS and Linux tarballs. | Use release checksums and keep caveats clear about local data paths. |
| Scoop | Manifest for Windows zip archives. | Point to `AgentMeter-windows-<arch>.zip` and expose `AgentMeter.exe`. |
| Winget | Manifest for Windows release archives. | Best after release signing or after maintainers accept unsigned portable packages. |
| Docker | Optional local demo image only. | Docker is less natural for reading host agent session files; document mounts explicitly if added. |

## Package Metadata Checklist

Use this wording consistently:

- Name: `AgentMeter`
- Summary: `Local-first dashboard for coding-agent session usage`
- License: `Apache-2.0`
- Website: `https://blog.lyle.ac.cn/AgentMeter/`
- Source: `https://github.com/LyleMi/AgentMeter`
- Data model: local SQLite database
- Privacy: no proxy, no cloud service, no telemetry

Do not claim installer signing, notarization, auto-update, cloud sync, or
provider billing reconciliation unless those features exist in the release.
