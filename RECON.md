# Hyrule Warriors (Wii U) — online-stack recon

Everything here is from **static analysis of the retail executable**
(`ProjectZ_r.rpx`) cross-referenced with the public
[kinnay.github.io](https://kinnay.github.io/) Wii U NEX database. No packet
capture of this title is known to exist, so the *wire layout* of every
protocol is inference from the NEX version, not observation — flagged as such
here and in [PROTOCOL_COVERAGE.md](PROTOCOL_COVERAGE.md).

## The executable

| | |
|---|---|
| File | `ProjectZ.rpx` (retail RPX, ELF32 BE PowerPC, Cafe OS), zlib section compression |
| Internal name | `ProjectZ` — Hyrule Warriors' development codename (Koei Tecmo, KTSL engine; build path `C:\ProjectZ\Binary\ProjectZ_r.rpx`) |
| Confirming strings | `ProjectZMiiverseProc`, `ProjectZAmiiboAccess`, `ProjectZ-SAVEDATA`, `C:\ProjectZ\Program\Library\KTSL\Source\WiiU\KtslDeviceCore.cpp` |

There is **no `nn_nex` / `nn_nds` / `nn_boss` RPL import** — NEX is
**statically linked** into the RPX, the same as Trine 2, Lost Reavers and
the Mario & Sonic titles. The decompressed `.text` / `.rodata` carry the
NEX protocol class sets, by rough string frequency:

| Protocol family | approx. string hits | Meaning |
|---|---:|---|
| DataStore | ~1375 | the dominant online workload |
| Ranking | ~362 | leaderboards |
| MatchMaking / MatchmakeExtension | ~390 | linked; see below |
| NAT Traversal | ~170 | linked; peer transport for MM |

## Authentication path

`nn::act::AcquireNexServiceToken(ACTNexAuthenticationResult*, u32 GameID)` is
present (mangled `AcquireNexServiceToken__Q2_2nn3actFP26ACTNexAuthenticationResultUi`),
with the log format string:

```
GameID:%08x Success:%d AcquireNexServiceTokenResult:%08x
```

That is the **standard Wii U NEX flow**: the game asks `nn::act` for a NEX
service token for its game server ID, `nn::act` talks to NASC, gets back a
Kerberos-style token plus the game server's address, and NEX's
`TicketGrantingProtocolClient` logs into the authentication server and
connects to the secure server over PRUDP.

## NEX configuration (from kinnay's database)

`https://kinnay.github.io/data/nexwiiu.json`, entry:

```json
{
  "id":     269995264,
  "aid":    1407375153548544,
  "name":   "Hyrule Warriors",
  "addr":   ["34.208.166.202", 43940],
  "key":    "7fcc1f7c",
  "branch": "branch:origin/release/ngs/3.8.x.200x",
  "build":  "build:3_8_13_2004_0"
}
```

| Field | Value | Notes |
|---|---|---|
| Game server ID | `1017cd00` | `id` (269995264) as 8 hex digits — the `%08x` the log format expects. Region-independent: all regions of Hyrule Warriors share one NEX game server. |
| Access key | `7fcc1f7c` | seeds every PRUDP packet signature. Not stored in plaintext in the RPX (computed/obfuscated by NEX at runtime); taken from kinnay's DB, the same source used for Trine 2 (`f9c35adc`) and Sochi 2014 (`585214a5`). |
| NEX version | **3.8.13** | `build:3_8_13_2004_0` |
| NGS branch | `origin/release/ngs/3.8.x.200x` | |
| `addr` | `34.208.166.202:43940` | the AWS box NASC handed out in the capture kinnay's DB was built from — dead, irrelevant, we self-host |

`aid` (1407375153548544 = `0x0005000010`... ) is `id | (0x5 << 48)`, a
synthetic value in kinnay's DB, **not** the retail eShop title ID (Hyrule
Warriors US is `0005000010115C00`). Only `id` / `key` / `build` matter for
the server.

### NEX 3.8.13 vs. the 3.4.x templates

This server was templated from Mario & Sonic Sochi 2014 (NEX 3.4.7). Two
workarounds those 3.4.x servers need are **not** applied here because 3.8.13
is past the versions that fixed them:

