# Prettier 3.6.2 compatibility

Measured by `go test -run TestPrettierCompat ./pkg/format` against the vendored snapshots in `testdata/prettier` (`js`, `jsx`, `typescript` from the prettier 3.6.2 tag). Comparison is byte-for-byte with the snapshot output, including trailing newlines.

| Set | Passed | Failed | Skipped | Parsed rate | Including skips |
| --- | ---: | ---: | ---: | ---: | ---: |
| JavaScript | 353 | 659 | 223 | 34.88% (353/1012) | 28.58% (353/1235) |
| JSX | 37 | 83 | 10 | 30.83% (37/120) | 28.46% (37/130) |
| TypeScript | 227 | 400 | 29 | 36.20% (227/627) | 34.60% (227/656) |
| Combined | 617 | 1142 | 262 | 35.08% (617/1759) | 30.53% (617/2021) |

The parsed rate is passed / (passed + failed). Skips are not failures.

## Skips

- 110 range or cursor formatting (not implemented)
- 67 experimental Prettier options (`experimentalTernaries`, `experimentalOperatorPosition`)
- 3 flow-only parsers
- 82 syntax errors from tsgo (import assertions, bind expressions, do expressions, legacy octal escapes, optional chains on private names, and similar)

## Remaining failures

Among the 1142 mismatches, the groups are:

- 316 content differs (tokens, not only whitespace)
- 291 breaks only (same tokens, different wrapping)
- 191 comments moved
- 187 parentheses only
- 71 semicolons
- 51 comment text or placement that also changes the token stream
- 33 semicolons mixed into a larger diff
- 2 quotes

The largest fixture directories still failing are comments, TypeScript conformance, JSX, arrows, ternaries, and method chains.
