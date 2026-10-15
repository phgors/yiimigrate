# 未映射的列类型一律报 UnsupportedOperationError，不做透传

四个 adapter 的类型映射 `default` 分支原先把未识别的类型名直接大写后拼进 SQL（静默产出垃圾语句），sqlserver 另有一套 enum/set `return ""` 空串机制，两种失败模式并存。决定：映射未命中统一返回 unsupported 错误，四方言同一失败模式；空串机制废弃。透传仅在将来出现显式 opt-in 时才允许。

## Considered Options

- 保留透传 + 白名单 opt-in：兼容「故意拼 SQL」的用法，但违背仓库 explicit-unsupported 政策，被否决。
