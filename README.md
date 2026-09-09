# Hyrule Warriors (Wii U) — NEX server

A preservation-oriented NEX server for the Wii U title **Hyrule Warriors**
(Koei Tecmo / Nintendo, 2014; internal codename "Project Z"),
`game_server_id` `1017cd00`. It speaks the game's PRUDP authentication and
secure protocols so the **Network Features** menu works again after the
official servers went away: online leaderboards, and the shared **Network
Link** / **My Fairy** objects other players download.

Built on the [Pretendo Network](https://github.com/PretendoNetwork) NEX
libraries (`nex-go`, `nex-protocols-go`, `nex-protocols-common-go`) — the
same stack as this org's Sonic & All-Stars Racing Transformed, Mario Tennis
and Mario & Sonic servers. `internal/nex-protocols-common-go-patch/` is a
vendored fork carrying the DataStore/S3 changes the object-upload flow needs.

The evidence trail for every constant and design choice is in
[RECON.md](RECON.md); per-protocol status is in
[PROTOCOL_COVERAGE.md](PROTOCOL_COVERAGE.md).

## Recovered configuration

| Field | Value | Source |
|---|---|---|
| Game server ID | `1017cd00` (269995264) | kinnay [`nexwiiu.json`](https://kinnay.github.io/data/nexwiiu.json) `id`, as `%08x` |
| Access key | `7fcc1f7c` | kinnay `key` — **confirmed on hardware** (PRUDP handshake completes) |
| NEX version | `3.8.13` | kinnay `build:3_8_13_2004_0`; console reports server version `0x7d2` (2002) |
| NGS branch | `origin/release/ngs/3.8.x.200x` | kinnay |
| Structure headers | on (`UseStructureHeader = true`) | **confirmed on hardware** — `LoginEx` `AuthenticationInfo` decodes cleanly |
| PRUDPv1 CONNECT-ACK | modern scheme (`LegacyConnectionSignature = false`) | **confirmed on hardware** — the 3.4.x titles need the legacy empty-signature scheme; a real Hyrule Warriors console rejects that and needs the modern one |

The retail RPX (`ProjectZ_r.rpx`) statically links NEX (no `nn_nex` RPL
import) and drives it through `nn::act::AcquireNexServiceToken` with the game
server ID above — the standard Wii U NASC → `nex_token` → PRUDP path, the
same as Trine 2, Lost Reavers and the Mario & Sonic titles. Its `.text` /
`.rodata` carry the DataStore (heaviest), Ranking, MatchmakeExtension and NAT
Traversal class sets.

## Scope

- **Ticket Granting** — login / secure-server handoff
- **Secure Connection**, **Utility** — baseline secure-endpoint handshake
- **Ranking** — leaderboards (`GetRankings`, `GetRankingsAndCount`,
  `UploadScore`, common-data get/upload). Range / User / Near modes;
  friend-range degrades to User (no friends system).
- **DataStore** — the shared-object flow: `PostMetaBinary`,
  `PrepareGetObject` (download a Network Link / My Fairy by DataID or
  persistence slot), `PreparePostObject` → S3 presigned upload →
  `CompletePostObject`, plus meta / period / data-type updates and
  `GetMetasMultipleParam`.
- **NAT Traversal + MatchMaking + MatchMakingExt + MatchmakeExtension** —
  registered because the RPX links them, with the stock common handlers.
  Hyrule Warriors (Wii U) has no online co-op or versus, so this path is
  expected to stay idle; it is wired defensively, not from a capture.

No S3 bucket is required unless players upload objects (`PreparePostObject`).

```
        console                       this server
           │
           │  ── LoginEx ─────────────▶  authentication server  :27200
           │  ◀─ Kerberos ticket +
           │     secure server address
           │
           │  ── ticket, RegisterEx ──▶  secure server          :27201
           │  ── GetRankings / UploadScore ▶   leaderboards  → PostgreSQL
           │  ── PreparePostObject ────▶   object upload → S3 presigned PUT
           │  ── CompletePostObject ───▶   metadata          → PostgreSQL
```

## Database

One PostgreSQL database. `database/init_postgres.go` creates the
`hyrulewarriors_*` ranking tables and the `datastore` schema; the
`matchmaking` / `tracking` schema is created by `nex-protocols-common-go`
itself (in `CommonProtocol.SetManager`). See
[docs/matchmaking-schema.md](docs/matchmaking-schema.md) for the reference
layout.

## Running

### Local preservation mode

```bash
cp .env.example .env                     # PN_HYRULEWARRIORS_LOCAL_MODE=1 by default
cp settings.example.json settings.json   # add your console's PID + NEX password
docker compose up --build
```

In local mode there is no account server: player NEX passwords come from
`settings.json` and the login token is accepted unconditionally. Use it only
on an isolated network. Set `PN_HYRULEWARRIORS_SECURE_HOST` to the LAN IP the
console can reach this machine on (not `localhost`, unless the client runs
here too).

### Shared mode

Set `PN_HYRULEWARRIORS_LOCAL_MODE` to anything but `1` and provide
`PN_HYRULEWARRIORS_NEX_TOKEN_AES_KEY` (64 hex chars) and
`PN_HYRULEWARRIORS_NEX_PASSWORD_SECRET` (≥32 bytes hex), both matching your
account server. Login tokens are then decrypted and validated, and each
player's NEX password is derived as `HMAC-SHA256(secret, pid)`.

### Without Docker

```bash
export PN_HYRULEWARRIORS_AUTH_PORT=27200 PN_HYRULEWARRIORS_SECURE_PORT=27201
export PN_HYRULEWARRIORS_SECURE_HOST=<LAN-IP-of-this-machine>
export PN_HYRULEWARRIORS_POSTGRES_URI='postgres://hyrulewarriors:hyrulewarriors@localhost:5432/hyrulewarriors?sslmode=disable'
export PN_HYRULEWARRIORS_LOCAL_MODE=1
go build -o hyrulewarriors-nex . && ./hyrulewarriors-nex
```

### Pointing a console at it

The console reaches this server the same way as any Pretendo Wii U title: its
account server's `nex_token` response for game server ID `1017cd00` must
return this server's `PN_HYRULEWARRIORS_SECURE_HOST` / auth port. On a real
Pretendo or Protarium setup that means adding a game-server entry; in a fully
isolated setup, a DNS + account-server redirect for the title.

`PN_HYRULEWARRIORS_SECURE_HOST` **must be short** (~15 chars) — the retail
binary truncates it into a fixed-size buffer.

## Hardware status

Verified on a real Wii U (2026-09-09): `AcquireNexServiceToken` → `LoginEx` →
`RequestTicket` → secure `Register` → `DataStore::GetMetasMultipleParam` all
succeed and the game's Network Features menu opens without an error code. No
packet capture of the retail servers exists; the wire settings above were
found by iterating against the console.

Still unverified — needs real network content and/or a second console:

1. **Ranking field layouts** — the `hyrulewarriors_*` queries are a generic
   Postgres leaderboard store, not tuned to a captured response. Untested
   until the game actually submits/reads a leaderboard.
2. **DataStore `GetMeta` / object payloads** — `GetMetasMultipleParam`
   returns cleanly for absent objects (empty network), but the
   `DataStoreMetaInfo` encoding for a *populated* Network Link / My Fairy
   object is untested. `PROTOCOL_COVERAGE.md` has the details.
3. **Object upload** — needs `PN_S3_ENDPOINT` (see below); not yet wired on
   the reference deployment.

## License

AGPL-3.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE). No proprietary
Nintendo or Koei Tecmo code or assets are included.
