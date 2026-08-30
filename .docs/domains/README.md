domains — new-api 业务域子包集合（一域一包）。status（站点状态+relay 探测）、models（模型/定价）、channels（读/运维/管理三文件）、tokens（读写一体）、logs（检索/统计/按模型聚合）。每包 = raw DTO + Summary 裁剪 DTO + toSummary 构造时掩码 + 域函数 + 顶部上游契约注释。
