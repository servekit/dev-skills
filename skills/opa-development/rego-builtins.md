---
name: rego-builtins
description: "Rego built-in functions reference — load when needing any string/regex/set/object/array/time/crypto/net/http/encoding/aggregation operation in Rego policies. Contains ~144 functions organized by category with signatures and examples."
---

# Rego 内置函数参考

OPA 提供约 144 个内置函数。运行时错误默认返回 `undefined`（不终止评估），除非使用 `--strict-builtin-errors`。

所有正则表达式使用 **RE2 语法**（不支持回溯引用）。

---

## 字符串 (Strings) — 25 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `concat(delimiter, collection)` | 连接字符串集合 | `concat(",", {"a","b"})` → `"a,b"` |
| `contains(haystack, needle)` | 子串检查 | `contains("hello","ell")` → `true` |
| `endswith(search, base)` | 后缀检查 | `endswith("hello","llo")` → `true` |
| `startswith(search, base)` | 前缀检查 | `startswith("hello","hel")` → `true` |
| `sprintf(format, values)` | Go 风格格式化 | `sprintf("hi %s", ["world"])` → `"hi world"` |
| `lower(x)` | 转小写 | `lower("HELLO")` → `"hello"` |
| `upper(x)` | 转大写 | `upper("hello")` → `"HELLO"` |
| `replace(x, old, new)` | 替换所有匹配 | `replace("hello","l","r")` → `"herro"` |
| `split(x, delimiter)` | 按分隔符拆分 | `split("a,b",",")` → `["a","b"]` |
| `trim(value, cutset)` | 去首尾字符 | `trim("...hi...",".")` → `"hi"` |
| `trim_left(value, cutset)` | 去左侧字符 | |
| `trim_right(value, cutset)` | 去右侧字符 | |
| `trim_prefix(value, prefix)` | 去前缀 | `trim_prefix("hello","hel")` → `"lo"` |
| `trim_suffix(value, suffix)` | 去后缀 | `trim_suffix("hello","llo")` → `"he"` |
| `trim_space(value)` | 去首尾空白 | |
| `substring(value, offset, length)` | 提取子串 | `substring("hello",1,3)` → `"ell"` |
| `indexof(haystack, needle)` | 首次出现位置（-1 未找到） | `indexof("hello","l")` → `2` |
| `indexof_n(haystack, needle)` | 所有出现位置 | `indexof_n("hello","l")` → `[2,3]` |
| `format_int(number, base)` | 数字转字符串（指定进制） | `format_int(255,16)` → `"ff"` |
| `strings.count(search, substring)` | 计数非重叠出现 | `strings.count("aaa","a")` → `3` |
| `strings.reverse(x)` | 反转字符串 | `strings.reverse("abc")` → `"cba"` |
| `strings.replace_n(patterns, value)` | 批量替换 | `strings.replace_n({"a":"x"},"abc")` → `"xbc"` |
| `strings.any_prefix_match(search, base)` | 任一前缀匹配 | |
| `strings.any_suffix_match(search, base)` | 任一后缀匹配 | |
| `strings.render_template(value, vars)` | Go text/template 渲染 | |

## 正则表达式 (Regex) — 8 个函数

所有正则使用 RE2 语法。**推荐用原始字符串**：`` `[a-z]+` ``

| 函数 | 说明 | 示例 |
|------|------|------|
| `regex.match(pattern, value)` | 正则匹配 | `regex.match("^[a-z]+$","hello")` → `true` |
| `regex.find_n(pattern, value, number)` | 前 N 个匹配 | `regex.find_n("a.","abacad",2)` → `["ab","ac"]` |
| `regex.find_all_string_submatch_n(pattern, value, number)` | 所有子匹配组 | |
| `regex.is_valid(pattern)` | 验证正则语法 | |
| `regex.replace(s, pattern, value)` | 正则替换 | `regex.replace("hello","l+","r")` → `"hero"` |
| `regex.split(pattern, value)` | 按正则拆分 | `regex.split("[,.]+","a,b.c")` → `["a","b","c"]` |
| `regex.globs_match(glob1, glob2)` | 两个 glob 是否有重叠 | |
| `regex.template_match(template, value, delim_start, delim_end)` | 带嵌入正则的模板匹配 | |

## 集合 (Sets) — 5 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `x & y` | 交集 | `{1,2,3} & {2,3,4}` → `{2,3}` |
| `x \| y` | 并集 | `{1,2} \| {2,3}` → `{1,2,3}` |
| `x - y` | 差集 | `{1,2,3} - {2}` → `{1,3}` |
| `intersection(xs)` | 集合的集合的交集 | `intersection({{1,2},{2,3}})` → `{2}` |
| `union(xs)` | 集合的集合的并集 | `union({{1,2},{2,3}})` → `{1,2,3}` |

