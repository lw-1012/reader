# Reader

个人用英语原著简化阅读器：

- 把英文原著按 CEFR 难度（A2 / B1 / B2…）逐段简化
- 简化版逐句分析（中文翻译 + 词汇 + 语法点）
- 点单词或句子朗读（TTS）
- 多本书的书架 + 阅读进度同步
- 移动端为主，左右滑动翻段，支持加入主屏 (PWA)

技术栈：Go (`net/http` + `modernc.org/sqlite`，纯静态二进制) + Svelte 前端，前端编译产物用 `embed` 内嵌到 Go 二进制。单容器部署，正常内存占用 < 30MB。

LLM 与 TTS 统一走 OpenRouter（OpenAI 兼容的 `/chat/completions` + `/audio/speech`）。

---

## 部署

```bash
# 1. 设置密码
export READER_PASSWORD='your-strong-password'

# 2. 启动
docker compose up -d --build
```

默认监听 `:8080`。建议放在反向代理（Caddy / Traefik / Nginx）后面挂 HTTPS 再公网暴露。

数据持久化在 `reader_data` volume 里：

- `reader.db` — SQLite 主库
- `tts/*.mp3` — TTS 音频缓存（按 `sha256(model+voice+instruction+text)` 命名）

### 不用 Docker 直接跑

```bash
cd frontend && npm install && npm run build     # 把前端打到 backend/webui
cd ../backend && CGO_ENABLED=0 go build -o reader .
READER_PASSWORD=xxx ./reader
```

---

## 首次使用

1. 浏览器或手机打开 `http://your-host:8080/`，输入密码登录。
2. 右上 ⚙ → 填 OpenRouter API Key、模型、声音。默认值：
   - 简化 / 分析：`openai/gpt-4o-mini`
   - TTS：`openai/gpt-4o-mini-tts`，voice `alloy`
   - 难度：`B1`
3. 回到书架 → 右上 ＋ 上传 JSON。
4. 点书进入阅读，左右滑动翻段；切到「简化」tab 按需触发简化；点句末 💡 看逐句分析；点任意单词朗读。

---

## 书籍 JSON 格式

为了适配各种结构的书，导入接受 **三种嵌套形态**，统一在内部存为「树形 sections + 扁平 paragraphs」：

```jsonc
// 形态 A：章 → 节 → 段（已含的「欲望的演化」就是这种）
{
  "title": "Book title",
  "author": "Author",
  "chapters": [
    {
      "chapter_number": 1,
      "title": "Chapter 1 title",
      "sections": [
        {
          "title": "Section title",
          "paragraphs": [
            { "paragraph_id": "ch01-sec01-p001", "text": "..." }
          ]
        }
      ]
    }
  ]
}

// 形态 B：自定义层级（递归）
{
  "title": "...",
  "sections": [
    {
      "title": "Part I",
      "sections": [
        { "title": "Chapter 1", "paragraphs": [ { "text": "..." } ] }
      ]
    }
  ]
}

// 形态 C：完全扁平
{
  "title": "...",
  "paragraphs": [ { "text": "..." }, { "text": "..." } ]
}
```

> 任意一本书都能塞进树形结构：每段挂在它的「最近 section」上；展示时按章节树展开即可。

`sample/the_evolution_of_desire.json` 是个现成例子（903 段，10 章）。

---

## API（认证后）

| 方法+路径 | 说明 |
|---|---|
| `POST /api/auth/login` `{password}` | 登录，返回 cookie |
| `POST /api/auth/logout` | 退出 |
| `GET  /api/auth/check` | 当前是否已认证 |
| `GET  /api/settings` | 读取设置（api_key 已遮蔽） |
| `PUT  /api/settings` | 部分更新（只传要改的字段；`api_key` 留空或 `********` 表示不改） |
| `GET  /api/books` | 书架 |
| `POST /api/books/import` | 上传 JSON |
| `GET  /api/books/{id}` | 元信息 + 章节树（含每节段落范围） |
| `DELETE /api/books/{id}` | 删除 |
| `GET  /api/books/{id}/paragraphs?from=&to=` | 取段落（按 global_index 区间，单次最多 50） |
| `POST /api/paragraphs/{id}/simplify?level=B1&force=1` | 简化（缓存按 paragraph+level+prompt_hash） |
| `POST /api/analyze` `{text, level}` | 逐句分析（缓存按 sentence+level+prompt_hash） |
| `GET  /api/tts?text=&voice=` | TTS 音频（mp3，缓存到磁盘） |
| `GET  /api/books/{id}/progress` | 读取进度 |
| `PUT  /api/books/{id}/progress` `{paragraph_id, level}` | 更新进度 |

---

## 缓存策略

| 内容 | Key | 失效条件 |
|---|---|---|
| 简化版段落 | `(paragraph_id, level, sha256(simplify_prompt))` | 改简化 Prompt 后自动失效；UI 也支持「重新生成简化」按钮 |
| 句子分析 | `(sha256(sentence), level, sha256(analyze_prompt))` | 改分析 Prompt 自动失效 |
| TTS 音频 | `sha256(tts_model + voice + tts_instruction + text)` | 改 voice / instruction / 模型自动失效；磁盘保存 |

---

## 资源占用 / 限制

- 运行内存 20-30MB（实测空载 19MB，加载本书后未明显增长）；compose 设了 128MB 上限
- 单二进制 ~11MB，前端 bundle gzip 后 ~20KB
- TTS 文本限 4000 字符/次（句子级足够）
- 段落分页一次最多 50 段

---

## 修改提示词

`设置 → 提示词` 内编辑。两个占位符：

- `{LEVEL}` — 当前难度（A1…C2）
- `{TEXT}` — 仅 `analyze_prompt` 用，简化 prompt 中原文会自动拼在末尾

模型返回必须是 JSON。简化返回 `{simplified, sentences}`，分析返回 `{translation, vocab, grammar, notes}`。

---

## 安全注意

- 默认密码 `reader`，强烈建议通过 `READER_PASSWORD` 环境变量改掉
- API Key 明文存于 SQLite（个人单机用够了，别共享 volume）
- Cookie 在 HTTPS 下自动加 Secure flag
