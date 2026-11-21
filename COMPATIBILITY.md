# Prettier 3.6.2 compatibility

Measured by `go test -run TestPrettierCompat ./pkg/format` against the vendored snapshots in `testdata/prettier` (`js`, `jsx`, `typescript` from the prettier 3.6.2 tag). Comparison is byte-for-byte with the snapshot output, including trailing newlines.

| Set | Passed | Failed | Skipped | Parsed rate | Including skips |
| --- | ---: | ---: | ---: | ---: | ---: |
| JavaScript | 347 | 665 | 223 | 34.29% (347/1012) | 28.10% (347/1235) |
| JSX | 37 | 83 | 10 | 30.83% (37/120) | 28.46% (37/130) |
| TypeScript | 236 | 391 | 29 | 37.64% (236/627) | 35.98% (236/656) |
| Combined | 620 | 1139 | 262 | 35.25% (620/1759) | 30.68% (620/2021) |

The parsed rate is passed / (passed + failed). Skips are not failures.

## Skips

- 110 range or cursor formatting (not implemented)
- 67 experimental Prettier options (`experimentalTernaries`, `experimentalOperatorPosition`)
- 3 flow-only parsers
- 82 syntax errors from tsgo (import assertions, bind expressions, do expressions, legacy octal escapes, optional chains on private names, and similar)

## Remaining failures

Among the 1139 mismatches, the groups are:

- 319 content differs (tokens, not only whitespace)
- 265 breaks only (same tokens, different wrapping)
- 187 comments moved
- 219 parentheses only
- 67 semicolons
- 50 comment text or placement that also changes the token stream
- 30 semicolons mixed into a larger diff
- 2 quotes

The largest fixture directories still failing are comments, TypeScript conformance, JSX, arrows, ternaries, and method chains.