## 对象 (Objects) — 13 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `object.get(object, key, default)` | 安全取值带默认 | `object.get({"a":1},"b",0)` → `0` |
| `object.keys(object)` | 所有键（集合） | `object.keys({"a":1})` → `{"a"}` |
| `object.filter(object, keys)` | 按键过滤 | `object.filter({"a":1,"b":2},{"a"})` → `{"a":1}` |
| `object.remove(object, keys)` | 移除指定键 | `object.remove({"a":1,"b":2},{"a"})` → `{"b":2}` |
| `object.union(a, b)` | 合并（b 覆盖 a） | `object.union({"a":1},{"a":2})` → `{"a":2}` |
| `object.union_n(objects)` | 合并多个对象 | |
| `object.subset(super, sub)` | 子集检查 | `object.subset({"a":1,"b":2},{"a":1})` → `true` |
| `json.filter(object, paths)` | 按路径过滤 | `json.filter({"a":{"b":1}},[["a","b"]])` → `{"a":{"b":1}}` |
| `json.remove(object, paths)` | 按路径移除 | |
| `json.patch(target, patches)` | JSON Patch (RFC 6902) | |
| `json.match_schema(doc, schema)` | JSON Schema 校验 | |
| `json.verify_schema(schema)` | 验证 Schema 有效性 | |
| `json.marshal_with_options(x, opts)` | 带选项序列化 | `json.marshal_with_options(x, {"pretty":true})` |

## 数组 (Arrays) — 4 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `array.concat(x, y)` | 拼接 | `array.concat([1,2],[3])` → `[1,2,3]` |
| `array.slice(arr, start, stop)` | 切片 | `array.slice([1,2,3,4],1,3)` → `[2,3]` |
| `array.reverse(arr)` | 反转 | |
| `array.flatten(arr)` | 展平（一层） | `array.flatten([[1,2],[3]])` → `[1,2,3]` |

## 聚合 (Aggregates) — 6 个函数

适用于集合、数组、对象。

| 函数 | 说明 | 示例 |
|------|------|------|
| `count(collection)` | 计数 | `count({1,2,3})` → `3`；`count("hi")` → `2` |
| `sum(collection)` | 求和 | `sum({1,2,3})` → `6` |
| `product(collection)` | 求积 | `product({2,3,4})` → `24` |
| `max(collection)` | 最大值 | |
| `min(collection)` | 最小值 | |
| `sort(collection)` | 排序（返回数组） | `sort({3,1,2})` → `[1,2,3]` |

## 编码 (Encoding) — 19 个函数

### Base64
| 函数 | 说明 |
|------|------|
| `base64.encode(x)` | 编码 |
| `base64.decode(x)` | 解码 |
| `base64.is_valid(x)` | 验证格式 |
| `base64url.encode(x)` | URL 安全编码 |
| `base64url.decode(x)` | URL 安全解码 |
| `base64url.encode_no_pad(x)` | URL 安全编码（无填充） |

### Hex
| 函数 | 说明 |
|------|------|
| `hex.encode(x)` | 编码 |
| `hex.decode(x)` | 解码 |

### JSON / YAML
| 函数 | 说明 |
|------|------|
| `json.marshal(x)` | 序列化为 JSON 字符串 |
| `json.unmarshal(x)` | 从 JSON 解析 |
| `json.is_valid(x)` | 验证 JSON 格式 |
| `yaml.marshal(x)` | 序列化为 YAML |
| `yaml.unmarshal(x)` | 从 YAML 解析 |
| `yaml.is_valid(x)` | 验证 YAML 格式 |

### URL Query
| 函数 | 说明 | 示例 |
|------|------|------|
| `urlquery.encode(x)` | 编码查询字符串 | |
| `urlquery.decode(x)` | 解码查询字符串 | |
| `urlquery.encode_object(object)` | 对象→查询字符串 | `urlquery.encode_object({"a":"1"})` → `"a=1"` |
| `urlquery.decode_object(x)` | 查询字符串→对象 | |

## 时间 (Time) — 10 个函数

所有时间以**纳秒 Unix 时间戳**表示。

| 函数 | 说明 | 示例 |
|------|------|------|
| `time.now_ns()` | 当前时间 | |
| `time.parse_ns(layout, value)` | 按 Go 布局解析 | `time.parse_ns("2006-01-02","2024-01-01")` |
| `time.parse_rfc3339_ns(value)` | 解析 RFC3339 | `time.parse_rfc3339_ns("2024-01-01T00:00:00Z")` |
| `time.parse_duration_ns(duration)` | 解析持续时间 | `time.parse_duration_ns("1h30m")` → `5400000000000` |
| `time.format(x)` | 格式化为 RFC3339 | |
| `time.format(x, layout)` | 按 Go 布局格式化 | `time.format(t, "2006-01-02")` |
| `time.date(x)` | `[year, month, day]` | |
| `time.date(x, timezone)` | 指定时区的日期 | `time.date(t, "America/New_York")` |
| `time.clock(x)` | `[hour, minute, second]` | |
| `time.clock(x, timezone)` | 指定时区的时钟 | `time.clock(t, "UTC")` |
| `time.weekday(x)` | 星期几（字符串） | |
| `time.diff(ns1, ns2)` | 时间差 `[years, months, days, hours, minutes, seconds]` | |
| `time.add_date(ns, years, months, days)` | 日期加减 | |

