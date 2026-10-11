package main

const uiHTML = `<!DOCTYPE html><html lang="pt-BR"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="theme-color" content="#07101f"><link rel="icon" href="/favicon.ico"><title>Consecom Plugin • Central de Controle</title><style>
:root{color-scheme:dark;--bg:#070f1d;--panel:#0d1d32;--card:#0d2037;--stroke:#23415f;--muted:#90acc6;--white:#eef8ff;--mint:#33e6b1;--blue:#2baaff;--warn:#f1ba53}
*{box-sizing:border-box}body{margin:0;background:radial-gradient(ellipse at 30% -15%,#063769 0%,transparent 50%),var(--bg);font:14px 'Segoe UI',Inter,Arial,sans-serif;color:var(--white);min-height:100vh}button,input{font:inherit}button{cursor:pointer;background:#152b46;color:#e7f5ff;border:1px solid #31516f;border-radius:11px;padding:11px 15px;font-weight:600}button:hover{background:#204268}button.primary{background:linear-gradient(115deg,#00aa7c,#17d5a0);border-color:#00dbb0;color:#fff}button.primary:hover{filter:brightness(1.12)}button:focus-visible,input:focus-visible,select:focus-visible{outline:2px solid #57dcff;outline-offset:2px}button:disabled{opacity:.5;cursor:not-allowed}.app{display:grid;grid-template-columns:250px 1fr;min-height:100vh}.side{background:linear-gradient(#0b1b30,#081525);border-right:1px solid #193a59;display:flex;flex-direction:column;padding:24px 16px;position:sticky;top:0;height:100vh}.logo{display:flex;align-items:center;gap:10px;font-size:18px;font-weight:800}.logo img{height:56px;width:56px;object-fit:contain}.side small{color:#86a9cb;display:block;margin:3px 0 32px;font-size:12px}.nav{display:flex;align-items:center;gap:12px;width:100%;border-color:transparent;background:transparent;text-align:left;margin-bottom:10px;padding:15px 14px}.nav.active{border-color:#146e8c;background:linear-gradient(90deg,#084a5d,#0c3455);color:#66ffdc}.nav .symbol{font-size:19px}.sideupdate{margin-top:auto;margin-bottom:12px}.update-button{width:100%;display:block;padding:13px 12px;text-align:center;border-radius:12px}.sideupdate-detail{margin-top:7px;text-align:center;color:#8baeca;font-size:11px;overflow-wrap:anywhere}.sidebottom{margin-top:0;background:#10243c;border:1px solid #1c496b;border-radius:14px;padding:15px;font-size:12px}.light{color:var(--mint)}.sidefooter{color:#7e9cba;margin-top:18px;font-size:12px}.main{padding:34px 30px 48px;max-width:1700px;width:100%;margin:0 auto;min-width:0}.top{display:flex;justify-content:space-between;align-items:flex-start;gap:15px}.top h1{font-size:30px;line-height:1.2;margin:0 0 7px;letter-spacing:-.6px}.sub{color:#a5bfd7;font-size:13px;line-height:1.6}.top-actions{display:flex;align-items:center;gap:12px;flex-wrap:wrap}.chip{border:1px solid #214465;background:#092139;padding:13px 15px;border-radius:14px;white-space:nowrap;font-size:12px}.dot{display:inline-block;width:9px;height:9px;background:#35ecae;border-radius:50%;box-shadow:0 0 12px #00d19b;margin-right:6px}.metrics{display:grid;grid-template-columns:repeat(5,minmax(0,1fr));gap:12px;margin:24px 0 14px}.metric{min-height:120px;border:1px solid #224361;border-radius:15px;background:linear-gradient(155deg,#0e2843,#0d1c31);padding:16px;position:relative;overflow:hidden}.metric small{display:block;color:#9bb8d1;font-size:12px}.metric strong{display:block;font-size:26px;margin:8px 0 2px}.metric .more{font-size:11px;color:#8daac7}.metric::after{content:'';position:absolute;width:90px;height:90px;border-radius:100px;right:-40px;top:-50px;background:#008bd815;pointer-events:none}.metric.server{background:linear-gradient(150deg,#064636,#09282b);border-color:#009579}.bar{height:6px;background:#23405d;border-radius:6px;margin-top:13px;overflow:hidden}.bar i{display:block;background:linear-gradient(90deg,#1b9fe4,#39e8bd);height:100%;width:0;transition:width .4s}.clickable{cursor:pointer;outline:none}.clickable:hover{border-color:#19b6e5}.drive-controls{position:absolute;right:8px;top:9px;display:flex;gap:3px}.drive-controls button{font-size:14px;padding:3px 8px;border-radius:8px}.columns{display:grid;grid-template-columns:1.15fr 1fr;gap:14px}.panel{border:1px solid #25405f;border-radius:15px;background:linear-gradient(155deg,#0e2139,#0b1a2c);padding:17px}.panel h2{font-size:17px;margin:0 0 5px}.panelheader{display:flex;align-items:center;gap:9px;margin-bottom:12px}.section-icon{font-size:23px;color:var(--mint)}.choices{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:8px}.choice{padding:16px 9px;min-height:135px;text-align:center;border-radius:12px;background:#0a1b2e;border:1px solid #254465;display:flex;align-items:center;flex-direction:column;gap:8px;justify-content:center}.choice.selected{background:#08352f;border-color:#1cecb0;box-shadow:inset 0 0 0 1px #0da87d}.choice .symbol{font-size:24px}.choice strong{font-size:13px}.choice small{font-size:11px;color:#a0bdd0;line-height:1.5}.scope{background:#08362f99;border:1px solid #147e6a;border-radius:13px;padding:20px;min-height:136px;display:flex;gap:15px;align-items:center}.scope-icon{font-size:35px}.scope strong{display:block;margin-bottom:8px}.scope p{color:#a0c2bd;margin:0;font-size:12px;line-height:1.55}.scopeextra{margin-top:10px;display:flex;gap:8px;flex-wrap:wrap}.scopeextra button{font-size:12px;padding:8px}.lab-preview{margin-top:14px}.flex-between{display:flex;justify-content:space-between;gap:15px;align-items:center}.canvas{height:310px;position:relative;overflow:hidden;border:1px solid #1a4160;border-radius:14px;background-color:#071b2b;background-image:radial-gradient(#1b4770 1px,transparent 1px);background-size:19px 19px}.canvas.full{height:calc(100vh - 205px);min-height:470px}.canvas .world{position:absolute;left:0;top:0;width:3000px;height:2000px;transform-origin:0 0;cursor:grab}.canvas .world.grabbing{cursor:grabbing}.agent{width:305px;min-height:172px;position:absolute;border:1px solid #397aca;border-radius:14px;background:linear-gradient(160deg,#102a47,#071627);box-shadow:0 10px 27px #0009;overflow:hidden;user-select:none}.agent.waiting{border-color:#d6a046}.agent.done{border-color:#16a77b}.agent.error{border-color:#e16a78}.agenttitle{padding:12px 12px 8px;display:flex;gap:9px;align-items:center}.agenttitle b{font-size:14px}.agenttitle small{display:block;color:#9cbad5;font-size:11px;margin-top:3px}.agentbody{padding:8px 12px 12px}.terminal{font:12px/1.55 Consolas,'Courier New',monospace;background:#041020;border-radius:8px;padding:10px;height:93px;overflow:auto;color:#add0e9;white-space:pre-wrap;overflow-wrap:anywhere;user-select:text}.agent .stamp{position:absolute;right:10px;top:14px;color:#9ebbd5;font-size:11px}.agent .agenticon{font-size:24px}.empty-canvas{position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);background:#0b2135ee;border:1px dashed #33627e;padding:28px;border-radius:16px;text-align:center;width:min(420px,83%);pointer-events:none}.empty-canvas strong{display:block;margin:10px 0}.empty-canvas p{font-size:13px;color:#96b3cd;line-height:1.6}.labtoolbar{display:flex;justify-content:space-between;gap:10px;align-items:center;flex-wrap:wrap;margin:18px 0}.muted{color:#8baeca;font-size:12px}.back{display:none}.toast{position:fixed;bottom:22px;right:22px;max-width:400px;padding:14px 18px;background:#0b513e;border:1px solid #14b88b;color:white;border-radius:11px;box-shadow:0 10px 35px #000a;display:none;z-index:20}.toast.bad{background:#572a37;border-color:#b9556d}.modalback{position:fixed;inset:0;background:#010710cc;display:none;place-items:center;z-index:50;padding:20px}.modal{background:#0d2137;border:1px solid #315774;border-radius:16px;padding:23px;max-width:590px;width:100%;max-height:90vh;overflow:auto;box-shadow:0 30px 90px #000b}.modal h2{margin-top:0}.modal p{line-height:1.55;color:#aec3d9}.modal input,.modal select{padding:10px;background:#071729;color:#ecf7ff;border:1px solid #456481;border-radius:8px;width:100%;margin:7px 0 13px}.modal .group{display:flex;gap:9px;justify-content:flex-end;flex-wrap:wrap}.pendingitem{border-top:1px solid #2b4966;padding:12px 0;display:flex;justify-content:space-between;align-items:center;gap:12px}.pendingitem small{display:block;color:#9db5cb;margin-top:5px}.pendingitem button{padding:8px;margin-left:5px}.rootline{padding:10px 0;border-bottom:1px solid #31465a;overflow-wrap:anywhere;font-size:12px;display:flex;justify-content:space-between;align-items:center;gap:8px}.rootline small{display:block;color:#94b0c8}.hidden{display:none!important}
@media(max-width:1280px){.metrics{grid-template-columns:repeat(3,minmax(0,1fr))}.columns{grid-template-columns:1fr}.main{padding:24px}}@media(max-width:760px){.app{grid-template-columns:1fr}.side{height:auto;position:relative;padding:12px 17px;gap:6px}.logo img{height:38px;width:38px}.side small,.sidefooter{display:none}.sideupdate{margin-top:6px}.sidebottom{display:block}.nav{width:auto;display:inline-flex}.main{padding:18px}.top{flex-direction:column}.top h1{font-size:24px}.metrics{grid-template-columns:repeat(2,minmax(0,1fr))}.choices{grid-template-columns:1fr}.choice{min-height:75px}.canvas.full{min-height:450px}.metric strong{font-size:23px}}


.context-header{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:12px;margin:16px 0}
.context-layout{display:grid;grid-template-columns:minmax(0,1fr) 285px;gap:14px}
.context-canvas{height:610px;min-width:0;border:1px solid #264760;border-radius:17px;background:radial-gradient(ellipse at 50% 40%,#112942,#081524 65%);overflow:hidden;position:relative}
.context-canvas svg{display:block;width:100%;height:100%;touch-action:none}
.context-node{cursor:pointer;filter:drop-shadow(0 4px 10px #0008)}.context-node circle{stroke:#c1e9ff55;stroke-width:1.3;transition:stroke .15s}
.context-node:hover circle{stroke:#fff;stroke-width:2.5}.context-node text{fill:#dceeff;font:13px 'Segoe UI',Arial,sans-serif;pointer-events:none;text-anchor:middle}
.context-edge{stroke:#547b9c;stroke-width:1.4;opacity:.55}
.context-detail{min-height:610px;background:#0c1e32;border:1px solid #284963;border-radius:16px;padding:18px;overflow-wrap:anywhere}
.context-detail h3{font-size:17px;margin:0 0 8px}
.context-info{font-size:12px;color:#9cbbd3;line-height:1.7;white-space:pre-wrap;max-height:360px;overflow:auto}
.context-control{display:flex;gap:8px;align-items:center;flex-wrap:wrap}.context-control select,.context-control input{background:#0a1a2b;color:#eaf6ff;border:1px solid #3b607e;border-radius:9px;padding:10px;min-width:180px;max-width:100%}
.context-control button{padding:9px 12px}
.context-empty{position:absolute;left:50%;top:48%;transform:translate(-50%,-50%);text-align:center;color:#86a8c8;width:min(360px,85%);pointer-events:none}
.context-meta{color:#8fb0c8;font-size:12px;line-height:1.6}
.context-legend{display:flex;gap:16px;flex-wrap:wrap;font-size:12px;color:#9bbad1}.context-legend i{display:inline-block;width:10px;height:10px;border-radius:50%;margin-right:6px}
@media(max-width:960px){.context-layout{grid-template-columns:1fr}.context-detail{min-height:230px}.context-canvas{height:480px}}
 
.integrations-title{margin:18px 0 10px;font-weight:700;font-size:16px}
.integration-cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(225px,1fr));gap:10px;margin:0 0 18px}
.integration-card{border:1px solid #284d5c;background:#0d2532;border-radius:14px;padding:14px}
.integration-card h3{font-size:14px;margin:0 0 7px}.integration-card p{font-size:12px;line-height:1.4;color:#a6c2d5;min-height:33px}
.integration-card button{font-size:12px;padding:8px 12px}
.integration-status{font-size:11px;color:#70d8af}

/* Contexto: workspace three-pane inspired by graph knowledge apps */
#contextPage{--ctxbg:#10151b;--ctxpanel:#161d26;--ctxstroke:#2c3945;--ctxmuted:#9caebd}
#contextPage .top{margin-bottom:16px}
#contextPage .context-hero{display:flex;align-items:center;justify-content:space-between;gap:18px;background:linear-gradient(115deg,#123a32,#192335 55%,#201f3a);border:1px solid #377964;border-radius:18px;padding:15px 20px;margin-bottom:14px;flex-wrap:wrap}
#contextPage .context-hero strong{font-size:15px;display:block}
#contextPage .context-hero p{font-size:12px;color:#b0c7ce;margin:5px 0 0}
#contextPage .context-pill{color:#b9eeda;border:1px solid #50856e;background:#0b2c27;border-radius:999px;padding:7px 11px;font-size:12px}
#contextPage .context-header{background:var(--ctxpanel);border:1px solid var(--ctxstroke);border-radius:13px;padding:10px 13px;margin-bottom:12px}
#contextPage .context-workspace{display:flex;align-items:center;gap:9px;flex-wrap:wrap}
#contextPage .context-layout{grid-template-columns:240px minmax(0,1fr) 295px;gap:10px;align-items:stretch}
#contextPage .context-explorer{min-width:0;background:var(--ctxpanel);border:1px solid var(--ctxstroke);border-radius:15px;min-height:610px;padding:13px;overflow:auto}
#contextPage .context-explorer h3{font-size:12px;text-transform:uppercase;letter-spacing:.12em;color:#8399a9;margin:0 0 11px}
#contextPage .context-search{width:100%;min-width:0;background:#0e1520;border:1px solid #324456;border-radius:10px;padding:10px;color:#e6eff4;margin-bottom:14px}
#contextPage .context-folder{width:100%;display:flex;gap:8px;align-items:center;background:transparent;border:0;text-align:left;padding:9px 10px;font-weight:500;font-size:12px;border-radius:9px;color:#d5e5e9}
#contextPage .context-folder:hover,#contextPage .context-note-row:hover{background:#283747}
#contextPage .context-folder .folder-glyph{color:#b6a6ec}
#contextPage .context-folder .tree-count{margin-left:auto;color:#8b9bab;font-size:10px}
#contextPage .context-folder.active{background:#173d35;border:1px solid #34765f}
#contextPage .context-note-row{border:0;width:100%;display:block;text-align:left;background:none;color:#a6bccb;font-size:12px;padding:8px 8px 8px 28px;border-radius:8px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
#contextPage .context-explorer-footer{font-size:11px;line-height:1.6;color:#76939a;border-top:1px solid #324252;padding-top:12px;margin-top:17px}
#contextPage .context-canvas{background:radial-gradient(circle at 51% 48%,#182e32 0,#111d2b 47%,#10161f 85%);border-color:var(--ctxstroke)}
#contextPage .context-canvas:after{content:'';position:absolute;inset:0;pointer-events:none;opacity:.17;background-image:radial-gradient(#80b49d .85px,transparent .85px);background-size:24px 24px}
#contextPage .context-canvas svg{position:relative;z-index:1}
#contextPage .context-node:hover{filter:drop-shadow(0 0 12px #56e6b377)}
#contextPage .context-detail{background:var(--ctxpanel);border-color:var(--ctxstroke)}
#contextPage .context-detail h3{color:#e5f4f6}
#contextPage .context-info{background:#0f1922;border:1px solid #2c3b49;border-radius:10px;padding:12px;min-height:175px}
#contextPage .context-meta{line-height:1.6}
#contextPage .context-toolbar{position:absolute;left:12px;top:12px;z-index:2;background:#13232ccf;border:1px solid #314c55;border-radius:10px;padding:8px 11px;font-size:11px;color:#acd8ce}
#contextPage .integrations-title{display:flex;align-items:center;gap:9px}
#contextPage .integration-card{background:linear-gradient(145deg,#172532,#12202a);border-color:#294856;transition:transform .18s,border-color .18s}
#contextPage .integration-card:hover{transform:translateY(-2px);border-color:#3a9477}
#contextPage .integration-card h3{font-size:15px}
#contextPage .integration-card p{min-height:35px}
#contextPage .integration-status{display:inline-block;border-radius:999px;background:#163f33;padding:4px 8px;color:#85e8bf}
#contextPage .integration-hint{display:block;overflow-wrap:anywhere;color:#8ea8b4;font-size:10px;margin-top:8px}
#contextPage .context-vault-advanced{font-size:11px;color:#96b4c7}
@media(max-width:1220px){#contextPage .context-layout{grid-template-columns:210px minmax(0,1fr)}#contextPage .context-detail{grid-column:1/-1;min-height:180px}}
@media(max-width:760px){#contextPage .context-layout{grid-template-columns:1fr}#contextPage .context-explorer{min-height:180px;max-height:380px}#contextPage .context-canvas{height:420px}#contextPage .context-detail{grid-column:auto}}

/* v4.10 Obsidian-inspired graph: only connection cards + uninterrupted canvas. */
#contextPage{display:flex;flex-direction:column;gap:12px;height:calc(100vh - 96px);min-height:660px}
#contextPage.hidden{display:none!important}
#contextPage .context-card-strip{display:flex;flex-wrap:nowrap;overflow-x:auto;scrollbar-width:thin;gap:10px;margin:0;padding:2px 0 8px;flex:0 0 auto;min-height:150px}
#contextPage .integration-card{flex:0 0 223px;min-height:145px;padding:14px;border-radius:14px;background:linear-gradient(145deg,#182430,#0d1a25);border:1px solid #304452;box-shadow:0 4px 20px #00000024}
#contextPage .integration-card h3{font-size:14px;letter-spacing:-.1px}
#contextPage .integration-card p{font-size:11px;min-height:0;line-height:1.45;margin:8px 0;color:#9db4c1}
#contextPage .integration-status{font-size:10px;padding:5px 9px}
#contextPage .context-graph-infinite{flex:1;position:relative;min-height:460px;width:100%;border-radius:16px;overflow:hidden;border:1px solid #26374d;background-color:#101820;background-image:radial-gradient(#2e4250 1px,transparent 1px);background-size:25px 25px;touch-action:none}
#contextPage .context-graph-infinite svg{display:block;width:100%;height:100%;cursor:grab;touch-action:none}
#contextPage .context-graph-infinite svg.dragging{cursor:grabbing}
#contextPage .context-node{cursor:grab;outline:none}
#contextPage .context-node.dragging{cursor:grabbing}
#contextPage .context-node circle{stroke:#daf8ff50;stroke-width:1.5;transition:filter .14s,stroke .14s}
#contextPage .context-node:hover circle,#contextPage .context-node:focus circle{stroke:#f3ffff;stroke-width:2.5;filter:drop-shadow(0 0 10px #7bece17f)}
#contextPage .context-node text{font:12px 'Segoe UI',Arial,sans-serif;fill:#d7e7f0;stroke:#0e1c26;stroke-width:3.5px;paint-order:stroke;pointer-events:none;text-anchor:middle}
#contextPage .context-node.folder text{font-weight:650}
#contextPage .context-node.note text{font-size:11px}
#contextPage .context-edge{stroke:#607d9b;stroke-width:1.2;opacity:.4;pointer-events:none}
#contextPage .graph-actions{position:absolute;top:14px;left:14px;z-index:3;display:flex;align-items:center;gap:5px;padding:6px;border:1px solid #3c5365;border-radius:12px;background:#111e2ae8;backdrop-filter:blur(10px)}
#contextPage .graph-actions button{border:0;min-height:34px;padding:5px 9px;background:transparent;font-size:18px;color:#b9d1df}
#contextPage .graph-actions button:hover{background:#2b3c4d}
#contextPage .graph-actions input{width:210px;min-width:0;padding:8px;background:#0d1722;border:1px solid #35485a;color:#e8f3f7;border-radius:8px;font-size:12px}
#contextPage .context-graph-caption{position:absolute;bottom:13px;left:15px;z-index:2;color:#88a2b1;font-size:11px;pointer-events:none;background:#0e1b25c7;border-radius:8px;padding:7px 10px}
#contextPage .context-graph-preview{position:absolute;z-index:4;right:16px;top:70px;width:min(340px,calc(100% - 32px));max-height:65%;overflow:auto;padding:18px;background:#142431f5;border:1px solid #38576a;border-radius:14px;box-shadow:0 16px 55px #0009}
#contextPage .context-preview-title{font-size:16px;font-weight:700;padding-right:28px;margin-bottom:12px}
#contextPage .context-preview-body{font-size:12px;line-height:1.65;color:#bad0de;white-space:pre-wrap;overflow-wrap:anywhere;max-height:300px;overflow:auto}
#contextPage .context-preview-close{position:absolute;right:9px;top:8px;font-size:20px;border:0;background:transparent;padding:2px 8px}
#contextPage #contextOpenObsidian{margin-top:12px;font-size:12px}
#contextPage .context-graph-settings{position:absolute;z-index:5;left:14px;top:70px;background:#152432f4;border:1px solid #3e5b69;border-radius:12px;padding:16px;width:min(285px,calc(100% - 28px));box-shadow:0 15px 35px #0007}
#contextPage .context-graph-settings label{display:block;font-size:12px;color:#c0d6e2;margin-top:14px}
#contextPage .context-graph-settings input{display:block;width:100%;accent-color:#51c6ae}
#contextPage .context-graph-settings p{color:#92a9b6;font-size:11px;line-height:1.5}
#contextPage .context-graph-menu{position:absolute;z-index:6;min-width:180px;background:#152b36;border:1px solid #456171;border-radius:10px;padding:5px;box-shadow:0 18px 35px #000a}
#contextPage .context-graph-menu button{display:block;text-align:left;width:100%;border:0;background:transparent;border-radius:7px;padding:9px;font-size:12px}
#contextPage .context-graph-menu button:hover{background:#304351}
#contextPage .context-graph-empty{position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);max-width:380px;text-align:center;pointer-events:none;font-size:13px;color:#b0c6cf}
@media(max-width:900px){#contextPage{height:calc(100dvh - 120px);min-height:560px}#contextPage .integration-card{flex-basis:205px}#contextPage .graph-actions input{width:140px}}
@media(max-width:500px){#contextPage .graph-actions input{width:110px}#contextPage .graph-actions button{font-size:14px;padding:5px}#contextPage .context-graph-caption span:last-child{display:none}}


/* 4.11 — compact autonomy control and exclusive browser switcher */
.approval-compact{max-width:590px;padding:11px 6px 5px}
.approval-levels{display:flex;justify-content:space-between;gap:5px;margin-bottom:8px}
.approval-levels button{flex:1;min-width:0;text-align:center;border:1px solid #28435a;border-radius:9px;background:#101f31;font-size:11px;font-weight:600;line-height:1.3;padding:8px 4px;color:#91adc4}
.approval-levels button.selected{border-color:#3fc5a3;color:#dcfff4;background:#174537}
#modeSlider{display:block;width:100%;margin:8px 0;accent-color:#2dd6aa;cursor:pointer;min-height:27px}
.approval-status{display:flex;justify-content:space-between;align-items:center;gap:14px;font-size:12px;color:#a2c3cf}
.approval-status span{font-weight:700;color:#daf7ec}.approval-status small{font-size:11px;text-align:right;color:#94aebc}
.exclusive-browser-choice{display:flex;gap:9px;flex-wrap:wrap;padding:10px 0 12px}
.exclusive-browser-choice button{flex:1;min-width:160px;padding:13px 15px}
.exclusive-browser-choice button.selected{border-color:#37d6a9;background:#104d3d;box-shadow:inset 0 0 0 1px #2ac391}

/* Obsidian-like graph with index summary and subdued file node labels */
#contextPage .context-index-status{position:absolute;top:14px;right:15px;z-index:4;padding:10px 14px;min-width:180px;max-width:260px;background:#101923e8;border:1px solid #354557;border-radius:13px;display:flex;flex-direction:column;gap:3px;backdrop-filter:blur(10px);pointer-events:none}
#contextIndexCount{font-size:13px;color:#ccece9;letter-spacing:-.1px}
#contextIndexSubtitle{font-size:10px;color:#90a9bb}
#contextPage .context-graph-infinite{background-color:#0e1117;background-image:radial-gradient(#293440 0.65px,transparent .65px);background-size:27px 27px;border-color:#2a3341}
#contextPage .context-edge{stroke:#64758b;stroke-width:.9;opacity:.22}
#contextPage .context-node.indexed-file circle{stroke:#9bb8c0aa;stroke-width:.4}
#contextPage .context-node.indexed-file text{opacity:0;font-size:10px;transition:opacity .2s}
#contextPage .context-node.indexed-file:hover text,#contextPage .context-node.indexed-file:focus text{opacity:1}
#contextPage .context-node.indexed-folder circle{stroke:#d1bfff8a;stroke-width:1.4;filter:drop-shadow(0 0 8px #836dd533)}
#contextPage .context-node.indexed-folder text{font-size:12px;fill:#d6cdf5;font-weight:650}
#contextPage .context-node.vault circle{filter:drop-shadow(0 0 12px #927adb77)}
#contextPage .context-node.app circle{filter:drop-shadow(0 0 7px #81dccb50)}
#contextPage .context-graph-preview{background:#151b25f7;border-color:#4b5167}
#contextPage .context-graph-settings{background:#171d28f6;border-color:#394659}
@media(max-width:850px){#contextPage .context-index-status{right:10px;top:67px;padding:8px 11px;min-width:140px}#contextIndexCount{font-size:11px}}


/* v4.13: local-to-ChatGPT composer, integrated in the existing dashboard */
.live-compose{margin:15px 0 21px;padding:19px;border:1px solid #246960;border-radius:17px;background:linear-gradient(110deg,#112f35,#12243b 60%,#162744);box-shadow:0 8px 30px #0002}
.live-compose h2{font-size:19px;font-weight:700;margin:0 0 7px}
.live-compose .live-sub{font-size:12px;color:#b0ccdc;margin:0 0 13px;line-height:1.6}
.live-compose textarea{width:100%;min-height:105px;resize:vertical;border:1px solid #365773;border-radius:12px;background:#0c1c2b;color:#f5fcff;font:14px/1.6 'Segoe UI',sans-serif;padding:13px;outline:0}
.live-compose textarea:focus{border-color:#52d8bc;box-shadow:0 0 0 2px #42cdaa2b}
.live-compose .live-actions{display:flex;gap:12px;align-items:center;flex-wrap:wrap;margin:11px 0}
.live-compose .live-status{font-size:12px;color:#b4cddd;white-space:pre-wrap}
.live-compose .live-history{display:grid;gap:6px;margin-top:12px;max-height:220px;overflow:auto}
.live-compose .live-item{display:flex;align-items:center;justify-content:space-between;gap:10px;border:1px solid #294c59;border-radius:10px;padding:10px;font-size:12px;background:#0b202e}
.live-compose .live-item span{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.live-compose .live-item b{font-size:11px;color:#8ddbbf;white-space:nowrap}
.live-compose .live-item button{font-size:11px;padding:6px 9px}


/* First-use activation. Collapsed by default to keep the main chat in view. */
.consecom-onboarding{border:1px solid #285f73;border-radius:17px;background:linear-gradient(115deg,#122d3c,#14283f);padding:15px 18px;margin-bottom:14px}
.consecom-onboarding-head{display:flex;align-items:center;justify-content:space-between;gap:14px;flex-wrap:wrap}
.consecom-onboarding strong{font-size:14px;color:#e8f9ff}
.consecom-onboarding p{font-size:12px;color:#adcbd9;line-height:1.6;margin:5px 0}
.consecom-onboarding-actions{display:flex;gap:9px;align-items:center;flex-wrap:wrap}
.consecom-onboarding details{margin-top:8px;font-size:12px;color:#b9dce5}
.consecom-onboarding summary{cursor:pointer;display:list-item;width:max-content;max-width:100%}
.consecom-onboarding textarea{display:block;width:100%;margin-top:10px;resize:vertical;min-height:135px;max-height:290px;border-radius:11px;border:1px solid #31546c;background:#0a1c2b;color:#e5f5ff;padding:12px;font:12px/1.6 'Segoe UI',sans-serif}
.consecom-onboarding .onboarding-note{color:#84aabe;font-size:11px}
@media(max-width:680px){.consecom-onboarding{padding:13px}.consecom-onboarding-actions button{width:100%}}

/* v4.14 - actual two-way transcript with separate message roles */
.live-compose{margin:14px 0 22px;padding:0!important;border:1px solid #2b485c!important;border-radius:20px!important;background:#0b1827!important;box-shadow:0 12px 36px #0004!important;overflow:hidden}
.live-chat-head{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:17px 20px;border-bottom:1px solid #243d51;background:linear-gradient(115deg,#103a43,#15263b)}
.live-chat-identity{display:flex;align-items:center;gap:12px;min-width:0}
.live-chat-avatar{display:grid;place-items:center;flex:none;width:39px;height:39px;background:#1c746a;border:1px solid #49b39b;border-radius:14px;color:#e5fff4;font-size:20px}
.live-chat-head h2{font-size:17px;font-weight:700;margin:0}
.live-chat-head p{margin:3px 0 0;font-size:11px;color:#b1c9d5}
.live-chat-head button{padding:7px 11px;font-size:12px}
.live-chat-stream{display:flex;flex-direction:column;gap:14px;overflow-y:auto;min-height:255px;max-height:445px;padding:22px 20px;background:radial-gradient(circle at 90% 0%,#0c3041 0,transparent 52%),#0a1725}
.live-chat-empty{margin:auto;text-align:center;max-width:320px;font-size:13px;line-height:1.7;color:#95b3c4}
.live-chat-row{display:flex;flex-direction:column;max-width:min(85%,680px);min-width:0;align-self:flex-start}
.live-chat-row.mine{align-self:flex-end}
.live-chat-bubble{white-space:pre-wrap;overflow-wrap:anywhere;padding:12px 15px;font-size:13px;line-height:1.65;background:#1d3349;color:#f0faff;border:1px solid #345a74;border-radius:15px 15px 15px 5px}
.live-chat-row.mine .live-chat-bubble{background:linear-gradient(135deg,#116d5f,#15584e);border-color:#278e77;border-radius:15px 15px 5px 15px}
.live-chat-row.progress .live-chat-bubble{background:#173042;border-style:dashed;border-color:#68829d;color:#d9e5ed}
.live-chat-typing{display:flex;align-items:center;gap:9px;min-height:32px;font-size:12px;color:#c6d7e5}
.live-chat-typing-dots{display:inline-flex;gap:4px;align-items:center;height:12px}
.live-chat-typing-dots i{width:6px;height:6px;border-radius:50%;background:#66d9b2;animation:consecomTyping 1.25s ease-in-out infinite;opacity:.4}
.live-chat-typing-dots i:nth-child(2){animation-delay:.17s}
.live-chat-typing-dots i:nth-child(3){animation-delay:.34s}
@keyframes consecomTyping{0%,65%,100%{opacity:.35;transform:translateY(0)}30%{opacity:1;transform:translateY(-3px)}}
@media(prefers-reduced-motion:reduce){.live-chat-typing-dots i{animation:none;opacity:.8}}
.live-chat-meta{color:#779cae;font-size:10px;margin:5px 5px 0;display:flex;justify-content:space-between;gap:10px}
.live-chat-row.mine .live-chat-meta{justify-content:flex-end}
.live-chat-bottom{padding:15px 17px 12px;border-top:1px solid #234055}
.live-chat-editor{display:flex;align-items:end;gap:10px}
.live-chat-editor textarea{flex:1;min-height:44px;max-height:132px;resize:vertical;padding:11px 12px;background:#122539;border-radius:13px;border:1px solid #2d536c;color:#f0faff;font:13px/1.5 'Segoe UI',sans-serif}
.live-chat-editor textarea:focus{border-color:#31c89e}
.live-chat-editor button{flex:none;min-height:44px}
.live-chat-info{display:flex;justify-content:space-between;gap:14px;flex-wrap:wrap;margin-top:9px;color:#8aa9ba;font-size:11px}
.live-chat-status{white-space:pre-wrap;max-width:100%}
@media(max-width:680px){.live-chat-stream{padding:13px;max-height:400px}.live-chat-row{max-width:92%}.live-chat-editor{align-items:stretch}.live-chat-editor textarea{width:100%}}
</style></head><body><div class="app"><aside class="side"><div class="logo"><img src="/logo.png" alt="Símbolo Consecom"><div>Consecom <span class="light">Plugin</span></div></div><small>Seu computador conectado ao ChatGPT</small><button class="nav active" id="navPanel" onclick="page('panel')"><span class="symbol">⌂</span> Painel</button><button class="nav" id="navLab" onclick="page('lab')"><span class="symbol">◈</span> Laboratório de Agentes</button><button class="nav" id="navContext" onclick="page('context')"><span class="symbol">◎</span> Contexto</button><div class="sideupdate"><button class="primary update-button" id="sideUpdateBtn" onclick="updateNow()">↻ Atualizar</button><div class="sideupdate-detail" id="updateSummary">Fonte oficial • GitHub</div></div><div class="sidebottom"><div><span class="dot"></span>Servidor MCP <b class="light">Em execução</b></div><div style="margin-top:9px" id="sideConnection">Aguardando conexão do ChatGPT</div></div><div class="sidefooter">Versão 4.14.6 • Servidor local<br>127.0.0.1:8765</div></aside><main class="main">
<section id="panelPage"><header class="top"><div><h1>Central de Controle do Computador</h1><div class="sub">Controle local do Windows conectado ao ChatGPT, com permissões administradas por você.</div></div><div class="top-actions"><div class="chip"><span class="dot"></span><span id="chatStatus">Aguardando ChatGPT</span></div><button class="primary" onclick="openModal('connection')">↗ Conectar ao ChatGPT</button></div></header><section class="consecom-onboarding" id="consecomOnboarding" aria-label="Ativar o Consecom nesta conversa">
<div class="consecom-onboarding-head"><div>
<strong>Ative o Consecom em uma conversa do ChatGPT</strong>
<p>1. Instale o Consecom no Windows. 2. Conecte o plugin MCP ao ChatGPT. 3. Copie o prompt e envie em uma conversa.</p>
</div><div class="consecom-onboarding-actions"><button type="button" class="primary" onclick="copyConsecomSetupPrompt()">⧉ Copiar prompt de ativação</button></div></div>
<details><summary>Ver o prompt que será copiado</summary>
<textarea id="consecomSetupPrompt" readonly spellcheck="false" aria-label="Prompt de ativação do Consecom Live">Ative esta conversa como minha sessão principal do Consecom Plugin.

O programa Consecom deve estar instalado no Windows e o plugin MCP deve ter sido conectado manualmente a esta conta ChatGPT.

1. Verifique a conexão real chamando atualizacao_status e consecom_chat_historico do Consecom. Se as ferramentas não aparecerem, peça para eu conectar/reativar o plugin e atualizar a página; não finja que está conectado.
2. Chame consecom_live_abrir com modo="mensagens" e apresente o componente Consecom Live nesta conversa.
3. Peça para eu manter marcado "Permitir envio automático das mensagens digitadas no painel" e o componente aberto enquanto quiser receber mensagens.
4. Para cada mensagem recebida do painel Windows, responda normalmente e publique a resposta correspondente usando consecom_chat_responder com o ID exato da mensagem, texto completo e tipo="final". Em tarefas longas, use tipo="andamento" para comunicar o progresso e tipo="final" para concluir.
5. Respeite as permissões e aprovações do Windows; nunca execute uma operação que exija autorização local sem a autorização necessária.
6. Não use ChatGPT Work, API paga da OpenAI, segundo MCP ou um serviço de IA adicional.

Ao concluir, confirme somente o que conseguiu verificar. O componente depende desta conversa estar ativa e pode ser suspenso pelo ChatGPT; não prometa conexão contínua em segundo plano.</textarea></details>
<p class="onboarding-note" id="consecomSetupCopyStatus" role="status">O prompt abre o Live nesta conversa; não instala o programa nem garante funcionamento com o ChatGPT fechado.</p>
</section>
<div class="live-compose" id="liveCompose" aria-label="Chat com ChatGPT">
<div class="live-chat-head"><div class="live-chat-identity"><span class="live-chat-avatar">✦</span><div>
<h2>Chat com ChatGPT</h2><p>Conversa bidirecional via Consecom MCP · uma conexão</p><p id="liveBridgeConnection" aria-live="polite" style="margin:4px 0 0;font-size:11px;color:#c9a46f">Conferindo se o Consecom Live está ativo...</p></div></div>
<div style="display:flex;gap:6px;flex-wrap:wrap"><button onclick="showConsecomLatency()" title="Diagnosticar os tempos por etapa, sem ler conteúdos">◷ Latência</button><button onclick="refreshConsecomChat(true)" title="Buscar novas mensagens">↻ Atualizar</button></div></div>
<div class="live-chat-stream" id="liveChatStream" role="log" aria-live="polite" aria-relevant="additions">
<div class="live-chat-empty">Esta é sua conversa com o ChatGPT. Escreva abaixo, mantenha o Consecom Live ativo na conversa do ChatGPT e veja a resposta chegar aqui.</div>
</div>
<div class="live-chat-bottom"><label for="liveUserMessage" style="display:block;font-size:11px;color:#9bbacc;margin-bottom:6px">Mensagem para o ChatGPT</label>
<div class="live-chat-editor"><textarea id="liveUserMessage" maxlength="2000" rows="2" placeholder="Pergunte algo ou peça uma implementação..."></textarea>
<button class="primary" id="liveSendButton" onclick="sendConsecomMessage()">➤ Enviar</button></div>
<div class="live-chat-info"><span id="liveComposeState" class="live-chat-status">Pronto para conversar.</span>
<span>Enter para enviar · Shift+Enter para nova linha</span></div>
</div></div>

<div class="metrics"><div class="metric"><small>RAM em uso</small><strong id="ramUsed">—</strong><div class="more" id="ramTotal">Memória física</div><div class="bar"><i id="ramUsedBar"></i></div></div><div class="metric"><small>RAM disponível</small><strong id="ramFree">—</strong><div class="more">Disponível no Windows</div><div class="bar"><i id="ramFreeBar"></i></div></div><div class="metric"><small>CPU</small><strong id="cpu">—</strong><div class="more">Uso total medido</div><div class="bar"><i id="cpuBar"></i></div></div><div class="metric clickable" id="diskTile" role="button" tabindex="0" onclick="nextDisk()" onkeydown="if(event.key==='Enter'||event.key===' '){event.preventDefault();nextDisk()}"><div class="drive-controls"><button onclick="event.stopPropagation();nextDisk(-1)" title="Disco anterior">‹</button><button onclick="event.stopPropagation();nextDisk(1)" title="Próximo disco">›</button></div><small id="diskLabel">Disco (—)</small><strong id="diskUsed">—</strong><div class="more" id="diskTotal">Clique para alternar</div><div class="bar"><i id="diskBar"></i></div></div><div class="metric server"><small>Servidor MCP</small><strong style="font-size:19px" id="serverStatus"><span class="dot"></span> Em execução</strong><div class="more" id="uptime">—</div></div></div>
<div class="columns"><section class="panel"><div class="panelheader"><span class="section-icon">⬡</span><div><h2>Modo de aprovação e permissões</h2><div class="muted">Decida quando autorizar ações solicitadas pelo ChatGPT.</div></div></div><div class="approval-compact"><div class="approval-levels"><button id="modeAlways" type="button" onclick="setMode('always')">Perguntar sempre</button><button id="modeCategory" type="button" onclick="setMode('category')">Por categoria</button><button id="modeMaximum" type="button" onclick="setMode('maximum')">Aprovação máxima</button></div><input id="modeSlider" type="range" min="0" max="2" step="1" value="0" aria-label="Nível de autonomia do Consecom" oninput="previewModeSlider(this.value)" onchange="commitModeSlider(this.value)"><div class="approval-status"><span id="approvalCurrentName">Perguntar sempre</span><small id="approvalHint">Solicita confirmação para ações no computador</small></div></div><div id="pendingWrap" class="hidden" style="margin-top:16px;padding:12px;background:#122c40;border:1px solid #3d5d73;border-radius:11px"><b id="pendingTitle">Aprovações pendentes</b><div id="pendingList"></div></div></section><section class="panel"><div class="panelheader"><span class="section-icon">▣</span><div><h2>Escopo de acesso</h2><div class="muted">Arquivos e projetos disponíveis pelas permissões escolhidas.</div></div></div><div class="scope"><span class="scope-icon">📁</span><div><strong id="scopeTitle">Pastas aprovadas</strong><p id="scopeDesc">Carregando permissões...</p></div></div><div class="scopeextra" id="scopeActions"><button onclick="openModal('folders')">Gerenciar pastas aprovadas</button></div></section></div>
<section class="panel lab-preview" style="margin-bottom:14px"><div class="flex-between"><div><div class="panelheader"><span class="section-icon">◉</span><div><h2>Navegador exclusivo do Consecom</h2><div class="muted" id="browserSummary">Sessão exclusiva • Chrome / Edge</div></div></div><div class="sub">Escolha apenas Chrome ou Microsoft Edge. A sessão exclusiva do Consecom é persistente e não acessa cookies, abas ou logins do seu navegador pessoal.</div></div><button onclick="openModal('browser')">⚙ Escolher navegador</button></div></section>

<section class="panel lab-preview"><div class="flex-between"><div class="panelheader"><span class="section-icon">♧</span><div><h2>Laboratório de Agentes</h2><div class="muted">Visualize atividades reais reportadas por agentes, em um canvas interativo.</div></div></div><button onclick="page('lab')">⛶ Visualizar</button></div><div class="canvas" id="previewCanvas"><div class="world" id="previewWorld"></div><div class="empty-canvas" id="emptyPreview"><span style="font-size:29px">⌘</span><strong>Aguardando agentes</strong><p>Os agentes aparecerão aqui quando registrarem suas atividades pelo MCP. Nenhum terminal simulado.</p></div></div></section></section>
<section id="labPage" class="hidden"><header class="top"><div><h1>Laboratório de Agentes</h1><div class="sub">Mapa espacial de agentes reais conectados ao Consecom Plugin, com logs de atividade.</div></div><div class="top-actions"><button onclick="resetView()">◎ Centralizar</button><button onclick="page('panel')">← Voltar ao painel</button></div></header><div class="labtoolbar"><div class="muted" id="agentCount">0 agentes conectados</div><div class="top-actions"><button onclick="zoomBy(-.2)">− Zoom</button><button onclick="zoomBy(.2)">+ Zoom</button><button onclick="refresh()">↻ Atualizar</button></div></div><div class="canvas full" id="labCanvas"><div class="world" id="labWorld"></div><div class="empty-canvas" id="emptyLab"><span style="font-size:30px">⌘</span><strong>Nenhum agente conectado ainda</strong><p>O Laboratório recebe agentes e linhas de terminal enviadas por ferramentas MCP. Não inicia tarefas sozinho nem inventa progresso.</p></div></div><p class="muted" style="margin-top:13px">Arraste o espaço para mover o mapa; use a roda do mouse para aproximar. Os cartões recebem atualizações a cada 7 segundos.</p></section>

<section id="contextPage" class="hidden">
<div id="contextIntegrations" class="integration-cards context-card-strip" aria-label="Conectar aplicativos"><span class="muted">Identificando integrações...</span></div>
<div id="contextCanvas" class="context-graph-infinite">
 <div class="graph-actions" aria-label="Controles do grafo">
  <input id="contextGraphSearch" placeholder="⌕ Buscar contexto ou pasta..." aria-label="Buscar nós no grafo" oninput="contextFilterGraph()" autocomplete="off">
  <button title="Aproximar" aria-label="Aproximar" onclick="contextZoom(1.22)">＋</button>
  <button title="Afastar" aria-label="Afastar" onclick="contextZoom(1/1.22)">−</button>
  <button title="Centralizar grafo" aria-label="Centralizar grafo" onclick="contextReset()">◎</button>
  <button title="Configurações do grafo" aria-label="Configurações do grafo" onclick="contextSettingsToggle()">⚙</button>
 </div>
 <div id="contextGraphSettings" class="context-graph-settings hidden"><b>Aparência do grafo</b><label>Repulsão <input type="range" id="contextRepel" min="25" max="180" value="96" oninput="contextForceChanged()"></label><label>Distância das ligações <input type="range" id="contextDistance" min="65" max="240" value="128" oninput="contextForceChanged()"></label><p>Arraste círculos para mover · espaço vazio para navegar · roda para zoom. Clique para prévia; botão direito para ações.</p></div>
 <svg id="contextSVG" viewBox="0 0 1200 720" preserveAspectRatio="none" aria-label="Grafo de conhecimento com nós arrastáveis, pastas e relações" role="img"></svg>
 <div id="contextEmpty" class="context-graph-empty hidden">O grafo aparecerá assim que houver fontes conectadas.<p>Use os cartões de integração acima.</p></div>
 <div id="contextGraphPreview" class="context-graph-preview hidden"><button class="context-preview-close" onclick="contextClosePreview()" aria-label="Fechar prévia">×</button><div id="contextDetailTitle" class="context-preview-title"></div><div id="contextDetailBody" class="context-preview-body"></div><button id="contextOpenObsidian" onclick="openContextInObsidian()" class="hidden">Abrir nota no Obsidian ↗</button></div>
 <div id="contextGraphMenu" class="context-graph-menu hidden"></div>
 <div class="context-index-status" id="contextIndexStatus"><strong id="contextIndexCount">Indexando...</strong><span id="contextIndexSubtitle">Buscando arquivos do Obsidian</span></div><div class="context-graph-caption"><span id="contextCounters">Carregando fontes...</span> <span>· Pastas agrupadas por origem · Arraste, aproxime e explore</span></div>
</div>
<div hidden aria-hidden="true"><select id="contextVaultSelect"></select><button id="contextAutoVaultBtn" onclick="connectObsidianAuto()">Conectar cofre</button><span id="contextHeroState"></span><span id="contextVaultName"></span><span id="contextSync"></span></div>
</section>
</main></div><div class="modalback" id="modalBack" onclick="if(event.target===this)closeModal()"><div class="modal" role="dialog" aria-modal="true"><div id="modalContent"></div><div class="group" style="margin-top:17px"><button onclick="closeModal()">Fechar</button></div></div></div><div id="toast" class="toast"></div><script>
const KEY='___TOKEN___';let state={roots:[],disks:[],agents:[],mode:'always'};let selectedDisk=0;let fetching=false;let activePage='panel';let agentViews={};let transform={x:0,y:0,z:1};let dragging=null;
function dom(tag,text,cls){let e=document.createElement(tag);if(text!==undefined)e.textContent=String(text);if(cls)e.className=cls;return e}
function toast(msg,bad=false){let e=document.getElementById('toast');e.textContent=msg;e.className='toast'+(bad?' bad':'');e.style.display='block';clearTimeout(window.toastHide);window.toastHide=setTimeout(()=>e.style.display='none',4500)}
async function api(path,data){let opt={headers:{'X-Local-Key':KEY}};if(data!==undefined){opt.method='POST';opt.headers['Content-Type']='application/json';opt.body=JSON.stringify(data)}let r=await fetch('/api/'+path,opt);let text=await r.text();let v;try{v=JSON.parse(text)}catch{throw Error(text)}if(!r.ok||v.ok===false)throw Error(v.error||'Falha '+r.status);return v}

async function copyConsecomSetupPrompt(){
 const field=document.getElementById('consecomSetupPrompt'),status=document.getElementById('consecomSetupCopyStatus');
 if(!field||!status)return;
 try{
  const prompt=field.value;
  if(navigator.clipboard&&navigator.clipboard.writeText){
   await navigator.clipboard.writeText(prompt);
  }else{
   field.closest('details').open=true;
   field.focus();field.select();
   if(!document.execCommand('copy'))throw Error('Cópia não autorizada');
  }
  status.textContent='Prompt copiado. Cole em uma conversa do ChatGPT que tenha o Consecom MCP conectado.';
  toast('Prompt de ativação copiado.');
 }catch(e){
  field.closest('details').open=true;
  field.focus();field.select();
  status.textContent='Não foi possível copiar automaticamente. Selecione o texto exibido e copie manualmente.';
  toast('Selecione e copie o prompt mostrado abaixo.',true);
 }
}

let liveComposeBusy=false;
let liveChatSignature='';
let liveChatBooted=false;
const latencyAcknowledged=new Set();
const liveDeliveryNames={queued:"Aguardando conexão",claimed:"Encaminhando ao ChatGPT",host_resolved:"Aguardando resposta",host_rejected:"Envio recusado",superseded:"Reenviada",answered:"Respondida"};
async function sendConsecomMessage(){
 const input=document.getElementById('liveUserMessage'),btn=document.getElementById('liveSendButton');
 if(liveComposeBusy)return;
 const message=input.value.trim();
 if(!message){toast('Escreva uma mensagem antes de enviar.',true);input.focus();return}
 if(new TextEncoder().encode(message).length>2000){toast('Mensagem acima de 2.000 bytes.',true);return}
 liveComposeBusy=true;btn.disabled=true;mark('liveComposeState','Enviando pelo MCP Consecom...');
 try{
  const result=(await api('live/send',{message})).result;
  input.value='';
  mark('liveComposeState','Mensagem '+result.id+' registrada. Aguardando resposta do ChatGPT.');
  await refreshConsecomChat(true);
 }catch(e){mark('liveComposeState',e.message);toast(e.message,true)}
 finally{liveComposeBusy=false;btn.disabled=false;input.focus()}
}
async function refreshConsecomChat(force=false){
 const box=document.getElementById('liveChatStream');
 if(!box)return;
 try{
  const response=(await api('live/chat')).result;
  const messages=Array.isArray(response.messages)?response.messages:[];
  const bridge=response.connection||{status:'disconnected',automatic_ready:false};
   const bridgeLabel=document.getElementById('liveBridgeConnection');
   if(bridgeLabel){
    const names={ready:'● Live detectado · recepção automática ativa',passive:'◐ Live detectado · envio automático não confirmado',disconnected:'○ Live não detectado · abra ou reabra o componente no ChatGPT'};
    bridgeLabel.textContent=names[bridge.status]||names.disconnected;
    bridgeLabel.style.color=bridge.status==='ready'?'#7cddba':'#e4ae73';
   }
   const lastPending=[...messages].reverse().find(m=>m.role==='user'&&m.status==='queued');
   let queueWarning='';
   if(lastPending&&!bridge.automatic_ready){
    const age=Date.now()-Date.parse(lastPending.created_at);
    if(Number.isFinite(age)&&age>15000){
     queueWarning='A mensagem está na fila há '+Math.round(age/1000)+'s. Reabra o Consecom Live no ChatGPT.';
    }
   }
   const staleClaim=messages.find(m=>m.role==='user'&&m.status==='claimed'&&!m.dispatched_at&&m.claimed_at&&Date.now()-Date.parse(m.claimed_at)>=120000);
   const claimWarning=staleClaim?'Mensagem '+staleClaim.id+' ficou sem confirmação de envio. Confira a conversa antes de reenviar.':'';
   const awaiting=messages.filter(m=>m.role==='user'&&m.status!=='answered'&&m.status!=='superseded'&&m.status!=='host_rejected'&&(m.status==='host_resolved'||Boolean(m.dispatched_at))).filter(m=>!messages.some(r=>r.role==='assistant'&&r.kind==='final'&&r.event_id===m.event_id));
   const typingFor=awaiting.length?awaiting[awaiting.length-1].event_id:'';
   const sig=messages.map(x=>x.id+':'+x.status+':'+(x.dispatched_at||'')).join('|')+'|typing:'+typingFor+'|stale:'+(staleClaim?staleClaim.id:'');
  if(!force&&sig===liveChatSignature){
   if(queueWarning||claimWarning)mark('liveComposeState',queueWarning||claimWarning);
   return;
  }
  liveChatSignature=sig;
  const nearBottom=box.scrollHeight-box.scrollTop-box.clientHeight<90;
  const fragment=document.createDocumentFragment();
  const sentAt=new Map(messages.filter(m=>m.role==="user").map(m=>[m.event_id,Date.parse(m.created_at)]));
  if(messages.length===0){
   fragment.append(dom('div','Escreva sua primeira mensagem. As respostas do ChatGPT aparecerão aqui, sem API paga.','live-chat-empty'));
  }
  for(const entry of messages){
   const row=dom('div',undefined,'live-chat-row'+(entry.role==='user'?' mine':'')+(entry.kind==='andamento'?' progress':''));
   const msg=dom('div',entry.text||'','live-chat-bubble');
   const meta=dom('div',undefined,'live-chat-meta');
   let stamp='';
   try{stamp=new Date(entry.created_at).toLocaleTimeString('pt-BR',{hour:'2-digit',minute:'2-digit'})}catch(e){}
   const who=entry.role==='user'?'Você':entry.kind==='andamento'?'ChatGPT · trabalhando':'ChatGPT';
   const startTime=sentAt.get(entry.event_id),endTime=Date.parse(entry.created_at);
   const elapsed=entry.role==='assistant'&&Number.isFinite(startTime)&&Number.isFinite(endTime)&&endTime>=startTime?
      ' · '+Math.round((endTime-startTime)/1000)+'s desde envio':'';
   meta.append(dom('span',who),dom('span',stamp+(entry.role==='user'?' · '+(liveDeliveryNames[entry.status]||entry.status):elapsed)));
   row.append(msg,meta);
   const retryable=entry.role==='user'&&(entry.status==='host_rejected'||(entry.status==='claimed'&&!entry.dispatched_at&&entry.claimed_at&&Date.now()-Date.parse(entry.claimed_at)>=120000));
   if(retryable){
    const retry=dom('button',entry.status==='claimed'?'Verificar e reenviar':'Reenviar mensagem');
    retry.onclick=()=>retryConsecomMessage(entry.id);
    retry.style.marginTop='8px';row.append(retry);
   }
   fragment.append(row);
  }
  if(typingFor){
   // This is a waiting animation, not streamed output or evidence that the
   // model has generated tokens. It appears only after confirmed dispatch.
   const row=dom('div',undefined,'live-chat-row');
   row.setAttribute('data-typing-for',typingFor);
   const bubble=dom('div',undefined,'live-chat-bubble');
   const line=dom('div',undefined,'live-chat-typing');
   line.setAttribute('role','status');
   line.setAttribute('aria-label','Mensagem encaminhada; aguardando resposta do ChatGPT');
   line.title='Indicador de espera: não comprova que o modelo esteja digitando em tempo real.';
   const dots=dom('span',undefined,'live-chat-typing-dots');
   dots.setAttribute('aria-hidden','true');
   for(let i=0;i<3;i++)dots.append(dom('i'));
   line.append(dom('span','ChatGPT está digitando'),dots);
   bubble.append(line);
   const meta=dom('div','Mensagem encaminhada · aguardando resposta','live-chat-meta');
   row.append(bubble,meta);
   fragment.append(row);
  }
  box.replaceChildren(fragment);
  // One authenticated acknowledgement per recently displayed final message.
  // Capture the browser-render time, not the next 900ms polling deadline.
  const renderStamp=new Date().toISOString();
  for(const entry of messages){
   if(entry.role!=='assistant'||entry.kind!=='final'||latencyAcknowledged.has(entry.id))continue;
   const age=Date.now()-Date.parse(entry.created_at);
   if(!Number.isFinite(age)||age<0||age>600000)continue;
   latencyAcknowledged.add(entry.id);
   void api('live/displayed',{event_id:entry.event_id,reply_id:entry.id,observed_at:renderStamp})
    .catch(()=>latencyAcknowledged.delete(entry.id));
  }
  if(!liveChatBooted||nearBottom||force)box.scrollTop=box.scrollHeight;
  liveChatBooted=true;
  const last=messages[messages.length-1];
  if(last){
   if(queueWarning||claimWarning)mark('liveComposeState',queueWarning||claimWarning);
   else if(typingFor)mark('liveComposeState','Mensagem encaminhada ao ChatGPT. Aguardando a resposta...');
   else if(last.role==='assistant'&&last.kind==='final')mark('liveComposeState','Resposta recebida do ChatGPT.');
   else if(last.role==='user')mark('liveComposeState',liveDeliveryNames[last.status]||'Aguardando ChatGPT...');
  }
 }catch(e){mark('liveComposeState','Chat temporariamente indisponível: '+e.message)}
}
async function showConsecomLatency(){
 try{
  const data=(await api('live/latency')).result;
  const latest=[...(data.messages||[])].reverse().find(x=>x.durations&&x.durations.total_to_reply_ms!==undefined);
  if(!latest){mark('liveComposeState','Ainda não existem respostas concluídas para diagnosticar.');return}
  const ms=latest.durations;
  const sec=n=>n===undefined?'—':(n/1000).toFixed(2)+'s';
  mark('liveComposeState','Latência '+latest.event_id+': fila '+sec(ms.queue_ms)+' · reivindicação→envio '+sec(ms.claim_to_followup_ms)+' · envio→resposta '+sec(ms.followup_to_reply_ms)+' · resposta→painel '+sec(ms.reply_to_display_ms)+' · total '+sec(ms.total_to_reply_ms)+'. Maior etapa: '+latest.bottleneck+'.');
 }catch(e){toast('Não foi possível consultar a latência: '+e.message,true)}
}
async function retryConsecomMessage(id){
 if(!confirm('Reenviar esta mensagem? Ela pode ser recebida duas vezes no ChatGPT.'))return;
 try{const r=(await api('live/retry',{id})).result;toast('Nova tentativa '+r.id);await refreshConsecomChat(true)}
 catch(e){toast(e.message,true)}
}
const liveComposeInput=document.getElementById('liveUserMessage');
if(liveComposeInput)liveComposeInput.addEventListener('keydown',function(e){
 if(e.key==='Enter'&&!e.shiftKey&&!e.isComposing){
  e.preventDefault();
  void sendConsecomMessage();
 }
});
setInterval(()=>{if(activePage==='panel'&&!document.hidden)void refreshConsecomChat()},900);
void refreshConsecomChat(true);
async function waitForPluginRestart(previousVersion){const until=Date.now()+90000;while(Date.now()<until){await new Promise(resolve=>setTimeout(resolve,1000));try{const r=await fetch('/health?updated='+Date.now(),{cache:'no-store'});if(!r.ok)continue;const body=await r.json();if(body.service==='consecom-mcp'&&body.version!==previousVersion){location.replace('/?updated='+Date.now());return}}catch(e){}}mark('updateSummary','Atualização iniciada. Reabra o painel se ele não reconectar.')}
async function updateNow(){const btn=document.getElementById('sideUpdateBtn');if(btn.disabled)return;btn.disabled=true;btn.textContent='↻ Verificando...';mark('updateSummary','Buscando atualização oficial...');const previousVersion=state.updates?.version||'';try{let r=(await api('updates/local',{})).result;if(r.status==='sem_atualizacao'){toast('Consecom já está na última versão ('+r.latest+')');mark('updateSummary','Você já está atualizado')}else if(r.status==='iniciada'){toast(r.message||'Instalador validado e iniciado');mark('updateSummary','Aguardando reinício para recarregar o painel...');waitForPluginRestart(previousVersion)}else{toast(r.message||'Atualização solicitada');mark('updateSummary','Aguardando atualização')}}catch(e){toast(e.message,true);mark('updateSummary','Falha: '+e.message)}finally{btn.disabled=false;btn.textContent='↻ Atualizar'}}
function gb(n){return(Number(n||0)/1073741824).toFixed(1)+' GB'}
function percent(n){return Math.max(0,Math.min(100,n||0))+'%'}
function nextDisk(dir=1){if(!state.disks.length)return;selectedDisk=(selectedDisk+dir+state.disks.length)%state.disks.length;showDisk()}
function showDisk(){let d=state.disks[selectedDisk];document.getElementById('diskLabel').textContent=d?'Disco ('+d.label+')':'Disco (indisponível)';document.getElementById('diskUsed').textContent=d?gb(d.used):'—';document.getElementById('diskTotal').textContent=d?'de '+gb(d.total)+' • clique para alternar':'Nenhum disco legível';document.getElementById('diskBar').style.width=d?percent(d.used/d.total*100):'0%'}
function mark(id,value){document.getElementById(id).textContent=value}
function page(p){activePage=p;document.getElementById('panelPage').classList.toggle('hidden',p!=='panel');document.getElementById('labPage').classList.toggle('hidden',p!=='lab');document.getElementById('contextPage').classList.toggle('hidden',p!=='context');document.getElementById('navPanel').classList.toggle('active',p==='panel');document.getElementById('navLab').classList.toggle('active',p==='lab');document.getElementById('navContext').classList.toggle('active',p==='context');history.replaceState({},'',p==='lab'?'#agentes':p==='context'?'#contexto':'#painel');if(p==='context')reloadContext();else renderAgents();window.scrollTo(0,0)}
function showMode(){for(let [v,id] of [['always','modeAlways'],['category','modeCategory'],['maximum','modeMaximum']])document.getElementById(id).classList.toggle('selected',state.mode===v);const modeIndex={always:0,category:1,maximum:2};document.getElementById('modeSlider').value=modeIndex[state.mode]??0;previewModeSlider(modeIndex[state.mode]??0);let max=state.mode==='maximum';mark('scopeTitle',max?'Acesso ampliado aos discos locais':'Pastas explicitamente autorizadas');mark('scopeDesc',max?'Pode ler, criar, editar e mover arquivos; excluir com recuperação e restaurar backups. Sistema, credenciais e áreas protegidas continuam bloqueados.':state.roots.length+' pasta(s) autorizada(s). A leitura fica restrita a elas; escrita precisa ser habilitada por pasta.');document.getElementById('scopeActions').classList.toggle('hidden',max)}
function previewModeSlider(value){const names=['Perguntar sempre','Aprovar por categoria','Aprovação máxima'];const hints=['Confirma cada ação sensível','Autoriza por tipo de operação','Automatiza operações permitidas, mantendo restrições de segurança'];mark('approvalCurrentName',names[Number(value)]||names[0]);mark('approvalHint',hints[Number(value)]||hints[0]);}
async function commitModeSlider(value){const modes=['always','category','maximum'];try{await setMode(modes[Number(value)]||'always')}finally{showMode()}}
async function setMode(mode){if(mode==='maximum'){let ok=confirm('ATENÇÃO: em Aprovação máxima, comandos, janelas do Windows, alterações Git e cliques/uploads no navegador podem ser executados SEM confirmação repetida. O modo não permite acesso a credenciais/sistema, comandos arbitrários nem caminhos protegidos. Ativar acesso ampliado? Ativar?');if(!ok)return;let phrase=prompt('Para confirmar localmente, digite LIBERAR:');if(phrase!=='LIBERAR'){toast('Aprovação máxima não ativada.',true);return}await sendMode(mode,phrase)}else{await sendMode(mode,'')}}
async function sendMode(mode,confirm){try{await api('policy',{mode,confirm});state.mode=mode;showMode();toast('Política atualizada: '+mode)}catch(e){toast(e.message,true)}}
function renderPending(list){let wrap=document.getElementById('pendingWrap'),box=document.getElementById('pendingList');box.replaceChildren();wrap.classList.toggle('hidden',!list.length);mark('pendingTitle',list.length+' aprovação(ões) pendente(s)');for(let p of list){let line=dom('div',undefined,'pendingitem'),label=dom('div');label.append(dom('b',p.kind.toUpperCase()),dom('small',p.description));if(p.preview)label.append(dom('small','Prévia: '+p.preview));let actions=dom('div');let no=dom('button','Negar'),yes=dom('button','Aprovar','primary');no.onclick=async()=>{try{await api('actions/deny',{id:p.id});await refresh()}catch(e){toast(e.message,true)}};yes.onclick=async()=>{if(!confirm('Executar esta ação no computador?\n'+p.description))return;try{let r=await api('actions/approve',{id:p.id});toast(r.result||'Ação autorizada');await refresh()}catch(e){toast(e.message,true)}};actions.append(no,yes);line.append(label,actions);box.append(line)}}
function makeAgent(agent,i){let statusMap={running:'Em execução',waiting:'Aguardando',done:'Concluído',error:'Erro'};let colors=['#a465ff','#19a7ff','#f6ae41','#15d69b'];let node=dom('article',undefined,'agent '+agent.status);node.style.borderColor=colors[i%4];node.dataset.id=agent.id;let header=dom('div',undefined,'agenttitle'),badge=dom('span',i%2===0?'◈':'⌘','agenticon'),head=dom('div');head.append(dom('b',agent.name),dom('small',(statusMap[agent.status]||'Ativo')+' • '+(agent.source||'MCP')));header.append(badge,head);node.append(header);let body=dom('div',undefined,'agentbody');let term=dom('div',undefined,'terminal');let lines=(agent.logs||[]).slice(-5).map(x=>'> '+x.text).join('\n');term.textContent=(agent.task||'Sem tarefa informada')+'\n'+(lines||'> Aguardando saída real...');body.append(term);node.append(body);let pos=agentViews[agent.id]||{x:65+(i%3)*360,y:95+Math.floor(i/3)*235};node.style.left=pos.x+'px';node.style.top=pos.y+'px';node.addEventListener('pointerdown',e=>{if(e.button!==0||e.target.closest('.terminal'))return;if(activePage!=='lab')return;dragging={id:agent.id,sx:e.clientX,sy:e.clientY,oldX:pos.x,oldY:pos.y};e.stopPropagation()});return node}
function renderAgents(){for(const type of ['preview','lab']){let world=document.getElementById(type+'World'),empty=document.getElementById('empty'+(type==='preview'?'Preview':'Lab'));world.replaceChildren();let agents=state.agents;empty.classList.toggle('hidden',agents.length>0);agents.slice(0,type==='preview'?4:36).forEach((a,i)=>{let el=makeAgent(a,i);if(type==='preview'){el.style.left=(40+i*350)+'px';el.style.top='78px'}world.append(el)});if(type==='preview'){world.style.transform='scale(.72)'}else{updateView()}}mark('agentCount',state.agents.length+' agente(s) reportando atividades')}
function updateView(){document.getElementById('labWorld').style.transform='translate('+transform.x+'px,'+transform.y+'px) scale('+transform.z+')'}
function zoomBy(delta){transform.z=Math.min(2.25,Math.max(.45,transform.z+delta));updateView()}
function resetView(){transform={x:0,y:0,z:1};updateView()}
const labCanvas=document.getElementById('labCanvas');labCanvas.addEventListener('wheel',e=>{e.preventDefault();zoomBy(e.deltaY<0?.1:-.1)},{passive:false});labCanvas.addEventListener('pointerdown',e=>{if(e.target.closest('.agent'))return;dragging={canvas:true,x:e.clientX,y:e.clientY,tx:transform.x,ty:transform.y}});window.addEventListener('pointermove',e=>{if(!dragging)return;if(dragging.canvas){transform.x=dragging.tx+e.clientX-dragging.x;transform.y=dragging.ty+e.clientY-dragging.y;updateView()}else{agentViews[dragging.id]={x:dragging.oldX+(e.clientX-dragging.sx)/transform.z,y:dragging.oldY+(e.clientY-dragging.sy)/transform.z};let target=document.querySelector('#labWorld [data-id="'+dragging.id+'"]');if(target){target.style.left=agentViews[dragging.id].x+'px';target.style.top=agentViews[dragging.id].y+'px'}}});window.addEventListener('pointerup',()=>dragging=null);
async function refresh(){if(fetching)return;fetching=true;try{let s=await api('status'),m=s.metrics||{};mark('ramUsed',gb(m.used_ram_bytes));mark('ramFree',gb(m.available_ram_bytes));mark('ramTotal','de '+gb(m.total_ram_bytes)+' físicos');mark('cpu',String(m.cpu_percent??'—')+'%');let total=m.total_ram_bytes||1;document.getElementById('ramUsedBar').style.width=percent(m.used_ram_bytes/total*100);document.getElementById('ramFreeBar').style.width=percent(m.available_ram_bytes/total*100);document.getElementById('cpuBar').style.width=percent(m.cpu_percent);let diskName=state.disks[selectedDisk]?.label;state.disks=s.disks||[];if(diskName){let i=state.disks.findIndex(d=>d.label===diskName);if(i>=0)selectedDisk=i}if(selectedDisk>=state.disks.length)selectedDisk=0;showDisk();state.roots=s.roots||[];state.mode=s.system?.mode||'always';state.agents=s.agents||[];state.tunnel=s.tunnel||{};state.updates=s.updates||{};if(!document.getElementById('sideUpdateBtn').disabled)mark('updateSummary','Repositório oficial • '+(state.updates.version||'—'));let bp=s.browser?.default||{};mark('browserSummary',bp.browser?'Sessão exclusiva • '+bp.browser.toUpperCase()+' • pronta sob demanda':'Aguardando navegador instalado');showMode();renderAgents();renderPending(s.pending||[]);let connected=s.system?.mcp_recent===true;mark('chatStatus',connected?'MCP usado recentemente':state.tunnel.status==='pronto'?'Túnel pronto':state.tunnel.status==='conectando'?'Conectando túnel...':state.tunnel.status==='iniciando'?'Iniciando túnel...':'Aguardando ChatGPT');mark('sideConnection',connected?'MCP respondeu a uma chamada recente':'Sem atividade MCP recente');let started=Date.parse(s.system?.started_at||'');mark('uptime',Number.isFinite(started)?'Ativo há '+Math.floor((Date.now()-started)/60000)+' min':'Servidor ativo');}catch(e){mark('chatStatus','Servidor indisponível');toast('Falha: '+e.message,true)}finally{fetching=false}}

function connectionModal(root){
  root.append(dom('h2','Conectar ao ChatGPT'),dom('p','Conexão privada com a OpenAI, sem CMD. Configure apenas uma vez; o Consecom manterá o túnel ativo quando iniciar com o Windows.'));
  let st=state.tunnel||{};
  let status=dom('p',st.installed?'Cliente oficial v0.0.16 instalado.':'Cliente oficial ausente: execute o instalador Consecom Plugin v4.7.0 conectado à internet.');root.append(status);
  let t=dom('input');t.placeholder='Cole o ID tunnel_...';t.value=st.tunnel_id||'';t.autocomplete='off';
  let k=dom('input');k.type='password';k.placeholder='Cole a chave de execução da OpenAI Platform (sk-...)';k.autocomplete='off';
  let field1=dom('label','Identificador do túnel');let field2=dom('label','Chave da OpenAI Platform (Tunnels Read + Use)');
  root.append(field1,t,field2,k);
  let hint=dom('p','Essa chave é salva apenas no seu Windows, criptografada para o usuário atual. Ela não será mostrada novamente, nem enviada para esta conversa.');root.append(hint);
  let links=dom('div',undefined,'group');links.style.justifyContent='flex-start';
  for(let [label,url] of [['Ver meus túneis','https://platform.openai.com/settings/organization/tunnels'],['Criar chave de execução','https://platform.openai.com/settings/organization/api-keys']]){let a=dom('a',label);a.href=url;a.target='_blank';a.rel='noopener noreferrer';a.style.color='#59c8ff';a.style.marginRight='18px';links.append(a)}
  root.append(links);
  let stateLine=dom('p','Status atual: '+(st.status||'desconectado'));root.append(stateLine);
  if(st.message)root.append(dom('p',st.message));
  let actions=dom('div',undefined,'group');actions.style.marginTop='12px';
  let go=dom('button','Salvar e conectar','primary');go.onclick=async()=>{go.disabled=true;try{await api('tunnel/connect',{tunnel_id:t.value.trim(),api_key:k.value.trim()});k.value='';toast('Cliente de túnel iniciado. Aguarde o status Pronto.');await refresh();openModal('connection')}catch(e){toast(e.message,true);go.disabled=false}};
  actions.append(go);
  if(st.configured){let stop=dom('button','Desconectar e apagar chave');stop.onclick=async()=>{if(!confirm('Desconectar e remover a chave salva neste computador?'))return;try{await api('tunnel/disconnect',{});toast('Túnel desconectado');await refresh();openModal('connection')}catch(e){toast(e.message,true)}};actions.append(stop)}
  root.append(actions);
  root.append(dom('p','Depois que aparecer "Túnel pronto", entre em ChatGPT → Plugins → Adicionar servidor MCP personalizado → Túnel, cole o mesmo ID e conclua.'));
}


async function browserModal(root){
 root.append(dom('h2','Navegador exclusivo'),dom('p','O Consecom usa uma sessão separada com dados próprios. Escolha Chrome ou Microsoft Edge; não haverá conexão ao perfil pessoal nem avisos para depurar suas abas existentes.'));
 const box=dom('div','Consultando os navegadores...');root.append(box);
 try{
  const v=await api('browser/options');box.replaceChildren();
  const available=(v.browsers||[]).filter(x=>x.installed);
  if(!available.length){box.append(dom('p','Instale Google Chrome ou Microsoft Edge para habilitar a sessão exclusiva.'));return}
  const title=dom('label','Qual navegador o Consecom deve usar?'),choices=dom('div',undefined,'exclusive-browser-choice');
  let selected=v.default?.browser||available[0].id;
  const buttons=[];
  for(const choice of available){
   const btn=dom('button',(choice.id==='chrome'?'◉ ':'◇ ')+choice.name);
   btn.type='button';
   btn.onclick=()=>{selected=choice.id;for(const b of buttons)b.classList.toggle('selected',b.dataset.browser===selected)};
   btn.dataset.browser=choice.id;buttons.push(btn);choices.append(btn);
  }
  for(const btn of buttons)btn.classList.toggle('selected',btn.dataset.browser===selected);
  const hint=dom('p','Perfil permanente: Consecom exclusivo. Seus cookies e senhas pessoais não são copiados. Para sites que exigem login, entre uma vez nessa sessão separada.');
  const button=dom('button','Usar navegador selecionado','primary');
  button.onclick=async()=>{button.disabled=true;try{await api('browser/default',{browser:selected,profile:'managed:chatgpt'});toast('Navegador exclusivo definido. Consecom usará '+selected.toUpperCase());await refresh();closeModal()}catch(e){toast(e.message,true);button.disabled=false}};
  box.append(title,choices,hint,button);
 }catch(e){box.textContent='Falha ao consultar navegadores: '+e.message}
}


async function updatesModal(root){
 let opt=await api('updates/status');
 root.append(dom('h2','Política de atualização remota'),dom('p','Origem oficial fixa: '+opt.repo+'. O botão Atualizar no canto inferior esquerdo verifica, baixa, confere o SHA-256 e inicia o instalador sem pedir cadastro de repositório.'));
 let box=dom('label');box.style.display='flex';box.style.alignItems='center';box.style.gap='9px';
 let checkbox=dom('input');checkbox.type='checkbox';checkbox.style.width='auto';checkbox.checked=!!opt.remote_approved;
 box.append(checkbox,dom('span','Permitir que o ChatGPT inicie instalações pelo MCP sem aprovação presencial'));
 let warn=dom('p','A permissão remota permite executar atualizações futuras publicadas na origem oficial. Por padrão, pedidos do ChatGPT exigem aprovação no Windows.');
 let confirmLabel=dom('label','Para liberar remotamente, escreva: CONFIO NAS ATUALIZACOES'),ack=dom('input');ack.placeholder='CONFIO NAS ATUALIZACOES';
 let save=dom('button','Salvar permissão','primary');save.onclick=async()=>{try{await api('updates/settings',{remote_approved:checkbox.checked,confirm:ack.value.trim()});toast('Permissão atualizada');await refresh();closeModal()}catch(e){toast(e.message,true)}};
 root.append(box,warn,confirmLabel,ack,save);
}

function closeModal(){document.getElementById('modalBack').style.display='none'}
function openModal(which){let root=document.getElementById('modalContent');root.replaceChildren();if(which==='connection'){connectionModal(root);}else if(which==='browser'){browserModal(root);}else if(which==='updates'){updatesModal(root).catch(e=>toast(e.message,true));}else if(which==='folders'){root.append(dom('h2','Pastas autorizadas'),dom('p','Adicione apenas pastas de projetos. Acesso de escrita é opcional; no modo máximo o acesso ampliado continua bloqueando caminhos protegidos.'));for(let r of state.roots){let line=dom('div',undefined,'rootline'),info=dom('div');info.append(dom('b',r.name),dom('small',r.path));let btn=dom('button',r.writable?'Desativar escrita':'Habilitar escrita');btn.onclick=async()=>{try{await api('roots/writable',{id:r.id,writable:!r.writable});await refresh();openModal('folders')}catch(e){toast(e.message,true)}};let del=dom('button','Remover');del.onclick=async()=>{if(!confirm('Revogar pasta?'))return;try{await api('roots/delete',{id:r.id});await refresh();openModal('folders')}catch(e){toast(e.message,true)}};line.append(info,btn,del);root.append(line)}let nm=dom('input');nm.placeholder='Nome (ex: Projeto Prospector)';let path=dom('input');path.placeholder='C:\\Projetos ou D:\\Prospector';let add=dom('button','Adicionar pasta','primary');add.onclick=async()=>{try{await api('roots/add',{name:nm.value,path:path.value});toast('Pasta autorizada para leitura');await refresh();openModal('folders')}catch(e){toast(e.message,true)}};root.append(nm,path,add)}document.getElementById('modalBack').style.display='grid'}
if(location.hash==='#agentes')page('lab');else if(location.hash==='#contexto')page('context');else page('panel');refresh();setInterval(refresh,7000);

let contextData={sources:[],notes:[],index:{}},contextScale=1,contextOffset={x:0,y:0};
let contextSelectedNote=null,contextDrag=null,contextHoverID=null,contextSelectedID=null;
let contextGraphNodes=new Map(),contextGraphEdges=[],contextGraphParts=new Map(),contextGraphLines=[];
let contextGraphWorld=null,contextGraphAnimation=0,contextGraphFrames=0;
let contextRepulsion=96,contextLinkDistance=128;
const contextCanvas=document.getElementById('contextCanvas'),contextSVG=document.getElementById('contextSVG');
function svgElement(tag,attrs){const el=document.createElementNS('http://www.w3.org/2000/svg',tag);for(const k in attrs)el.setAttribute(k,attrs[k]);return el}
function contextGraphNormalize(name){return String(name||'').toLowerCase().replace(/[\s_.-]+/g,'')}
function contextGraphAppID(name){
 const s=contextGraphNormalize(name);
 if(s.includes('claude')||s.includes('cloudcode'))return 'claude-code';
 if(s.includes('opencode'))return 'opencode';
 if(s.includes('codex'))return 'codex';
 if(s.includes('chatgpt')||s.includes('openai'))return 'chatgpt';
 if(s.includes('cursor'))return 'cursor';
 if(s.includes('obsidian'))return 'obsidian';
 return '';
}
function contextGraphBuild(){
 const previous=contextGraphNodes;
 const nodes=new Map(),edges=[];
 function add(id,label,type,ref,color,parent){
  if(nodes.size>=255)return false;
  const old=previous.get(id),seq=nodes.size;
  let angle=seq*2.399963229728653,radius=75+41*Math.sqrt(seq);
  let x=600+Math.cos(angle)*radius,y=350+Math.sin(angle)*radius;
  if(parent&&nodes.has(parent)&&!old){
    const root=nodes.get(parent);
    const count=[...nodes.values()].filter(x=>x.parent===parent).length;
    const alpha=count*2.399963229728653;
    const spread=63+21*Math.sqrt(count);
    x=root.x+Math.cos(alpha)*spread;y=root.y+Math.sin(alpha)*spread;
  }
  nodes.set(id,{id,label,type,ref,color,parent,
   x:old?old.x:x,y:old?old.y:y,
   vx:old?old.vx:0,vy:old?old.vy:0,pinned:old?old.pinned:false});
  return true;
 }
 const index=contextData.index||{};
 if(contextData.vault_selected)add('vault','Obsidian · '+contextData.vault_name,'vault',null,'#bca6ff');
 const items=window.contextIntegrationItems||[];
 for(const app of items){
  if(app.id==='obsidian'||!(app.ready||app.configured||app.observed||app.detected))continue;
  add('app:'+app.id,app.label,'app',app,'#70cbbd');
 }
 for(const source of contextData.sources||[]){
  const id='folder:'+source.id;
  add(id,source.label,'folder',source,'#9bace7','app:'+contextGraphAppID(source.label));
  const app='app:'+contextGraphAppID(source.label);
  if(nodes.has(app))edges.push({a:app,b:id,type:'category'});
  else if(nodes.has('vault'))edges.push({a:'vault',b:id,type:'category'});
 }
 const sourceNotes=(contextData.notes||[]).slice(0,100);
 for(const note of sourceNotes){
  const id='note:'+note.id,from='folder:'+note.source_id;
  if(add(id,note.title,'note',note,'#69caa8',from)&&nodes.has(from))edges.push({a:from,b:id,type:'category'});
 }
 for(const note of sourceNotes)for(const linked of note.links||[]){
  if(nodes.has('note:'+note.id)&&nodes.has('note:'+linked))edges.push({a:'note:'+note.id,b:'note:'+linked,type:'link'});
 }
 // Physical Obsidian folders appear as their own clusters. This is a
 // read-only metadata index, not an import of conversation contents.
 const folders=(index.folders||[]).slice(0,48);
 for(const folder of folders){
  const id='vaultfolder:'+folder.path;
  if(add(id,folder.name+' · '+folder.count,'indexed-folder',folder,'#ad92eb','vault')&&nodes.has('vault')){
   edges.push({a:'vault',b:id,type:'category'});
  }
 }
 const knownFolders=new Set(folders.map(f=>f.path));
 const files=(index.files||[]).filter(f=>knownFolders.has(f.folder)).slice(0,140);
 for(const file of files){
  const folderID='vaultfolder:'+file.folder,id='vaultfile:'+file.path;
  if(add(id,file.title,'indexed-file',file,'#85a8b8',folderID)&&nodes.has(folderID)){
   edges.push({a:folderID,b:id,type:'category'});
  }
 }
 if(nodes.has('vault')){
  for(const app of items){
   if(app.id==='obsidian'||!app.ready||!nodes.has('app:'+app.id))continue;
   edges.push({a:'vault',b:'app:'+app.id,type:'transport'});
  }
 }
 contextGraphNodes=nodes;contextGraphEdges=edges;
}
function contextGraphTransform(){
 if(contextGraphWorld)contextGraphWorld.setAttribute('transform','translate('+contextOffset.x+' '+contextOffset.y+') scale('+contextScale+')');
 contextCanvas.style.backgroundPosition=(contextOffset.x*contextSVG.clientWidth/1200)+'px '+(contextOffset.y*contextSVG.clientHeight/720)+'px';
}
function contextGraphUpdateGeometry(){
 for(let i=0;i<contextGraphEdges.length;i++){
  const edge=contextGraphEdges[i],line=contextGraphLines[i],a=contextGraphNodes.get(edge.a),b=contextGraphNodes.get(edge.b);
  if(!line||!a||!b)continue;
  line.setAttribute('x1',a.x);line.setAttribute('y1',a.y);line.setAttribute('x2',b.x);line.setAttribute('y2',b.y);
 }
 for(const [id,el] of contextGraphParts){
  const node=contextGraphNodes.get(id);if(node)el.setAttribute('transform','translate('+node.x+' '+node.y+')');
 }
}
function contextGraphTick(){
 contextGraphAnimation=0;
 if(activePage!=='context'||contextGraphFrames>260||contextGraphNodes.size===0)return;
 contextGraphFrames++;
 const nodes=[...contextGraphNodes.values()];
 // Force-directed layout: repulsion, springs, gentle gravity and velocity damping.
 for(let i=0;i<nodes.length;i++){
  for(let j=i+1;j<nodes.length;j++){
   const a=nodes[i],b=nodes[j];
   let dx=a.x-b.x,dy=a.y-b.y;
   let d2=Math.max(160,dx*dx+dy*dy);
   const force=contextRepulsion*30/d2;
   const fx=dx*force/Math.sqrt(d2),fy=dy*force/Math.sqrt(d2);
   if(!a.pinned){a.vx+=fx;a.vy+=fy}
   if(!b.pinned){b.vx-=fx;b.vy-=fy}
  }
 }
 for(const edge of contextGraphEdges){
  const a=contextGraphNodes.get(edge.a),b=contextGraphNodes.get(edge.b);if(!a||!b)continue;
  const dx=b.x-a.x,dy=b.y-a.y,dist=Math.max(1,Math.hypot(dx,dy));
  const target=contextLinkDistance*(edge.type==='transport'?1.6:edge.type==='link'?.85:1);
  const strength=(dist-target)*.013;
  const fx=dx/dist*strength,fy=dy/dist*strength;
  if(!a.pinned){a.vx+=fx;a.vy+=fy}
  if(!b.pinned){b.vx-=fx;b.vy-=fy}
 }
 for(const n of nodes){
  if(n.pinned){n.vx=0;n.vy=0;continue}
  n.vx+=(600-n.x)*.0008;n.vy+=(350-n.y)*.0008;
  n.vx*=.84;n.vy*=.84;
  n.x+=Math.max(-8,Math.min(8,n.vx));
  n.y+=Math.max(-8,Math.min(8,n.vy));
 }
 contextGraphUpdateGeometry();
 if(contextGraphFrames<260)contextGraphAnimation=requestAnimationFrame(contextGraphTick);
}
function contextGraphResume(){
 contextGraphFrames=0;
 if(!contextGraphAnimation)contextGraphAnimation=requestAnimationFrame(contextGraphTick);
}
function contextGraphHighlight(){
 const term=(document.getElementById('contextGraphSearch')?.value||'').toLowerCase().trim();
 const neighborhood=new Set();
 if(contextHoverID){
  neighborhood.add(contextHoverID);
  for(const edge of contextGraphEdges){if(edge.a===contextHoverID)neighborhood.add(edge.b);if(edge.b===contextHoverID)neighborhood.add(edge.a)}
 }
 const matched=new Set();
 if(term)for(const n of contextGraphNodes.values())if(n.label.toLowerCase().includes(term))matched.add(n.id);
 for(const [id,el] of contextGraphParts){
  const n=contextGraphNodes.get(id);
  const searchDim=term&& !matched.has(id) ? .12:1;
  const hoverDim=contextHoverID && !neighborhood.has(id) ? .17:1;
  el.style.opacity=String(Math.min(searchDim,hoverDim));
  el.style.pointerEvents=searchDim===.12?'none':'auto';
 }
 for(let i=0;i<contextGraphEdges.length;i++){
  const edge=contextGraphEdges[i],line=contextGraphLines[i];
  const connected=contextHoverID&&(edge.a===contextHoverID||edge.b===contextHoverID);
  const searchDim=term && !matched.has(edge.a)&&!matched.has(edge.b);
  line.style.opacity=String(searchDim?.06:contextHoverID?(connected?.85:.06):edge.type==='transport'?.26:.39);
  line.setAttribute('stroke-width',connected?2.3:1.3);
 }
}
function contextFilterGraph(){contextGraphHighlight()}
function drawContext(){
 contextGraphBuild();
 contextGraphParts=new Map();contextGraphLines=[];
 contextSVG.replaceChildren();
 const world=svgElement('g',{});contextSVG.append(world);contextGraphWorld=world;
 const lines=svgElement('g',{});world.append(lines);
 for(const edge of contextGraphEdges){
  const line=svgElement('line',{class:'context-edge',stroke:edge.type==='link'?'#ae9be9':'#658595'});
  if(edge.type==='transport')line.setAttribute('stroke-dasharray','4 7');
  lines.append(line);contextGraphLines.push(line);
 }
 const points=svgElement('g',{});world.append(points);
 for(const n of contextGraphNodes.values()){
  const cls='context-node '+n.type;
  const el=svgElement('g',{class:cls,tabindex:'0',role:'button','aria-label':n.label,'data-node':n.id});
  const size=n.type==='vault'?26:n.type==='app'?16:n.type==='indexed-folder'?Math.min(23,11+Math.sqrt(n.ref?.count||1)):n.type==='folder'?13:n.type==='indexed-file'?5.5:6;
  const circle=svgElement('circle',{cx:0,cy:0,r:size,fill:n.color});
  const title=svgElement('title',{});title.textContent=n.label;
  circle.append(title);el.append(circle);
  const label=svgElement('text',{x:0,y:size+17});label.textContent=n.label.slice(0,n.type==='indexed-file'?34:42);
  el.append(label);
  el.addEventListener('pointerenter',()=>{contextHoverID=n.id;contextGraphHighlight()});
  el.addEventListener('pointerleave',()=>{contextHoverID=null;contextGraphHighlight()});
  el.addEventListener('keydown',e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();showContextNode(n)}});
  points.append(el);contextGraphParts.set(n.id,el);
 }
 contextGraphTransform();contextGraphUpdateGeometry();contextGraphHighlight();contextGraphResume();
 document.getElementById('contextEmpty').classList.toggle('hidden',contextGraphNodes.size>0);
 const metadata=contextData.index||{};
 mark('contextCounters',contextGraphNodes.size+' nós visíveis · '+contextGraphEdges.length+' relações · '+(contextData.notes||[]).length+' notas Consecom');
 contextIndexBadge(metadata);
}
function contextSettingsToggle(){document.getElementById('contextGraphSettings').classList.toggle('hidden')}
function contextForceChanged(){
 contextRepulsion=Number(document.getElementById('contextRepel').value);
 contextLinkDistance=Number(document.getElementById('contextDistance').value);
 for(const n of contextGraphNodes.values())n.pinned=false;
 contextGraphResume();
}
function contextZoom(multiplier,px=600,py=360){
 const old=contextScale;
 contextScale=Math.max(.25,Math.min(5,old*multiplier));
 contextOffset.x=px-(px-contextOffset.x)*contextScale/old;
 contextOffset.y=py-(py-contextOffset.y)*contextScale/old;
 contextGraphTransform();
}
function contextReset(){
 contextScale=1;contextOffset={x:0,y:0};
 let i=0;
 for(const n of contextGraphNodes.values()){
  const angle=i++*2.399963229728653;
  n.x=600+Math.cos(angle)*(75+50*Math.sqrt(i));
  n.y=350+Math.sin(angle)*(75+50*Math.sqrt(i));
  n.vx=0;n.vy=0;n.pinned=false;
 }
 contextGraphTransform();contextGraphUpdateGeometry();contextGraphResume();
}
function contextSVGPoint(e){
 const p=contextSVG.createSVGPoint();p.x=e.clientX;p.y=e.clientY;
 const matrix=contextSVG.getScreenCTM();
 if(!matrix)return{x:600,y:360};
 return p.matrixTransform(matrix.inverse());
}
function contextWorldPoint(e){
 const p=contextSVGPoint(e);
 return{x:(p.x-contextOffset.x)/contextScale,y:(p.y-contextOffset.y)/contextScale};
}
function contextGraphFocus(n){
 contextScale=1.2;contextOffset={x:600-n.x*contextScale,y:360-n.y*contextScale};
 contextGraphTransform();
}
function contextClosePreview(){document.getElementById('contextGraphPreview').classList.add('hidden');contextSelectedID=null}
function showContextNode(p){
 const preview=document.getElementById('contextGraphPreview');
 const title=document.getElementById('contextDetailTitle');
 const body=document.getElementById('contextDetailBody');
 const open=document.getElementById('contextOpenObsidian');
 const node=p.id?p:{type:p.note?'note':p.source?'folder':p.vault?'vault':'app',ref:p.note||p.source||null,label:p.vault?'Obsidian':p.source?.label||p.note?.title||'Integração',id:''};
 contextSelectedID=node.id;contextSelectedNote=node.type==='note'?node.ref:node.type==='indexed-file'?{vaultPath:node.ref.path}:null;
 title.textContent=((node.type==='folder'||node.type==='indexed-folder')?'◈ ':node.type==='note'||node.type==='indexed-file'?'◇ ':'◉ ')+node.label;
 open.classList.toggle('hidden',!contextSelectedNote||!contextData.vault_selected);
 preview.classList.remove('hidden');
 if(node.type==='vault'){
  body.textContent='Cofre Obsidian conectado: '+contextData.vault_name+'\n\nO Consecom utiliza a pasta Consecom/Contexto e não importa automaticamente outras notas do cofre.';
 }else if(node.type==='app'){
  body.textContent=(node.ref?.detail||'Aplicativo registrado')+'\n\nConexão de ferramenta e importação de histórico são operações distintas.';
 }else if(node.type==='folder'){
  const count=(contextData.notes||[]).filter(n=>n.source_id===node.ref?.id).length;
  body.textContent='Pasta lógica: '+node.label+'\nFonte: '+(node.ref?.origin||'MCP')+'\n'+count+' nota(s) cadastrada(s).\n\nAs fontes aparecem agrupadas por origem. Um nó não comprova que todos os históricos do aplicativo já foram importados.';
 }else if(node.type==='indexed-folder'){
  body.textContent='Pasta do Obsidian: '+node.ref.path+'\n'+node.ref.count+' arquivos Markdown indexados neste grupo.\n\nClique em outras bolinhas para explorar arquivos; o Consecom não abriu o conteúdo das notas.';
 }else if(node.type==='indexed-file'){
  body.textContent='Arquivo Markdown detectado no cofre:\n'+node.ref.path+'\n\nIndexado por nome e localização. O conteúdo da nota não foi importado automaticamente.';
  }else if(node.type==='note'){
  body.textContent='Lendo nota registrada...';
  const id=node.ref?.id;
  if(id)api('contexto/note/read',{id}).then(r=>{
   if(contextSelectedNote?.id===id)body.textContent=(r.result.content||'Nota sem conteúdo').slice(0,24000);
  }).catch(e=>{if(contextSelectedNote?.id===id)body.textContent=e.message});
 }
}
function contextGraphContextMenu(e,node){
 e.preventDefault();contextCloseMenu();
 const menu=document.getElementById('contextGraphMenu');
 const rect=contextCanvas.getBoundingClientRect();
 menu.style.left=Math.max(5,Math.min(rect.width-200,e.clientX-rect.left))+'px';
 menu.style.top=Math.max(5,Math.min(rect.height-150,e.clientY-rect.top))+'px';
 function item(label,action){const b=dom('button',label);b.onclick=()=>{contextCloseMenu();action()};menu.append(b)}
 item('Ver detalhes',()=>showContextNode(node));
 item('Centralizar nó',()=>contextGraphFocus(node));
 if(node.type==='note'&&contextData.vault_selected)item('Abrir no Obsidian',()=>{showContextNode(node);openContextInObsidian()});
 if(node.type==='folder'&&contextData.vault_selected)item('Nova ideia',()=>openContextNote());
 menu.classList.remove('hidden');
}
function contextCloseMenu(){document.getElementById('contextGraphMenu').classList.add('hidden');document.getElementById('contextGraphMenu').replaceChildren()}
contextSVG.addEventListener('pointerdown',e=>{
 if(e.button!==0)return;
 contextCloseMenu();
 const el=e.target.closest('.context-node');
 const point=contextSVGPoint(e);
 if(el){
  const node=contextGraphNodes.get(el.getAttribute('data-node'));
  if(!node)return;
  contextDrag={type:'node',id:node.id,start:point,last:point,moved:false};
  node.pinned=true;el.classList.add('dragging');
 }else{
  contextDrag={type:'pan',start:point,oldX:contextOffset.x,oldY:contextOffset.y,moved:false};
  contextSVG.classList.add('dragging');
 }
 e.preventDefault();contextSVG.setPointerCapture(e.pointerId);
});
contextSVG.addEventListener('pointermove',e=>{
 if(!contextDrag)return;
 const point=contextSVGPoint(e),drag=contextDrag;
 const diff=Math.hypot(point.x-drag.start.x,point.y-drag.start.y);
 if(diff>5)drag.moved=true;
 if(drag.type==='node'){
  const n=contextGraphNodes.get(drag.id);
  if(n){
   n.x+=(point.x-drag.last.x)/contextScale;
   n.y+=(point.y-drag.last.y)/contextScale;
   n.vx=0;n.vy=0;
   contextGraphUpdateGeometry();contextGraphResume();
  }
 }else{
  contextOffset={x:drag.oldX+point.x-drag.start.x,y:drag.oldY+point.y-drag.start.y};
  contextGraphTransform();
 }
 drag.last=point;
});
function contextFinishDrag(e){
 if(!contextDrag)return;
 const drag=contextDrag;contextDrag=null;
 contextSVG.classList.remove('dragging');
 if(drag.type==='node'){
  const n=contextGraphNodes.get(drag.id);
  contextGraphParts.get(drag.id)?.classList.remove('dragging');
  if(n&&!drag.moved)showContextNode(n);
  else contextGraphResume();
 }else if(!drag.moved)contextClosePreview();
 try{if(contextSVG.hasPointerCapture(e.pointerId))contextSVG.releasePointerCapture(e.pointerId)}catch{}
}
contextSVG.addEventListener('pointerup',contextFinishDrag);
contextSVG.addEventListener('pointercancel',contextFinishDrag);
contextSVG.addEventListener('contextmenu',e=>{
 const el=e.target.closest('.context-node');
 if(el){const n=contextGraphNodes.get(el.getAttribute('data-node'));if(n){contextGraphContextMenu(e,n);return}}
 e.preventDefault();contextCloseMenu();
});
contextSVG.addEventListener('wheel',e=>{
 if(activePage!=='context')return;
 e.preventDefault();
 const p=contextSVGPoint(e);
 contextZoom(e.deltaY<0?1.12:1/1.12,p.x,p.y);
},{passive:false});
contextSVG.setAttribute('tabindex','0');
contextSVG.addEventListener('keydown',e=>{
 if(e.target!==contextSVG)return;
 if(e.key==='+'||e.key==='='){e.preventDefault();contextZoom(1.2)}
 if(e.key==='-'){e.preventDefault();contextZoom(1/1.2)}
 const speed=e.shiftKey?60:24;
 if(e.key==='ArrowUp'){contextOffset.y+=speed;e.preventDefault()}
 if(e.key==='ArrowDown'){contextOffset.y-=speed;e.preventDefault()}
 if(e.key==='ArrowLeft'){contextOffset.x+=speed;e.preventDefault()}
 if(e.key==='ArrowRight'){contextOffset.x-=speed;e.preventDefault()}
 contextGraphTransform();
});
document.addEventListener('keydown',e=>{
 if(e.key==='Escape'&&activePage==='context'){
  contextCloseMenu();contextClosePreview();document.getElementById('contextGraphSettings').classList.add('hidden');
 }
});

async 
function contextIndexBadge(index){
 const count=document.getElementById('contextIndexCount');
 const subtitle=document.getElementById('contextIndexSubtitle');
 if(!count||!subtitle)return;
 if(index.status==='ready'){
  count.textContent=(index.total||0).toLocaleString('pt-BR')+' arquivos indexados';
  subtitle.textContent=(index.folders||[]).length+' pastas exibidas · '+(index.complete?'varredura concluída':'índice parcial');
 }else if(index.status==='indexing'){
  count.textContent='Indexando Obsidian...';
  subtitle.textContent='Lendo somente nomes e pastas · segundo plano';
 }else{
  count.textContent='Sem índice disponível';
  subtitle.textContent=index.source||'Conecte o Obsidian';
 }
}
let contextIndexPolling=false;
async function refreshContextIndex(){
 if(contextIndexPolling||activePage!=='context'||!contextData.vault_selected)return;
 contextIndexPolling=true;
 try{
  const result=(await api('contexto/indice')).result;
  const old=contextData.index||{};
  const previous=old.status+':'+old.total+':'+(old.folders||[]).length;
  contextData.index=result||{};
  contextIndexBadge(contextData.index);
  const next=contextData.index.status+':'+contextData.index.total+':'+(contextData.index.folders||[]).length;
  if(previous!==next)drawContext();
 }catch(e){contextIndexBadge({status:'error',source:e.message})}
 finally{contextIndexPolling=false}
}
setInterval(()=>{if(activePage==='context'&&contextData.index?.status==='indexing')refreshContextIndex()},1800);
setInterval(()=>{if(activePage==='context')refreshContextIndex()},90000);
function reloadContext(){
 return (async()=>{
 try{
 const r=(await api('contexto')).result;
 contextData=r.graph||{sources:[],notes:[]};
 contextData.index=r.index||{};
 const select=document.getElementById('contextVaultSelect');
 const previous=select.value;select.replaceChildren();
 const ph=dom('option','Cofres detectados');ph.value='';select.append(ph);
 for(const vault of r.detected_vaults||[]){
  const opt=dom('option',vault.name);opt.value=vault.id;select.append(opt);
 }
 if(previous&&[...select.options].some(o=>o.value===previous))select.value=previous;
 else if(contextData.vault_selected){
  const found=(r.detected_vaults||[]).find(v=>v.name===contextData.vault_name);
  if(found)select.value=found.id;
 }
 mark('contextVaultName',contextData.vault_selected?contextData.vault_name:'Aguardando conexão');
 mark('contextSync',contextData.vault_selected?'· sincronização ativa':r.obsidian_detected?'· detectado':'· não encontrado');
 mark('contextHeroState',contextData.vault_selected?'● Cofre conectado':'○ Sem cofre conectado');
 const btn=document.getElementById('contextAutoVaultBtn');
 btn.textContent=contextData.vault_selected?'✓ Reconectar':'Conectar automaticamente';
 btn.disabled=!r.obsidian_detected;
 await loadContextIntegrations();
 if(contextData.index.status==='indexing')setTimeout(refreshContextIndex,2100);
 }catch(e){toast('Contexto: '+e.message,true)}
 })()
}async function selectContextVault(){
 let id=document.getElementById('contextVaultSelect').value;if(!id){toast('Selecione um cofre existente',true);return}
 if(!confirm('Permitir ao Consecom criar notas apenas na pasta Consecom/Contexto deste cofre Obsidian?'))return;
 try{await api('contexto/vault/select',{id});toast('Cofre vinculado');await reloadContext()}catch(e){toast(e.message,true)}
}
async function loadContextIntegrations(){
 return (async()=>{
 const box=document.getElementById('contextIntegrations');if(!box)return;
 try{
  const payload=(await api('contexto/integracoes')).result;
  window.contextIntegrationItems=payload.integrations||[];
  box.replaceChildren();
  for(const item of window.contextIntegrationItems){
   const card=dom('div',undefined,'integration-card');
   const title=dom('h3',(item.id==='chatgpt'?'◉ ':item.id==='obsidian'?'◆ ':'▣ ')+item.label);
   let state=item.id==='chatgpt'?(item.observed?'MCP em uso':item.ready?'Túnel conectado':item.configured?'Configurado':'Desconectado'):(item.ready?'Conectado':item.observed?'Observado anteriormente':item.configured?'Configurado':item.detected?'Detectado':'Não detectado');
   card.append(title,dom('span',state,'integration-status'),dom('p',item.detail));
   if(item.workspace_hint)card.append(dom('small','Pasta identificada: '+item.workspace_hint,'integration-hint'));
   if(item.supported&&!item.configured){
    const btn=dom('button','Conectar','primary');
    btn.onclick=async()=>{
     if(!confirm('Configurar '+item.label+' no Consecom?'))return;
     btn.disabled=true;btn.textContent='Conectando...';
     try{
      const result=(await api('contexto/integracoes/conectar',{id:item.id})).result;
      toast(item.label+' configurado; conexão MCP será validada quando o cliente responder.');
      await reloadContext();
     }catch(e){toast(e.message,true);btn.disabled=false;btn.textContent='Tentar novamente'}
    };
    card.append(btn);
   }else if(item.workspace_hint&&!item.configured&&!item.supported){
    const index=dom('button','Adicionar ao grafo');
    index.onclick=async()=>{try{await api('contexto/integracoes/indexar',{id:item.id});toast('Fonte registrada; aplicativo MCP ainda não conectado.');await reloadContext()}catch(e){toast(e.message,true)}};
    card.append(index);
   }else if(item.id==='chatgpt'&&!item.ready){
    const btn=dom('button','Configurar túnel');btn.onclick=()=>openModal('connection');card.append(btn);
   }
   box.append(card);
  }
  drawContext();
 }catch(e){box.replaceChildren(dom('p','Falha ao consultar integrações: '+e.message));}
 })()
}
async function connectObsidianAuto(){
 const btn=document.getElementById('contextAutoVaultBtn');
 btn.disabled=true;btn.textContent='Conectando...';
 try{
  const r=(await api('contexto/vault/auto',{})).result;
  toast('Cofre '+r.name+' conectado automaticamente');
  await reloadContext();
 }catch(e){toast('Obsidian: '+e.message,true);btn.disabled=false;btn.textContent='Conectar automaticamente'}
}
function renderContextExplorer(){
 const tree=document.getElementById('contextExplorerTree');if(!tree)return;
 tree.replaceChildren();
 const query=(document.getElementById('contextSearchInput')?.value||'').trim().toLowerCase();
 const sources=contextData.sources||[];
 const notes=contextData.notes||[];
 const integrations=window.contextIntegrationItems||[];
 function folder(text,count,click){
  const btn=dom('button',undefined,'context-folder');
  btn.append(dom('span','▸','folder-glyph'),dom('span',text),dom('span',count,'tree-count'));
  btn.onclick=click;return btn;
 }
 // The ChatGPT tunnel appears first even before its first model handshake.
 const chat=integrations.find(v=>v.id==='chatgpt');
 if(chat&&('chatgpt'.includes(query)||query==='')){
  const row=folder('ChatGPT',chat.observed?'em uso':chat.ready?'túnel pronto':'inativo',()=>{mark('contextDetailTitle','ChatGPT');mark('contextDetailBody',chat.detail)});
  tree.append(row);
 }
 if(contextData.vault_selected&&(!query||contextData.vault_name.toLowerCase().includes(query)||'obsidian'.includes(query))){
  tree.append(folder('Obsidian · '+contextData.vault_name,'cofre',()=>showContextNode({vault:true})));
 }
 for(const source of [...sources].sort((a,b)=>a.label.localeCompare(b.label,'pt-BR'))){
  const children=notes.filter(n=>n.source_id===source.id&&(!query||n.title.toLowerCase().includes(query)));
  if(query&&!source.label.toLowerCase().includes(query)&&children.length===0)continue;
  const wrap=dom('div');
  const row=folder(source.label,children.length+' notas',()=>{wrap.classList.toggle('collapsed');showContextNode({source});});
  wrap.append(row);
  const content=dom('div');
  for(const n of children){
   const btn=dom('button','◇ '+n.title,'context-note-row');
   btn.title=n.title;btn.onclick=()=>showContextNode({note:n});content.append(btn);
  }
  if(!children.length){const placeholder=dom('small','Sem notas neste grupo');placeholder.style.cssText='color:#7799a5;margin:7px 0 10px 27px;display:block';content.append(placeholder)}
  wrap.append(content);tree.append(wrap);
 }
 if(!tree.children.length){tree.append(dom('p','Nenhum resultado para esta busca.','context-meta'))}
}

function openContextInObsidian(){
 if(!contextSelectedNote||!contextData.vault_selected||!contextData.vault_name){toast('Selecione uma nota do cofre conectado',true);return}
 let relative=contextSelectedNote.vaultPath||('Consecom/Contexto/'+contextSelectedNote.filename);
 location.href='obsidian://open?vault='+encodeURIComponent(contextData.vault_name)+'&file='+encodeURIComponent(relative);
}
function contextFormRoot(title){let root=document.getElementById('modalContent');root.replaceChildren();root.append(dom('h2',title));document.getElementById('modalBack').style.display='grid';return root}
function openContextSource(){
 let root=contextFormRoot('Registrar fonte MCP');
 root.append(dom('p','Cria uma fonte no grafo. Clientes como Claude Code e OpenCode também podem se registrar diretamente pela ferramenta MCP.'));
 let name=dom('input');name.placeholder='Ex.: Claude Code, OpenCode, ChatGPT';let btn=dom('button','Registrar fonte','primary');
 btn.onclick=async()=>{try{await api('contexto/source/register',{name:name.value});closeModal();await reloadContext();toast('Fonte registrada')}catch(e){toast(e.message,true)}};
 root.append(name,btn);
}
function openContextNote(){
 if(!contextData.vault_selected){toast('Conecte primeiro um cofre Obsidian',true);return}
 let sources=contextData.sources||[];if(!sources.length){toast('Registre uma fonte antes de criar uma nota',true);return}
 let root=contextFormRoot('Nova ideia');
 let select=dom('select');for(let s of sources){let opt=dom('option',s.label);opt.value=s.id;select.append(opt)}
 let name=dom('input');name.placeholder='Título da ideia';
 let body=dom('textarea');body.placeholder='Contexto, decisão, ideia, tarefa ou resumo autorizado...';body.style.cssText='width:100%;min-height:165px;background:#071729;color:#ecf7ff;border:1px solid #456481;border-radius:9px;padding:11px;margin:7px 0 13px;resize:vertical';
 let btn=dom('button','Salvar no Obsidian','primary');btn.onclick=async()=>{try{await api('contexto/note',{source_id:select.value,title:name.value,content:body.value,links:[]});closeModal();await reloadContext();toast('Nota sincronizada')}catch(e){toast(e.message,true)}};
 root.append(select,name,body,btn);
}

</script></body></html>`
