# Prettier 3.6.2 compatibility

Measured by `go test -run TestPrettierCompat ./pkg/format` against the vendored snapshots in `testdata/prettier` (`js`, `jsx`, `typescript` from the prettier 3.6.2 tag). Comparison is byte-for-byte with the snapshot output, including trailing newlines.

| Set | Passed | Failed | Skipped | Parsed rate | Including skips |
| --- | ---: | ---: | ---: | ---: | ---: |
| JavaScript | 348 | 664 | 223 | 34.39% (348/1012) | 28.18% (348/1235) |
| JSX | 33 | 87 | 10 | 27.50% (33/120) | 25.38% (33/130) |
| TypeScript | 228 | 399 | 29 | 36.36% (228/627) | 34.76% (228/656) |
| Combined | 609 | 1150 | 262 | 34.62% (609/1759) | 30.13% (609/2021) |

The parsed rate is passed / (passed + failed). Skips are not failures.

## Skips

- 110 range or cursor formatting (not implemented)
- 67 experimental Prettier options (`experimentalTernaries`, `experimentalOperatorPosition`)
- 3 flow-only parsers
- 82 syntax errors from tsgo (import assertions, bind expressions, do expressions, legacy octal escapes, optional chains on private names, and similar)

## Remaining failures

Among the 1150 mismatches, the groups are:

- 314 content differs (tokens, not only whitespace)
- 301 breaks only (same tokens, different wrapping)
- 196 comments moved
- 186 parentheses only
- 67 semicolons
- 51 comment text or placement that also changes the token stream
- 33 semicolons mixed into a larger diff
- 2 quotes

The largest fixture directories still failing are comments, TypeScript conformance, JSX, arrows, ternaries, and method chains.