## 加密 (Crypto) — 15 个函数

### 哈希
| 函数 | 说明 |
|------|------|
| `crypto.md5(x)` | MD5 |
| `crypto.sha1(x)` | SHA1 |
| `crypto.sha256(x)` | SHA256 |

### HMAC
| 函数 | 说明 |
|------|------|
| `crypto.hmac.md5(x, key)` | HMAC-MD5 |
| `crypto.hmac.sha1(x, key)` | HMAC-SHA1 |
| `crypto.hmac.sha256(x, key)` | HMAC-SHA256 |
| `crypto.hmac.sha512(x, key)` | HMAC-SHA512 |
| `crypto.hmac.equal(mac1, mac2)` | 时间安全比较（防时序攻击） |

### X.509 证书
| 函数 | 说明 |
|------|------|
| `crypto.x509.parse_certificates(pem)` | 解析 PEM 证书链 |
| `crypto.x509.parse_and_verify_certificates(pem)` | 解析并验证证书链 |
| `crypto.x509.parse_and_verify_certificates_with_options(certs, options)` | 解析并验证证书链（带选项，v0.63.0+） |
| `crypto.x509.parse_keypair(cert, pem)` | 从证书和私钥解析密钥对（v0.53.0+） |
| `crypto.x509.parse_certificate_request(csr)` | 解析 CSR |
| `crypto.x509.parse_rsa_private_key(pem)` | 解析 RSA 私钥 |
| `crypto.parse_private_keys(pem)` | 解析私钥 |

### JWT (io.jwt)
| 函数 | 说明 |
|------|------|
| `io.jwt.decode(token)` | 解码 JWT（不验证）→ `[header, payload, signature]` |
| `io.jwt.verify_*_hs256/384/512(token, secret)` | HMAC 签名验证 |
| `io.jwt.verify_*_rs256/384/512(token, cert)` | RSA 签名验证 |
| `io.jwt.verify_*_ps256/384/512(token, cert)` | RSA-PSS 签名验证 |
| `io.jwt.verify_*_es256/384/512(token, cert)` | ECDSA 签名验证 |
| `io.jwt.verify_eddsa(token, certificate)` | EdDSA 签名验证（v1.8.0+） |
| `io.jwt.decode_verify(token, constraints)` | 解码并验证（推荐） |

### JWT 签名生成

| 函数 | 说明 |
|------|------|
| `io.jwt.encode_sign(headers, payload, key)` | 编码并签名 JWT（结构化参数） |
| `io.jwt.encode_sign_raw(headers, payload, key)` | 编码并签名 JWT（原始字符串参数） |

`key` 支持格式：`{"kty":"oct","k":"..."}`（HMAC）、`{"kty":"RSA","n":"...","e":"...","d":"..."}`（RSA）、`{"kty":"EC","x":"...","y":"...","d":"..."}`（EC）。

## 网络 (Net) — 7 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `net.cidr_contains(cidr, cidr_or_ip)` | CIDR 包含检查 | `net.cidr_contains("10.0.0.0/8","10.1.2.3")` → `true` |
| `net.cidr_intersects(cidr1, cidr2)` | CIDR 重叠检查 | |
| `net.cidr_expand(cidr)` | 展开为所有主机地址 | |
| `net.cidr_is_valid(cidr)` | 验证 CIDR 格式 | |
| `net.cidr_merge(addrs)` | 合并相邻 CIDR | |
| `net.cidr_contains_matches(cidrs, cidrs_or_ips)` | 批量 CIDR 匹配 | |
| `net.lookup_ip_addr(name)` | DNS 查询 | |

## HTTP — 1 个函数

`http.send(request)` — 发送 HTTP 请求。**注意**：会触发 I/O，影响性能。

```rego
resp := http.send({
    "method": "GET",
    "url": "https://api.example.com/data",
    "headers": {"Authorization": "Bearer token"},
    "timeout": "5s",
    "cache": true,  # 启用缓存（同一评估周期内）
})
```

常用字段：`url`, `method`, `headers`, `body`, `raw_body`, `timeout`, `enable_redirect`, `cache`, `force_cache`, `force_cache_duration_seconds`, `tls_use_system_certs`, `tls_ca_cert_file`, `tls_insecure_skip_verify`, `raise_error`（默认 true，设为 false 时错误返回 undefined）, `max_retry_attempts`

