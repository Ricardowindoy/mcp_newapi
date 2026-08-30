client — new-api 管理面 HTTP 传输层（internal/newapi 根包）。Bearer PAT 鉴权、统一信封 {success,message,data} 解包、APIError 错误分类（可达性+状态码）；routes.go 是上游端点唯一耦合点；另有掩码/分页/业务常量共享件。零业务逻辑。
