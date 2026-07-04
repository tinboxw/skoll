# Skoll 鍔熻兘寮€鍙戜笌鐣岄潰浼樺寲鍊欓€?Work Items 2026-07-04

> 鏈枃浠舵浛浠ｅ彂甯?杩愮淮/鏂囨。瀵煎悜鐨勫悗缁€欓€夋睜銆俁elease銆佽繍缁淬€佹枃妗ｅ瀷浠诲姟鍏堟殏鍋溿€?> 褰撳墠姝ｅ紡 `docs/refactor/old/completed-m0-m7-2026-07-04/work_items.md` 淇濇寔 252/252 Done锛屼笉鍥炲啓鐮村潖瀹屾垚鎬併€?> Skoll 浠嶅浜庡紑婧愬熀纭€寤鸿闃舵锛屼笉鍋氭棫鎺ュ彛銆佹棫鏁版嵁缁撴瀯銆佹棫鎻掍欢鏍煎紡銆佹棫椤甸潰璺緞鍏煎鏂规銆?
## 瑙勫垝鍘熷垯

1. 涓嬩竴闃舵涓荤嚎闆嗕腑鍦ㄥ姛鑳藉紑鍙戙€佺晫闈紭鍖栥€佷氦浜掓晥鐜囥€佸墠绔€ц兘鍜屼骇鍝佸寲浣撻獙銆?2. 姣忎釜 Work Item 淇濇寔 0.5-1 澶╃矑搴︼紝楠屾敹涓嶉€氳繃蹇呴』杩斿伐銆?3. 鎵€鏈夊墠绔〉闈㈠繀椤诲寘鍚?loading銆乪mpty銆乪rror銆乶o-permission銆乻aving/disabled 绛夌姸鎬併€?4. API 鏀瑰姩蹇呴』鍚屾 OpenAPI銆佸墠绔?client銆佹潈闄?key銆佸璁?action銆?5. UI 涓嶅仛钀ラ攢椤点€佽楗版€ф笎鍙樸€佸祵濂楀崱鐗囧拰鏃ф祦绋嬪吋瀹瑰叆鍙ｃ€?
## 鐘舵€佹灇涓?
- `Todo`: 寰呮墽琛?- `Doing`: 鎵ц涓?- `Review`: 寰呴獙鏀?- `Failed`: 楠屾敹澶辫触锛岄渶杩斿伐
- `Done`: 楠屾敹閫氳繃涓斿凡鎻愪氦
- `Blocked`: 澶栭儴鏉′欢闃诲