## 数字 (Numbers) — 12 个函数

| 函数 | 说明 |
|------|------|
| `abs(x)` | 绝对值 |
| `ceil(x)` | 向上取整 |
| `floor(x)` | 向下取整 |
| `round(x)` | 四舍五入 |
| `rand.intn(str, n)` | 确定性伪随机 [0, n)，str 为种子 |
| `numbers.range(a, b)` | 闭区间数组 `[a, a+1, ..., b]` |
| `numbers.range_step(a, b, step)` | 带步长区间 |

算术运算符：`+`, `-`, `*`, `/`（整数除法）, `%`

## 图 (Graph) — 3 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `graph.reachable(graph, initial)` | 可达节点集合 | |
| `graph.reachable_paths(graph, initial)` | 可达路径数组 | |
| `walk(x, output)` | 递归遍历 `[path, value]` | |

## Glob — 2 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `glob.match(pattern, delimiters, match)` | Glob 匹配 | `glob.match("*.txt",["/"],"hi.txt")` → `true` |
| `glob.quote_meta(pattern)` | 转义 glob 特殊字符 | |

## 位运算 (Bits) — 6 个函数

| 函数 | 说明 |
|------|------|
| `bits.or(a, b)` | 按位或 |
| `bits.and(a, b)` | 按位与 |
| `bits.xor(a, b)` | 按位异或 |
| `bits.negate(x)` | 按位取反 |
| `bits.lsh(x, s)` | 左移 |
| `bits.rsh(x, s)` | 右移 |

## 类型检查 — 8 个函数

| 函数 | 说明 |
|------|------|
| `is_number(x)` | 是否为数字 |
| `is_string(x)` | 是否为字符串 |
| `is_boolean(x)` | 是否为布尔 |
| `is_array(x)` | 是否为数组 |
| `is_set(x)` | 是否为集合 |
| `is_object(x)` | 是否为对象 |
| `is_null(x)` | 是否为 null |
| `type_name(x)` | 类型名称字符串 |

## 类型转换

| 函数 | 说明 |
|------|------|
| `to_number(x)` | 字符串/布尔转数字 |

## 单位换算

| 函数 | 说明 |
|------|------|
| `units.parse_bytes(s)` | 解析字节字符串（"1KB"→1024） |
| `units.parse(x)` | 通用单位解析（"1KB"→1024，"2MB"→2097152 等） |

## UUID

| 函数 | 说明 |
|------|------|
| `uuid.rfc4122(s)` | 基于 seeds 生成确定性 UUID |

## GraphQL — 6 个函数

| 函数 | 说明 |
|------|------|
| `graphql.is_valid(query)` | 验证 GraphQL 查询语法 |
| `graphql.parse(query)` | 解析 GraphQL 查询为 AST |
| `graphql.parse_and_verify(query)` | 解析并验证 GraphQL 查询 |
| `graphql.parse_query(query)` | 解析 GraphQL 操作（query/mutation/subscription） |
| `graphql.parse_schema(schema)` | 解析 GraphQL schema 定义 |
| `graphql.schema_is_valid(schema)` | 验证 GraphQL schema 语法 |

## URI — 2 个函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `uri.is_valid(uri)` | 验证 URI 格式 | `uri.is_valid("https://example.com")` → `true` |
| `uri.parse(uri)` | 解析 URI 为对象（scheme/host/path/query 等） | `uri.parse("https://a.com/path?q=1")` → `{"scheme": "https", "host": "a.com", ...}` |

## Semver — 2 个函数

| 函数 | 说明 |
|------|------|
| `semver.is_valid(vsn)` | 验证语义化版本字符串 |
| `semver.compare(a, b)` | 比较两个语义化版本（-1/0/1） |

## Rego 元数据 — 3 个函数

| 函数 | 说明 |
|------|------|
| `rego.metadata.chain()` | 返回当前规则的元数据链（从内到外） |
| `rego.metadata.rule()` | 返回当前规则的元数据 |
| `rego.parse_module(filename, rego)` | 解析 Rego 模块为 AST |

## AWS — 1 个函数

| 函数 | 说明 |
|------|------|
| `providers.aws.sign_req(request, aws_config, time_ns)` | 对 HTTP 请求进行 AWS SigV4 签名 |

## 调试追踪 — 1 个函数

| 函数 | 说明 |
|------|------|
| `trace(note)` | 输出调试追踪信息（仅在使用 `--explain` 时可见） |

## OPA 运行时 — 1 个函数

| 函数 | 说明 |
|------|------|
| `opa.runtime()` | 返回 OPA 运行时信息（配置、环境变量等） |