- Sochi/Rio bump **Ranking** to `3.6.0` so `RankingRankData.UpdateTime`
  serializes. At 3.8.13 that field is already in the default layout.
- Sochi pins **MatchMaking** to `3.3.0` for the pre-3.4 `MatchmakeSession` /
  search-criteria layout. A 3.8.13 client uses the modern layout.

So `LibraryVersions` is left at the single `SetDefault(3, 8, 13)`.

### Structure headers — confirmed on hardware

Rio 2016 (3.4.7) writes structure-header version bytes before structured RMC
parameters; Sochi 2014 (also 3.4.7) does not — it is title-specific at 3.4.x.
By the 3.5+ era, structure headers on is the norm, so both endpoints set
`ByteStreamSettings.UseStructureHeader = true`. A real console `LoginEx`
(2026-09-09) decoded cleanly with this on — `AuthenticationInfo{Token,
NGSVersion:3, TokenType:1, ServerVersion:0x7d2}` parsed with no
"Structure content length" error. `nex/authentication.go` keeps the raw
`LoginEx` byte dump for future diffing.

### PRUDPv1 CONNECT-ACK signature — confirmed on hardware

The 3.4.x Wii U titles (Trine 2, the Mario & Sonic servers) need
`PRUDPV1Settings.LegacyConnectionSignature = true`: their console signs
CONNECT with an empty connection signature and rejects a CONNECT-ACK signed
any other way. A real Hyrule Warriors console (NEX 3.8.13, 2026-09-09) does
the **opposite** — with `LegacyConnectionSignature = true` it rejected every
CONNECT-ACK and retransmitted CONNECT every ~2 s until `106-0502`
(`Transport::ConnectionFailure`). Both endpoints are set to `false` (nex-go's
default, the modern scheme); the handshake then completes and the console
proceeds to `Register` and DataStore.

## Ranking surface

Hyrule Warriors' "Network Features" menu includes online rankings. The
server implements the standard Ranking protocol via
`nex-protocols-common-go`, backed by dedicated `hyrulewarriors_*` Postgres
tables (isolated so category numbers can't collide with other titles sharing
the database). Range / User / Near modes; friend-range degrades to User (no
friends system). The exact category numbers and `RankingScoreData` semantics
this title uses are **not known without a capture** — the store is generic.

## DataStore surface

The heaviest linked family. Hyrule Warriors uses DataStore for the shared
objects the Network Features menu exchanges:

- **Network Links** — a player publishes their roster/progress as an object;
  others download it and fight AI versions.
- **My Fairy** — shareable customised companion data.

Flow: `PostMetaBinary` / `PreparePostObject` → S3 presigned PUT →
`CompletePostObject` to publish; `PrepareGetObject` (by DataID or
persistence slot) / `GetMetasMultipleParam` to fetch. The vendored
`internal/nex-protocols-common-go-patch` provides the S3 presigner and the
adjusted method behaviour; `database/` provides the Postgres object store.
An S3-compatible endpoint (`PN_S3_ENDPOINT`, e.g. self-hosted MinIO) is
required only for uploads.

`DataStore::GetRatings` is wired to a generic empty-slot response — Hyrule
Warriors' shared objects are not rated content. Drop the handler if a
capture shows the title never calls it.

## Matchmaking surface

`MatchmakeExtension` / `MatchMaking` / `NAT Traversal` are in the RPX, so the
secure endpoint registers them with the **stock common handlers** and a
Postgres-backed `MatchmakingManager`. But Hyrule Warriors on Wii U has **no
online co-op and no online versus** — local co-op only. These symbols are
almost certainly linked NEX boilerplate (the same way Trine 2 links
`CreateCommunity` it never calls). The registration is defensive: a stray
call decodes and answers instead of dropping the connection. There is no
title-specific matchmaking decoder patch.

## What the server implements

Auth endpoint: **Ticket Granting**.
Secure endpoint: **Secure Connection**, **Utility**, **Ranking**,
**DataStore**, **NAT Traversal**, **MatchMaking**, **MatchMakingExt**,
**MatchmakeExtension**.

Not implemented: `AccountManagementService` (the account already exists by
the time the game reaches NEX), Miiverse / `nn_olv` (its own stack), BOSS /
SpotPass task delivery.