## F0: 浜у搧浣撻獙璇婃柇涓庤璁＄郴缁熷崌绾?
| 椤哄簭 | Work Item | Skill | 浠诲姟 | 浼樺厛绾?| 渚濊禆椤?| 浜や粯鐗?| 楠屾敹鏍囧噯 | 楠岃瘉鍛戒护 | 鐘舵€?|
| ---: | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | F0-01 | `skoll-frontend-design-refactor` | 鏍稿績椤甸潰浣撻獙瀹¤ | P0 | M7-06-04 | `docs/refactor/feature_ui_experience_audit_2026-07-04.md` | Dashboard/User/Role/Permission/Menu/Audit/File/Dictionary/Organization/Plugin/Setting 鍧囨湁闂娓呭崟鍜屼紭鍏堢骇 | 鏂囨。瀹￠槄 | Todo |
| 2 | F0-02 | `skoll-frontend-design-refactor` | 椤甸潰甯冨眬瀵嗗害涓庡眰绾ц鑼冨崌绾?| P0 | F0-01 | UI layout spec | 鏍囬鍖恒€佸伐鍏锋爮銆佺瓫閫夊尯銆佽〃鏍煎尯銆佽鎯呮娊灞夈€佸嵄闄╂搷浣滃竷灞€缁熶竴 | 鏂囨。瀹￠槄 | Todo |
| 3 | F0-03 | `skoll-vue-frontend` | 閫氱敤 PageShell/Toolbar 鍘熷瀷 | P0 | F0-02 | shared components | 鑷冲皯 2 涓〉闈㈡帴鍏ュ師鍨嬶紝鏃犳枃鏈孩鍑哄拰閲嶅宸ュ叿鏍忔牱寮?| `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 4 | F0-04 | `skoll-frontend-testing-refactor` | 鍓嶇瑙嗚/鐘舵€侀獙鏀舵竻鍗曞崌绾?| P0 | F0-03 | acceptance checklist | loading/empty/error/no-permission/saving/destructive/responsive 妫€鏌ラ」鍙洿鎺ョ敤浜庢瘡涓〉闈?| 鏂囨。瀹￠槄 | Todo |

## F1: 绠＄悊鎺у埗鍙版牳蹇冨伐浣滄祦澧炲己

| 椤哄簭 | Work Item | Skill | 浠诲姟 | 浼樺厛绾?| 渚濊禆椤?| 浜や粯鐗?| 楠屾敹鏍囧噯 | 楠岃瘉鍛戒护 | 鐘舵€?|
| ---: | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 5 | F1-01 | `skoll-vue-frontend` | Dashboard 杩愯惀鎬佸崌绾?| P0 | F0 | Dashboard page | 灞曠ず寰呭鐞嗛闄┿€佹渶杩戝璁°€佹彃浠跺仴搴枫€佺郴缁熻祫婧愬叆鍙ｏ紝鏀寔绌烘€佸拰閿欒鎬?| `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 6 | F1-02 | `skoll-api-contracts` | Dashboard 鑱氬悎 API | P0 | F1-01 | OpenAPI + handler/service | Dashboard 鏁版嵁鏉ヨ嚜鐪熷疄 API锛屼笉浣跨敤闈欐€佸亣鏁版嵁锛涢敊璇爜鍜屾潈闄愭槑纭?| `go test ./...`; OpenAPI review | Todo |
| 7 | F1-03 | `skoll-vue-frontend` | 鐢ㄦ埛/瑙掕壊鎵归噺鎿嶄綔鍗囩骇 | P0 | F0 | User/Role pages | 鎵归噺鍚仠銆佹壒閲忔巿鏉冨叆鍙ｃ€佹搷浣滅粨鏋滃弽棣堛€佸け璐ラ」鍒楄〃瀹屾暣 | `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 8 | F1-04 | `skoll-permission-rbac` | 鎵归噺鎺堟潈鏈嶅姟涓庡璁?| P0 | F1-03 | service/API/audit | 鎵归噺鎺堟潈鍏峰浜嬪姟杈圭晫銆佹潈闄愭牎楠屻€佸璁¤褰曞拰澶辫触鍥炴粴绛栫暐 | `go test ./internal/service/rbac/... ./internal/handler/http/v1/...` | Todo |
| 9 | F1-05 | `skoll-web-ui-design` | 鏉冮檺鐭╅樀浜や簰澧炲己 | P0 | F1-04 | Permission/Role UI | 鏀寔鎸夋ā鍧楃瓫閫夈€侀闄╂爣绛俱€佸樊寮傞瑙堛€佷繚瀛樺墠纭 | `cd web && npm run build` | Todo |
| 10 | F1-06 | `skoll-testing-automation` | 鏍稿績宸ヤ綔娴?smoke | P0 | F1-01..F1-05 | smoke record/tests | 鐧诲綍銆丏ashboard銆佺敤鎴锋壒閲忋€佽鑹叉巿鏉冦€佹潈闄愮煩闃典富娴佺▼鍙鏌?| `go test ./...`; `cd web && npm run build` | Todo |

## F2: 瀹¤銆侀闄╀笌鍙娴嬬晫闈骇鍝佸寲

| 椤哄簭 | Work Item | Skill | 浠诲姟 | 浼樺厛绾?| 渚濊禆椤?| 浜や粯鐗?| 楠屾敹鏍囧噯 | 楠岃瘉鍛戒护 | 鐘舵€?|
| ---: | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 11 | F2-01 | `skoll-observability-audit` | 椋庨櫓浜嬩欢瑙勫垯妯″瀷 | P0 | M2 | domain/service | 鏀寔椋庨櫓绛夌骇銆佷簨浠舵潵婧愩€佸缃姸鎬併€佸叧鑱旇祫婧愬拰澶勭悊浜?| `go test ./internal/domain/audit/... ./internal/service/audit/...` | Todo |
| 12 | F2-02 | `skoll-api-contracts` | 瀹¤椋庨櫓鐪嬫澘 API | P0 | F2-01 | OpenAPI + handler | 椋庨櫓缁熻銆佽秼鍔裤€乀OP actor/resource/action銆佹湭澶勭悊浜嬩欢鎺ュ彛瀹屾暣 | `go test ./internal/handler/http/v1/audit/...`; OpenAPI review | Todo |
| 13 | F2-03 | `skoll-vue-frontend` | 瀹¤椋庨櫓鐪嬫澘 UI | P0 | F2-02 | Audit page/dashboard panel | 瓒嬪娍銆佺瓫閫夈€佽鎯呫€佸缃€佺┖鎬併€侀敊璇€併€佹棤鏉冮檺鎬佸畬鏁?| `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 14 | F2-04 | `skoll-web-ui-design` | 瀹¤璇︽儏鎶藉眽浣撻獙鍗囩骇 | P1 | F2-03 | Audit detail drawer | 鍘熷 payload銆乨iff銆佸叧鑱斿璞°€佸鐞嗚褰曞彲鎵弿锛屼笉婧㈠嚭 | `cd web && npm run build` | Todo |
| 15 | F2-05 | `skoll-testing-automation` | 椋庨櫓瀹¤楠屾敹鏍锋湰 | P0 | F2-01..F2-04 | fixture/tests | 鐧诲綍澶辫触銆佹潈闄愭嫆缁濄€佹彃浠堕闄┿€佹壒閲忔巿鏉冮闄╁潎鍙煡璇㈠拰澶勭疆 | `go test ./...`; `cd web && npm run build` | Todo |

