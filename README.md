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
| Access key | `7fcc1f7c` | kinnay `key` |
| NEX version | `3.8.13` | kinnay `build:3_8_13_2004_0` |
| NGS branch | `origin/release/ngs/3.8.x.200x` | kinnay |
| Structure headers | on (`UseStructureHeader = true`) | NEX 3.8.x-era default — **unverified, no capture exists** |

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
           │  ── LoginEx ─────────────▶  authentication server  :26000
           │  ◀─ Kerberos ticket +
           │     secure server address
           │
           │  ── ticket, RegisterEx ──▶  secure server          :26001
           │  ── GetRankings / UploadScore ▶   leaderboards  → PostgreSQL
           │  ── PreparePostObject ────▶   object upload → S3 presigned PUT
           │  ── CompletePostObject ───▶   metadata          → PostgreSQL
```

## Database

One PostgreSQL database holds the `hyrulewarriors_*` ranking tables, the
`datastore` schema, and the `matchmaking` / `tracking` schemas. It is created
on first start. The `matchmaking` / `tracking` schema is **hand-authored** —
the NEX common library ships the queries but not the DDL. See
[docs/matchmaking-schema.md](docs/matchmaking-schema.md).

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
export PN_HYRULEWARRIORS_AUTH_PORT=26000 PN_HYRULEWARRIORS_SECURE_PORT=26001
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

## What is unverified

No Hyrule Warriors packet capture is known to exist. Two settings are the
first knobs to touch if login or leaderboard decode fails:

1. **`UseStructureHeader`** — set `true` on both endpoints as the NEX 3.8.x
   default. If `LoginEx` fails with "Structure content length longer than
   data size", flip both to `false`. The auth endpoint dumps raw `LoginEx`
   parameter bytes to stdout to make that call.
2. **Ranking / DataStore field layouts** — the `hyrulewarriors_*` queries
   are a generic Postgres leaderboard/object store, not tuned to a captured
   response. `PROTOCOL_COVERAGE.md` lists what to check first.

## License

AGPL-3.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE). No proprietary
Nintendo or Koei Tecmo code or assets are included.
