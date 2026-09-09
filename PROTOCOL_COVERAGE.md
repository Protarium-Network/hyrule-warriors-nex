# Protocol coverage

What this server implements for **Hyrule Warriors** (Wii U), and how sure we
are of each. No packet capture of this title is known to exist, so "status"
is about inference confidence, not observation. See [RECON.md](RECON.md).

## Authentication endpoint

| Protocol | Coverage | Status |
|---|---|---|
| Ticket Granting | Login / LoginEx / RequestTicket. `ValidateLoginData` accepts the account-server token (or anything, in local mode). | Stock common handler. |

`ByteStreamSettings.UseStructureHeader` is set **`true`** on both endpoints
(the NEX 3.5+ default). **Unverified** for this title. If `LoginEx` fails
with "Structure content length longer than data size", flip both endpoints
to `false`. `nex/authentication.go` dumps the raw `LoginEx` parameter bytes
to stdout so this can be checked against a real capture by hand.

## Secure endpoint

| Protocol | Coverage | Status |
|---|---|---|
| Secure Connection | baseline handshake, insecure `Register` | Stock. Required for any secure endpoint. |
| Utility | baseline (`AcquireNexUniqueID` etc.) | Stock. |
| Ranking | `GetRankings`, `GetRankingsAndCount`, `UploadScore`, common-data get/upload. Range / User / Near modes; friend-range → User. | Postgres-backed (`hyrulewarriors_*` tables). Generic store — category numbers and `RankingScoreData` semantics **not tuned to a capture**. |
| DataStore | `PostMetaBinary`, `PrepareGetObject` (DataID / persistence slot), `PreparePostObject` → S3 presigned PUT → `CompletePostObject`, `ChangeMeta`, `GetMetasMultipleParam`, period / meta-binary / data-type updates. `GetRatings` → generic empty slots. | Vendored `internal/` fork for the S3 presigner + method behaviour; Postgres object store. The linked-heaviest family — Network Link / My Fairy exchange. |
| NAT Traversal | `ReportNATProperties` etc. | Stock. Linked by the RPX; expected idle. |
| MatchMaking / MatchMakingExt | via the common `MatchmakingManager` | Stock. Linked by the RPX; expected idle. |
| MatchmakeExtension | stock common handlers; `Cleanup*` callbacks set to no-ops so `AutoMatchmake*Postpone` don't hard-fail | Stock. **No** title-specific decoder patch (the 3.4.x templates needed one; 3.8.13 speaks the modern layout). Linked by the RPX; expected idle — Hyrule Warriors on Wii U has no online co-op or versus. |

## First things to check against a real capture

1. **`UseStructureHeader`** on both endpoints (see above).
2. **Ranking category layout** — whether `GetRankings` responses line up
   client-side; the `3.6.0` `UpdateTime` bump the 3.4.x servers needed is
   assumed already-default at 3.8.13.
3. **DataStore `GetMeta` / `PrepareGetObject` field set** — Network Link
   objects may carry title-specific `MetaBinary` the client expects echoed.
4. Whether `GetRatings` is called at all.

## Not implemented

- Persistent gatherings / communities beyond the schema and the common
  library's own handlers — no evidence this title uses them.
- Miiverse (`nn_olv`) and BOSS / SpotPass — separate stacks, out of scope.