## F3: 鏂囦欢涓庢暟鎹鐞嗕綋楠屽崌绾?
| 椤哄簭 | Work Item | Skill | 浠诲姟 | 浼樺厛绾?| 渚濊禆椤?| 浜や粯鐗?| 楠屾敹鏍囧噯 | 楠岃瘉鍛戒护 | 鐘舵€?|
| ---: | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 16 | F3-01 | `skoll-file-storage-refactor` | 鏂囦欢鏍囩涓庝笟鍔″綊灞炴ā鍨?| P1 | M3 | domain/store | 鏂囦欢鏀寔 tags銆乥iz_type銆乥iz_id銆乷wner銆乿isibility 鏌ヨ | `go test ./internal/domain/file/... ./internal/store/...` | Todo |
| 17 | F3-02 | `skoll-api-contracts` | 鏂囦欢楂樼骇绛涢€?API | P1 | F3-01 | OpenAPI + handler | 鎸夋爣绛俱€佷笟鍔″綊灞炪€佷笂浼犺€呫€佸ぇ灏忋€佺被鍨嬨€佹椂闂寸瓫閫?| `go test ./internal/handler/http/v1/file/...`; OpenAPI review | Todo |
| 18 | F3-03 | `skoll-vue-frontend` | 鏂囦欢绠＄悊椤甸潰鍗囩骇 | P1 | F3-02 | File page | 鍒楄〃/缃戞牸鍒囨崲銆侀瑙堟娊灞夈€佹壒閲忓垹闄ゃ€佷笅杞姐€佹潈闄愭€佸畬鏁?| `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 19 | F3-04 | `skoll-data-dictionary-config` | 瀛楀吀鎵归噺瀵煎叆涓庡樊寮傞瑙?| P1 | M4 | service/API/UI | 瀵煎叆鍓嶅睍绀烘柊澧?鏇存柊/鍐茬獊锛屼繚瀛樺悗缂撳瓨澶辨晥鍜屽璁″畬鏁?| `go test ./...`; `cd web && npm run build` | Todo |
| 20 | F3-05 | `skoll-vue-frontend` | 缁勭粐鏍戞嫋鎷芥帓搴忎笌褰掑睘缂栬緫浼樺寲 | P1 | M4 | Organization page | 鎷栨嫿鎺掑簭銆佸矖浣嶅綊灞炪€佺敤鎴峰綊灞炵紪杈戝叿澶囩‘璁ゅ拰澶辫触鍙嶉 | `cd web && npm run build` | Todo |

## F4: 鐢熸垚鍣ㄤ笌浣庝唬鐮侀厤缃綋楠屽寮?
| 椤哄簭 | Work Item | Skill | 浠诲姟 | 浼樺厛绾?| 渚濊禆椤?| 浜や粯鐗?| 楠屾敹鏍囧噯 | 楠岃瘉鍛戒护 | 鐘舵€?|
| ---: | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 21 | F4-01 | `skoll-generator-refactor` | GeneratorSpec 鍙鍖栫紪杈戞ā鍨?| P0 | M5 | domain/service | 瀛楁銆佺储寮曘€佹牎楠屻€佹潈闄愩€佽彍鍗曘€侀〉闈㈤厤缃彲缁撴瀯鍖栫紪杈?| `go test ./internal/domain/generator/... ./internal/service/generator/...` | Todo |
| 22 | F4-02 | `skoll-api-contracts` | 鐢熸垚鍣ㄨ璁″櫒 API | P0 | F4-01 | OpenAPI + handler | spec draft銆乿alidate銆乨ry-run銆乨iff銆乤pply 鎺ュ彛瀹屾暣 | `go test ./internal/handler/http/v1/generator/...`; OpenAPI review | Todo |
| 23 | F4-03 | `skoll-vue-frontend` | 鐢熸垚鍣ㄨ璁″櫒 UI | P0 | F4-02 | Generator designer page | 瀛楁琛ㄦ牸銆佹牎楠岄厤缃€佹潈闄愯彍鍗曢厤缃€乨ry-run diff銆侀敊璇€佸畬鏁?| `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 24 | F4-04 | `skoll-code-generator` | 鐢熸垚椤甸潰浣撻獙妯℃澘鍗囩骇 | P1 | F4-03 | generator templates | 鐢熸垚椤甸粯璁ゅ叿澶?PageShell銆佺姸鎬佺粍浠躲€佹潈闄愭寜閽€佹娊灞夎〃鍗?| `go test ./...`; `cd web && npm run build` | Todo |
| 25 | F4-05 | `skoll-testing-automation` | 鐢熸垚鍣ㄨ璁″櫒绔埌绔獙鏀?| P0 | F4-01..F4-04 | tests/fixtures | 浠?UI spec 鍒?dry-run 鍒扮敓鎴?demo module 鍙鏌?| `go test ./...`; `cd web && npm run build` | Todo |

