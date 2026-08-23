// Package webui serves a lightweight single-page editor for the proofreading
// workbench. All data comes from the /api endpoints, so the page always shows
// server-computed results. The page is intentionally small; the business logic
// lives in the Go packages.
package webui

import "net/http"

const page = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>无障碍字幕时轴校对工作台</title>
<style>
body{font-family:-apple-system,"PingFang SC",sans-serif;margin:0;background:#f5f6f8;color:#1f2329}
header{background:#1f2937;color:#fff;padding:14px 20px}
h1{font-size:18px;margin:0}
main{max-width:960px;margin:20px auto;padding:0 16px}
.card{background:#fff;border-radius:8px;padding:16px;margin-bottom:16px;box-shadow:0 1px 3px rgba(0,0,0,.08)}
.row{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
button{background:#2563eb;color:#fff;border:0;border-radius:6px;padding:7px 12px;cursor:pointer}
button.ghost{background:#e5e7eb;color:#1f2329}
input,select{padding:7px;border:1px solid #d1d5db;border-radius:6px}
table{width:100%;border-collapse:collapse;font-size:14px}
th,td{padding:8px 6px;text-align:left;border-bottom:1px solid #eef0f3;vertical-align:top}
.badge{display:inline-block;padding:2px 8px;border-radius:10px;font-size:12px;margin:1px}
.err{background:#fee2e2;color:#b91c1c}
.warn{background:#fef3c7;color:#b45309}
.info{background:#dbeafe;color:#1d4ed8}
.ok{background:#dcfce7;color:#166534}
.small{font-size:12px;color:#6b7280}
#msg{margin-top:10px;font-size:13px;color:#b91c1c;white-space:pre-wrap}
</style>
</head>
<body>
<header><h1>无障碍字幕时轴校对工作台</h1></header>
<main>
  <div class="card row">
    <button onclick="demo()">载入示例数据</button>
    <button class="ghost" onclick="refresh()">刷新</button>
    <select id="media" onchange="loadMedia(this.value)"></select>
    <span class="small" id="summary"></span>
  </div>
  <div class="card">
    <h3>片段列表</h3>
    <div class="row" style="margin-bottom:8px">
      <button onclick="loadSegments()">加载片段</button>
      <button class="ghost" onclick="loadQuality()">质量检查</button>
      <button class="ghost" onclick="loadVersions()">发布版本</button>
      <button class="ghost" onclick="loadConflicts()">编辑冲突</button>
      <button onclick="publishNow()">发布版本</button>
    </div>
    <div id="segments"><p class="small">尚未加载。请先载入示例数据或选择媒体。</p></div>
  </div>
  <div class="card">
    <h3>编辑片段文本</h3>
    <div class="row">
      <input id="segId" placeholder="片段ID" style="width:220px">
      <input id="baseVer" type="number" placeholder="base_version" style="width:110px">
      <input id="actor" placeholder="操作人" style="width:110px">
      <input id="newText" placeholder="新文本" style="width:260px">
      <button onclick="editSegment()">提交编辑</button>
    </div>
    <div id="msg"></div>
  </div>
  <div class="card"><div id="quality"></div></div>
  <div class="card"><div id="versions"></div></div>
  <div class="card"><div id="conflicts"></div></div>
</main>
<script>
let current = "";
async function j(url, opt) { const r = await fetch(url, opt); const t = await r.json(); if (!r.ok) throw new Error((t.error||'') + (t.conflict ? ' | ' + JSON.stringify(t.conflict) : '')); return t; }
async function demo() { const d = await j('/api/demo', {method:'POST'}); await refresh(); if (d.media_id) { await loadMedia(d.media_id); } msg('示例数据已载入: ' + JSON.stringify(d)); }
async function refresh() {
  const list = await j('/api/media'); const sel = document.getElementById('media');
  sel.innerHTML = list.map(m => '<option value="'+m.id+'">'+m.title+' ('+m.status+')</option>').join('');
  if (list.length) await loadMedia(sel.value);
  const st = await j('/api/stats'); document.getElementById('summary').textContent = '媒体'+st.media+' 片段'+st.segments+' 质量项'+st.quality+' 版本'+st.versions+' 冲突'+st.conflicts;
}
async function loadMedia(id) { current = id; await loadSegments(); await loadQuality(); await loadVersions(); await loadConflicts(); }
function esc(s){return String(s||'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));}
async function loadSegments() {
  if(!current) return; const segs = await j('/api/media/'+current+'/segments');
  const rows = segs.map(s => '<tr><td>#'+s.index+'</td><td>'+s.start_ms+'..'+s.end_ms+'ms</td><td>'+esc(s.text)+'</td><td>'+esc(s.speaker_id)+'</td><td>v'+s.version+'</td><td>'+esc(s.status)+'</td></tr>').join('');
  document.getElementById('segments').innerHTML = segs.length ? '<table><tr><th>#</th><th>时轴</th><th>文本</th><th>说话人</th><th>版本</th><th>状态</th></tr>'+rows+'</table>' : '<p class="small">无片段</p>';
}
async function loadQuality() {
  if(!current) return; const q = await j('/api/media/'+current+'/quality');
  const s = await j('/api/media/'+current+'/quality/summary');
  const badges = q.map(x => '<span class="badge '+(x.severity==='error'?'err':x.severity==='warning'?'warn':'info')+'">'+x.rule+'@#'+(x.segment_id||'').split('_').pop()+'</span>').join('');
  document.getElementById('quality').innerHTML = '<h3>质量检查</h3><p>'+badges+'</p><p class="small">共'+s.total+'项，错误'+s.errors+'，警告'+s.warnings+'，重叠'+s.overlaps+'，空洞'+s.gaps+'，超速'+s.overspeed+'，空行'+s.empties+'</p>';
}
async function loadVersions() {
  if(!current) return; const v = await j('/api/media/'+current+'/versions');
  document.getElementById('versions').innerHTML = '<h3>发布版本</h3>' + (v.length ? v.map(x=>'<span class="badge '+(x.withdrawn?'warn':'ok')+'">v'+x.version_no+(x.withdrawn?'(已撤回)':'')+'</span>').join('') : '<span class="small">暂无</span>');
}
async function loadConflicts() {
  if(!current) return; const c = await j('/api/media/'+current+'/conflicts');
  document.getElementById('conflicts').innerHTML = '<h3>编辑冲突</h3>' + (c.length ? c.map(x=>'<p class="small">'+esc(x.explanation)+'</p>').join('') : '<span class="small">无</span>');
}
async function publishNow() {
  if(!current) return msg('请先选择媒体'); const p = await j('/api/media/'+current+'/publish', {method:'POST', headers:{'content-type':'application/json'}, body:JSON.stringify({actor:'web', label:'web-publish'})}); msg('已发布 v'+p.version_no); loadVersions();
}
async function editSegment() {
  const body = {actor: document.getElementById('actor').value, base_version: parseInt(document.getElementById('baseVer').value)};
  const t = document.getElementById('newText').value; if (t) body.text = t;
  try { const s = await j('/api/segments/'+document.getElementById('segId').value+'/edit', {method:'POST', headers:{'content-type':'application/json'}, body:JSON.stringify(body)}); msg('编辑成功 → v'+s.version); loadSegments(); loadQuality(); }
  catch(e){ msg('编辑失败: ' + e.message); loadConflicts(); }
}
function msg(m){ document.getElementById('msg').textContent = m; }
refresh();
</script>
</body>
</html>`

// Handler returns the embedded web UI.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("content-type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
}
