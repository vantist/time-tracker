# Changelog

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