## F5: 鎻掍欢鎺у埗鍙颁笌寮€鍙戣€呬綋楠屽崌绾?
| 椤哄簭 | Work Item | Skill | 浠诲姟 | 浼樺厛绾?| 渚濊禆椤?| 浜や粯鐗?| 楠屾敹鏍囧噯 | 楠岃瘉鍛戒护 | 鐘舵€?|
| ---: | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 26 | F5-01 | `skoll-plugin-platform` | 鎻掍欢杩愯鍋ュ悍鐘舵€佹ā鍨?| P1 | M6 | domain/service | 鎻掍欢鍚仠銆佹渶杩戦敊璇€侀闄┿€侀厤缃€佺伆搴︾姸鎬佸彲鑱氬悎 | `go test ./internal/plugin/...` | Todo |
| 27 | F5-02 | `skoll-api-contracts` | 鎻掍欢鍋ュ悍涓庝换鍔?API | P1 | F5-01 | OpenAPI + handler | 鎻掍欢鍋ュ悍銆佷换鍔¤繘搴︺€侀敊璇鎯呫€佸璁＄嚎绱㈡帴鍙ｅ畬鏁?| `go test ./internal/handler/http/v1/plugin/...`; OpenAPI review | Todo |
| 28 | F5-03 | `skoll-vue-frontend` | 鎻掍欢璇︽儏椤典俊鎭灦鏋勫崌绾?| P1 | F5-02 | Plugin detail UI | 姒傝銆佹潈闄愩€侀厤缃€侀闄┿€佹棩蹇椼€佷换鍔°€佸洖婊氬垎鍖烘竻鏅?| `cd web && npm run typecheck`; `cd web && npm run build` | Todo |
| 29 | F5-04 | `skoll-web-ui-design` | 鎻掍欢瀹夎/鍗囩骇鍚戝浼樺寲 | P1 | F5-03 | Plugin wizard UI | 棰勬銆侀闄╃‘璁ゃ€侀厤缃€佹墽琛岃繘搴︺€佸け璐ユ仮澶嶆楠ゅ畬鏁?| `cd web && npm run build` | Todo |
| 30 | F5-05 | `skoll-testing-automation` | 鎻掍欢鎺у埗鍙版祻瑙堝櫒 smoke | P1 | F5-01..F5-04 | smoke record/tests | 瀹夎銆侀厤缃€佸惎鍋溿€佸崌绾с€佸洖婊氥€侀闄╂姤鍛婁富娴佺▼鍙鏌?| `go test ./...`; `cd web && npm run build` | Todo |

## 鎺ㄨ崘鎵ц椤哄簭

1. 鍏堟墽琛?F0锛岀粺涓€椤甸潰澹炽€佸伐鍏锋爮銆佺姸鎬佸拰楠屾敹鏍囧噯銆?2. F1 涓?F2 浣滀负 P0 鍔熻兘涓荤嚎骞惰鎺ㄨ繘锛屽厛鎻愬崌鏍稿績绠＄悊鍜岄闄╁璁′綋楠屻€?3. F4 鐨勭敓鎴愬櫒璁捐鍣ㄤ紭鍏堢骇楂樹簬 F3/F5锛屽洜涓哄畠鑳芥斁澶у悗缁ā鍧楀紑鍙戞晥鐜囥€?4. F3/F5 浣滀负 P1 浣撻獙澧炲己锛岄€傚悎鍦?F0 璁捐绯荤粺钀藉湴鍚庢帹杩涖€?