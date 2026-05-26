# convert 协议转换
convert 包提供了 OpenAI Chat、OpenAI Responses 和 Anthropic 三个协议之间的互转功能。

详细代码请查看 [convert 包](https://pkg.go.dev/github.com/llm-ux/llm-proxy/server/internal/handler/convert)。

对 convert 的约束：
1. 结构定义来自 openai 和 anthropic 官方SDK，分别是 https://www.npmjs.com/package/openai https://www.npmjs.com/package/@anthropic-ai/sdk。
2. 协议转换需要90%的代码覆盖；
3. 结构定义的字段补全注释说明使用场景，给出示例值；
4. 转换函数中对转换的字段加以说明：等价转换、兼容转换、不支持转换等；
5. 需要支持 sse 协议的转换；
6. 优先使用 interface 类型，方便未来扩展新的转换目标协议；
7. 不要使用 hard code, 使用 constants 定义，注释说明用途；
8. 安装不同的协议做代码拆分，避免代码或测试写在同一个文件；
9. 任务分阶段执行，每次执行后验证结果；
