# H4-04 双语响应式浏览器矩阵

> 默认中文；覆盖中文/英文、桌面/390x844，以及 loading、empty、error、no-permission、saving、destructive、responsive 状态。

## Contact Sheets

- `zh-CN-desktop-matrix.png`
- `zh-CN-mobile-matrix.png`
- `en-US-desktop-matrix.png`
- `en-US-mobile-matrix.png`

## Results

| Locale | Viewport | State | Result | Overflow | Evidence |
| --- | --- | --- | --- | --- | --- |
| zh-CN | desktop | responsive | Pass | 0px | Business dashboard renders at the target viewport. |
| zh-CN | desktop | loading | Pass | 0px | Page skeleton remains visible while API requests are delayed. |
| zh-CN | desktop | empty | Pass | 0px | A no-match customer filter renders the localized empty state. |
| zh-CN | desktop | error | Pass | 0px | A blocked customer API request renders the localized error state. |
| zh-CN | desktop | destructive | Pass | 0px | The guarded scan dialog exposes localized confirm and cancel actions. |
| zh-CN | desktop | saving | Pass | 0px | The scan action exposes a stable localized saving state while its request is delayed. |
| zh-CN | desktop | no_permission | Pass | 0px | A restricted user sees the localized denial page instead of a silent dashboard redirect. |
| zh-CN | mobile | responsive | Pass | 0px | Business dashboard renders at the target viewport. |
| zh-CN | mobile | loading | Pass | 0px | Page skeleton remains visible while API requests are delayed. |
| zh-CN | mobile | empty | Pass | 0px | A no-match customer filter renders the localized empty state. |
| zh-CN | mobile | error | Pass | 0px | A blocked customer API request renders the localized error state. |
| zh-CN | mobile | destructive | Pass | 0px | The guarded scan dialog exposes localized confirm and cancel actions. |
| zh-CN | mobile | saving | Pass | 0px | The scan action exposes a stable localized saving state while its request is delayed. |
| zh-CN | mobile | no_permission | Pass | 0px | A restricted user sees the localized denial page instead of a silent dashboard redirect. |
| en-US | desktop | responsive | Pass | 0px | Business dashboard renders at the target viewport. |
| en-US | desktop | loading | Pass | 0px | Page skeleton remains visible while API requests are delayed. |
| en-US | desktop | empty | Pass | 0px | A no-match customer filter renders the localized empty state. |
| en-US | desktop | error | Pass | 0px | A blocked customer API request renders the localized error state. |
| en-US | desktop | destructive | Pass | 0px | The guarded scan dialog exposes localized confirm and cancel actions. |
| en-US | desktop | saving | Pass | 0px | The scan action exposes a stable localized saving state while its request is delayed. |
| en-US | desktop | no_permission | Pass | 0px | A restricted user sees the localized denial page instead of a silent dashboard redirect. |
| en-US | mobile | responsive | Pass | 0px | Business dashboard renders at the target viewport. |
| en-US | mobile | loading | Pass | 0px | Page skeleton remains visible while API requests are delayed. |
| en-US | mobile | empty | Pass | 0px | A no-match customer filter renders the localized empty state. |
| en-US | mobile | error | Pass | 0px | A blocked customer API request renders the localized error state. |
| en-US | mobile | destructive | Pass | 0px | The guarded scan dialog exposes localized confirm and cancel actions. |
| en-US | mobile | saving | Pass | 0px | The scan action exposes a stable localized saving state while its request is delayed. |
| en-US | mobile | no_permission | Pass | 0px | A restricted user sees the localized denial page instead of a silent dashboard redirect. |

## Reproduce

```powershell
python -m pip install -r scripts/requirements-browser.txt
python scripts/h4-browser-matrix.py --base-url http://127.0.0.1:5174
```
