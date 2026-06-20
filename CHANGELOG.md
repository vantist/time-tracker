# Changelog

## [1.9.0](https://github.com/vantist/time-tracker/compare/v1.8.0...v1.9.0) (2026-06-20)


### Features

* **pricing:** expand pricing table with 2026 models and custom pricing ([02fd0c3](https://github.com/vantist/time-tracker/commit/02fd0c30da1933efee45e2c5786b09387022e1c2))
* **pricing:** implement version and preview suffix normalization ([30e4497](https://github.com/vantist/time-tracker/commit/30e4497a0133c2a6c1d56ba7744526a296ab6714))

## [1.8.0](https://github.com/vantist/time-tracker/compare/v1.7.0...v1.8.0) (2026-06-20)


### Features

* **pricing:** add pricing and tests for gpt-5.4 and gpt-5-mini ([e5f930e](https://github.com/vantist/time-tracker/commit/e5f930e1a823c5699a3cb60d0884a8ad956bcc22))
* **record:** integrate copilot-cli and antigravity log parsers with response hook ([7e54cb2](https://github.com/vantist/time-tracker/commit/7e54cb2affaa61d18c67f899509a949390b528f9))
* **transcript:** add antigravity transcript.jsonl parser and tests ([875e571](https://github.com/vantist/time-tracker/commit/875e571cb8a94ce819b4ed77fd7e78fcf656f96d))
* **transcript:** add copilot events.jsonl parser and tests ([a238087](https://github.com/vantist/time-tracker/commit/a238087620c6b82ab1cc012a928749a68d239d77))

## [1.7.0](https://github.com/vantist/time-tracker/compare/v1.6.0...v1.7.0) (2026-06-19)


### Features

* **dashboard:** display model usages ratio and details in web dashboard ([8942088](https://github.com/vantist/time-tracker/commit/89420883e00b0d2d37b65fab01ece02b7713b381))
* **db:** add turn_model_usages table and migrate backfill ([8a1b8cd](https://github.com/vantist/time-tracker/commit/8a1b8cd2a0da027940baa006e470aa3613c622ea))
* **pricing:** support cost calculation for single ModelUsage ([6878796](https://github.com/vantist/time-tracker/commit/68787960fd2736fe20db0c259066653b13b4ebb0))
* **recorder,reconcile:** write details to turn_model_usages on response and reconcile ([476a911](https://github.com/vantist/time-tracker/commit/476a9118d0e283d3794c6df89f658c25d636b013))
* **report:** add agent normalization and report data structures ([d529f36](https://github.com/vantist/time-tracker/commit/d529f365719b2de9e49f00196e3aa152e2c7b0b9))
* **report:** aggregate sessions and calculate user active time by agent ([15cba3c](https://github.com/vantist/time-tracker/commit/15cba3ce6d000be33f9e69aa37dd96b991b9f4ca))
* **report:** format by agent summary and include agent field in cli text output ([b72e095](https://github.com/vantist/time-tracker/commit/b72e095d4d255ce5beb0880087aabbc3c8639dde))
* **report:** load sessions tool column in SQL query ([54a4f14](https://github.com/vantist/time-tracker/commit/54a4f14dc4fc2ab1b13cfcde69f28a25f2bcf581))
* **report:** support model usage breakdown in CLI report ([a1c250b](https://github.com/vantist/time-tracker/commit/a1c250b849ac9d3a93cf5da3418325a7855aefff))
* **serve:** render by agent table and agent column in dashboard web ui ([a2d9ac8](https://github.com/vantist/time-tracker/commit/a2d9ac87b30e18da0f947c325ea91c1c1b2668b0))
* **transcript:** add ModelUsage structure and aggregate subagents by model ([983eea2](https://github.com/vantist/time-tracker/commit/983eea28781ff1eb358fd7bb8eda2301fe264cee))

## [1.6.0](https://github.com/vantist/time-tracker/compare/v1.5.0...v1.6.0) (2026-06-19)


### Features

* **report:** add project-specific input/output tokens and ByWorkItem field ([07d5c5a](https://github.com/vantist/time-tracker/commit/07d5c5a4f96e91f94ee964c2065905003634acf6))
* **report:** align FormatText to print structured ASCII tables ([05005ff](https://github.com/vantist/time-tracker/commit/05005ff157e031fd9f1b5028dded1ef6506f4b3b))
* **report:** simplify CLI report printing and format project dashboard tokens ([bccb713](https://github.com/vantist/time-tracker/commit/bccb713792992f7fcd2b1ce7729074e339c6149e))

## [1.5.0](https://github.com/vantist/time-tracker/compare/v1.4.0...v1.5.0) (2026-06-19)


### Features

* **aggregator:** 新增 Interval 型別及 UserIntervals、MergeAndSum 函數 ([61f869a](https://github.com/vantist/time-tracker/commit/61f869a7ec27969778f69e9e1b5f84d559e3cbb3))
* **db:** 新增 turns 表四個欄位（model、cache_creation_5m/1h、subagent_tokens_settled） ([c721854](https://github.com/vantist/time-tracker/commit/c721854105445a0ea51eb69f04c5fb20453e1fa4))
* ExtractLastTurn、WindowResult 整合、reconcile subagent_tokens_settled、pricing 5m/1h 費率 ([7e6880d](https://github.com/vantist/time-tracker/commit/7e6880d31bd48cd3adb8820d3af6dcab22d157d2))
* **transcript:** WindowResult typed struct + extractSubagentTokens to 邊界 + cache 5m/1h 欄位 ([320f091](https://github.com/vantist/time-tracker/commit/320f091fd61266e8138c136e29bdac8c509ab62a))


### Bug Fixes

* **aggregator:** UserIntervals 略過非正值 interval（守衛 d &gt; 0） ([dbf1398](https://github.com/vantist/time-tracker/commit/dbf1398ca61afc54fa0fa2ca2219b1c9fa1df362))
* **reconcile:** guard nextOffset by transcript path match ([935f18b](https://github.com/vantist/time-tracker/commit/935f18b3bdf4584987bfb6eadb05ac2bc4b1dc39))
* **recorder:** countLines 改用 bufio.Scanner（1MB buffer）避免大 transcript 全量讀入記憶體 ([17ed65c](https://github.com/vantist/time-tracker/commit/17ed65c801d6ab62c80c8abfc1aca9734e2d0a3a))
* **report:** 改用 interval-based user time 聚合，支援多 session merge 去重 ([6b2aca6](https://github.com/vantist/time-tracker/commit/6b2aca666b36becac0ee94e3272a6ff80897e10c))
* **report:** 移除 groupByWorkItem 未使用的 idleThreshold 參數 ([e3797d6](https://github.com/vantist/time-tracker/commit/e3797d6b319c43729802463f5515e04e24030e80))
* **token-tracking:** 修正 code review 發現的 5 個 bug ([99d33b7](https://github.com/vantist/time-tracker/commit/99d33b77df852abc636979ecc23cc9a70a305fe3))

## [1.4.0](https://github.com/vantist/time-tracker/compare/v1.3.2...v1.4.0) (2026-06-18)


### Features

* **dashboard:** By Work Item table 新增 Project 欄位 ([8bf3201](https://github.com/vantist/time-tracker/commit/8bf3201292dff954a9ba73c7c0963c1a4a197a7a))
* **report:** 以複合 key 修正 work item 跨 repo label 碰撞 ([5aed2d5](https://github.com/vantist/time-tracker/commit/5aed2d561176ac2696867585842cf90c1fdba31e))


### Bug Fixes

* **report:** 用 struct key 取代 | 分隔符以防止 work_item 碰撞 ([d7ee48a](https://github.com/vantist/time-tracker/commit/d7ee48a9d6a6049bb50bbaa7debc549e4bbbbb0a))
* **transcript:** 將 content 欄位移至 message 底下以修正 subagent token 遺失 ([766164e](https://github.com/vantist/time-tracker/commit/766164e62e06056ca702d1676bf2e745fbd74d04))

## [1.3.2](https://github.com/vantist/time-tracker/compare/v1.3.1...v1.3.2) (2026-06-18)


### Bug Fixes

* **install:** 修正安裝腳本中的儲存庫名稱 ([a6b406a](https://github.com/vantist/time-tracker/commit/a6b406a251a9c3ba85297f3a06b4e63ed16f476e))

## [1.3.1](https://github.com/vantist/time-checker/compare/v1.3.0...v1.3.1) (2026-06-18)


### Bug Fixes

* **workflow:** 修正自動合併釋出 PR 的命令選項 ([bfe8344](https://github.com/vantist/time-checker/commit/bfe8344defeff227233abca45284da7b032dc81f))

## [1.3.0](https://github.com/vantist/time-checker/compare/v1.2.0...v1.3.0) (2026-06-18)


### Features

* **docs:** 新增 README 文件以說明 AI 工作時間追蹤器的安裝與使用 ([26cf93c](https://github.com/vantist/time-checker/commit/26cf93c54329e5680b7c0a1a3f9916d3017e9259))
* **install:** 更新安裝腳本以支持自定義安裝目錄 ([cf1ae88](https://github.com/vantist/time-checker/commit/cf1ae88724567e7f8230eb782383ab2975709206))

## [1.2.0](https://github.com/vantist/time-checker/compare/v1.1.0...v1.2.0) (2026-06-18)


### Features

* **aggregator,reconcile,report:** include session-start gap in user time; backfill tokens when response_at set but tokens null ([0bafad7](https://github.com/vantist/time-checker/commit/0bafad7d51f35302cf5a83553b54e1aca892f15e))
* **cli:** read PROCESS_PID/PROCESS_START env vars in tt record prompt ([3d1b0e6](https://github.com/vantist/time-checker/commit/3d1b0e60a085a61d4d4c3e2632c252aa20399243))
* **cmd:** add --transcript-path flag to tt record prompt ([0873b4b](https://github.com/vantist/time-checker/commit/0873b4b36e99d9703f07d8042e80a9b9cc38a911))
* **cmd:** add contentBlock and subagentMeta structs for subagent token capture ([1ad23b6](https://github.com/vantist/time-checker/commit/1ad23b6ae157a17816ee8b6d559a44bb9ebc3d3c))
* **cmd:** add extractFromTranscriptAtOffset for anchor-based token extraction ([d9136dc](https://github.com/vantist/time-checker/commit/d9136dcf44f8e335d91579c6b43d68314868ed50))
* **cmd:** implement extractSubagentTokens with tests (TDD) ([7fa372f](https://github.com/vantist/time-checker/commit/7fa372f6640f2fa347bdc23df94f2231a1913598))
* **cmd:** integrate subagent tokens into extractFromTranscriptAtOffset ([284738a](https://github.com/vantist/time-checker/commit/284738a77438010ac8ba92ae4416875a03860099))
* **cmd:** use stored prompt_line_offset for anchor-based token extraction ([66d3516](https://github.com/vantist/time-checker/commit/66d3516521ba37b407822bdc8c5e98fd3143e686))
* **cmd:** wire MaybeReconcile into serve, report, and /api/report handler ([aaa3581](https://github.com/vantist/time-checker/commit/aaa3581bc15838c220794ec7f3f37855aa7c3399))
* **dashboard:** add user time/work item columns and By Work Item section ([9b9e64c](https://github.com/vantist/time-checker/commit/9b9e64c798f09967f9cf4273ac0cc87ad5e39ef0))
* **db,recorder:** add transcript_path + prompt_line_offset to turns ([0829e4e](https://github.com/vantist/time-checker/commit/0829e4e08efdb66dd5811972fe9bbe1929e881d7))
* **db:** add process_pid, process_start, conversation_id columns via PRAGMA-based migration ([feddc5d](https://github.com/vantist/time-checker/commit/feddc5d39758d7b5484cedeb242ce8811417cdd6))
* **db:** add ProcessPID/ProcessStart/ConversationID to Session; stable-key upsert logic ([7c7f698](https://github.com/vantist/time-checker/commit/7c7f69891f8f03a73c508c9eaac7b9fb57694f85))
* **db:** reuse session row on claude --resume ([fcc37b9](https://github.com/vantist/time-checker/commit/fcc37b991a7214fc7de561cf64c3fed4add004f9))
* **pricing:** normalize gateway prefix and update pricing table ([d8b23f7](https://github.com/vantist/time-checker/commit/d8b23f720cc00309ed8180bdaa1537661368020e))
* **process:** add cross-platform StartTime package (darwin + other) ([fdc8884](https://github.com/vantist/time-checker/commit/fdc8884b4b57b023de1f89adb62489e21bdcb033))
* **process:** add Windows StartTime via GetProcessTimes ([586954d](https://github.com/vantist/time-checker/commit/586954d904d26b71697b252e769859b89254bcff))
* **reconcile:** add cross-process flock lock with unix/windows implementations ([e188136](https://github.com/vantist/time-checker/commit/e18813645e22a899b8b53ca983ed895c3ca399b9))
* **reconcile:** implement MaybeReconcile, hasActiveSession, and dangling turn backfill ([ff2cd3e](https://github.com/vantist/time-checker/commit/ff2cd3e4e4c0ec1bdfb7c905efdf2ad480c0e65b))
* **recorder,db:** UpsertSession returns stable ID; turns use stable session ID across /clear ([8a7808f](https://github.com/vantist/time-checker/commit/8a7808f9b6f0333baab9dd05f118c1356bcf202f))
* **recorder:** add ProcessPID/ProcessStart to PromptInput; pass to UpsertSession ([56c8e29](https://github.com/vantist/time-checker/commit/56c8e296b72f5a9bb3b4fb56cc2fd3ba481fde85))
* **recorder:** extract model from transcript and backfill sessions.model ([12fd328](https://github.com/vantist/time-checker/commit/12fd3284f768fd8e6f8e9ac1efbd737bb7ef85d8))
* **record:** resolve parent PID/start via process.StartTime when env vars absent ([ae9f56d](https://github.com/vantist/time-checker/commit/ae9f56d31b8911e9a2141dfe95a569e270095673))
* **report:** add CacheCreationTokens, ByProject, DailyStat to Result struct and Query ([7f5ebab](https://github.com/vantist/time-checker/commit/7f5ebab639733e6869db89f3206f0bb6af6999aa))
* **report:** add html.go with HandleDashboard and HandleAPIReport ([147236c](https://github.com/vantist/time-checker/commit/147236cfbad2ff0be1532631466431d0e76dfffb))
* **report:** add UserActiveTimeSec/UserTimeSec/WorkItem fields, always compute Groups ([4221528](https://github.com/vantist/time-checker/commit/4221528e2f02c9b6bc7c59439dcb38bc154e85bd))
* **report:** FormatJSON adds cache_creation_tokens, cache_read_tokens, by_project, daily fields ([6850825](https://github.com/vantist/time-checker/commit/6850825f2abd73b559eefd85762349eecba00bf2))
* **report:** FormatText with Tokens/Cost/ByProject blocks and formatInt helper ([24d67fa](https://github.com/vantist/time-checker/commit/24d67fae7fa924c2e7c1fe7f8e89fada9d3f769a))
* **setup:** idempotent hook setup via _owner marker ([79f4a5d](https://github.com/vantist/time-checker/commit/79f4a5d204006fa0005f479b66d24d7a94b64f0c))
* **setup:** simplify UserPromptSubmit hook to "tt record prompt" ([0812f82](https://github.com/vantist/time-checker/commit/0812f82e201bf0d034483c74c7f524206d116faf))
* **setup:** update UserPromptSubmit hook to pass PROCESS_PID/PROCESS_START ([0abd47b](https://github.com/vantist/time-checker/commit/0abd47b127e6efb52ab28d2c2cb4a2d2442e7775))
* **spec:** 新增多項規範文件以支持成本追蹤、報告查詢及會話管理 ([d981238](https://github.com/vantist/time-checker/commit/d981238d21fb569893cd8fa01f78e08d323521fb))
* **transcript:** extract ExtractWindow, extractSubagentTokens into internal/transcript package ([24f79d1](https://github.com/vantist/time-checker/commit/24f79d194f342e8e2b8f2daddfdd4992cba89e40))
* **tt:** add serve subcommand with web dashboard at port 7890 ([302fe66](https://github.com/vantist/time-checker/commit/302fe6614fdb62e542621417c161d3822213fd44))
* **work,recorder:** pass CWD as project to workitem API ([ac0bc64](https://github.com/vantist/time-checker/commit/ac0bc6467aa0a3b0b1cbc0e9d7f52d1d5de52861))
* **workitem:** per-project work item storage with git root resolution ([037abde](https://github.com/vantist/time-checker/commit/037abde6fb8336f0c3f2765405db91d1e6e3e7f3))


### Bug Fixes

* **recorder,cmd:** address code review findings ([b1ed056](https://github.com/vantist/time-checker/commit/b1ed056ebcd20ad5d2c75d11356abf09731238bf))
* **record:** fallback to ppid when env var override parse fails; extract sumWindow ([9b76f76](https://github.com/vantist/time-checker/commit/9b76f76e33e0ec2152c1df5099b01eeecf7cd7e4))
* **report:** remove dead ByProject loop in Query ([e8f1223](https://github.com/vantist/time-checker/commit/e8f122349c2cb6df024fbbfe72bb2cc2b6cabb4c))
* **review:** suppress warning spam, explicit fallback error discard, fix test assertion ([830870b](https://github.com/vantist/time-checker/commit/830870b4d2b65fcf33dfca4e094839adca9b6166))
* three code-review findings from session review ([f6e0e48](https://github.com/vantist/time-checker/commit/f6e0e48f5ac4ab19f6c24e263e7732a33ab1053e))

## [1.1.0](https://github.com/vantist/time-checker/compare/v1.0.0...v1.1.0) (2026-06-17)


### Features

* **github-cicd-release:** 建立 GitHub CI/CD 發布流程 ([fa75195](https://github.com/vantist/time-checker/commit/fa7519504e2f23449b7905c2aad44a0b66ec6654))
* **hook-integration:** 新增 Claude Code 與 Copilot CLI hooks 設定及事件處理 ([fa75195](https://github.com/vantist/time-checker/commit/fa7519504e2f23449b7905c2aad44a0b66ec6654))
* **report-query:** 新增報表查詢功能 ([fa75195](https://github.com/vantist/time-checker/commit/fa7519504e2f23449b7905c2aad44a0b66ec6654))
* **session-management:** 新增 session 與 turn 資料模型 ([fa75195](https://github.com/vantist/time-checker/commit/fa7519504e2f23449b7905c2aad44a0b66ec6654))
* **time-aggregation:** 新增時間聚合計算邏輯 ([fa75195](https://github.com/vantist/time-checker/commit/fa7519504e2f23449b7905c2aad44a0b66ec6654))
* **work-item-tagging:** 新增工作項目標記管理 ([fa75195](https://github.com/vantist/time-checker/commit/fa7519504e2f23449b7905c2aad44a0b66ec6654))

## 1.0.0 (2026-06-17)


### Features

* add Session type and UpsertSession with INSERT OR IGNORE semantics ([5a14e03](https://github.com/vantist/time-checker/commit/5a14e03e6e99f43f45fb9024b849541461d9f39e))
* config management (idle-threshold), tt config set, report reads config ([1958b4f](https://github.com/vantist/time-checker/commit/1958b4f590adcf917a24bd21bed120668e2804bf))
* implement RecordPrompt with session upsert and turn insertion ([d955fa8](https://github.com/vantist/time-checker/commit/d955fa8d6ca22f54e912cb31363945cac18c1548))
* initialize tt project with Go module, cobra root, and DB schema ([71d520f](https://github.com/vantist/time-checker/commit/71d520fa8faf57967e176f2a4606a07ef0ea465c))
* RecordResponse, silent error wrappers, pricing, and record subcommands ([1152b18](https://github.com/vantist/time-checker/commit/1152b181172900ad90c6a7f6af85e03a4adb92a6))
* time aggregation, report query, text/json/by-work-item output, tt report command ([ad4caf2](https://github.com/vantist/time-checker/commit/ad4caf2ce59fc6f7c21fc5e39c016ca10a84d140))
* tt setup --claude-code merges hooks, --copilot prints instructions ([9c6e247](https://github.com/vantist/time-checker/commit/9c6e247db4b3aea8ad80369c317bba68980818ed))
* work item tagging — Set/Get/Clear, tt work command, RecordPrompt integration ([faf9215](https://github.com/vantist/time-checker/commit/faf9215dc921fb68a770958295cbfabe300d4cfe))


### Bug Fixes

* guard type assertions in setup merge, simplify token JSON parsing ([976730f](https://github.com/vantist/time-checker/commit/976730fa1086995f4b6f291e66568d4a82cb2ade))
* use PAT for release-please to bypass PR creation restriction ([2082b38](https://github.com/vantist/time-checker/commit/2082b38ff1132f0d627b2f7376bd46b09bc65e99))
* use PAT secret for release-please to allow PR creation ([0fe308f](https://github.com/vantist/time-checker/commit/0fe308f169b432166cb6adecf9fc7b8c63098f9e))
