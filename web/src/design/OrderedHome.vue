<template>
  <main class="canvas-home layout-design-home ordered-home natural-home" :class="{'ordered-editing':editing,'slot-overview-active':overview,'home-semantic-colors':colorStyle==='vivid'}" :data-color-style="colorStyle" data-ordered-home @pointerdown.capture="sizeClickPending=false" @click.capture="eatSizeClick" @click="blankClick">
    <header class="lab-header"><router-link class="lab-brand" :to="target('/')"><span class="canvas-logo">Q<i></i></span><span>Quickstart<small>iStoreOS</small></span></router-link><span class="lab-header-context">{{t('我的网络空间','My network space')}}</span><div class="lab-header-actions"><button @click="toggleTheme" :aria-label="t('切换主题','Toggle theme')">{{theme==='dark'?'☼':'◐'}}</button><details class="lab-scenes"><summary :aria-label="t('设计预览','Design preview')"><span class="direct-preview-long">{{t('设计预览','Design preview')}}</span><span class="direct-preview-short">{{t('预览','Preview')}}</span></summary><div><small>{{t('合成数据 · 刷新恢复示例','Synthetic data · Reload resets')}}</small><label>{{t('布局示例','Layout sample')}}<select aria-label="布局示例" :value="route.query.preset||'normal'" @change="changeSample"><option value="normal">{{t('默认布局','Default layout')}}</option><option value="half">{{t('应用半宽','Half-width apps')}}</option><option value="stacked">{{t('左单右双','Single left, stacked right')}}</option><option value="mirrored">{{t('左双右单','Stacked left, single right')}}</option><option value="apps-devices">{{t('应用与设备同排','Apps beside devices')}}</option><option value="network-stack">{{t('完整连接＋紧凑端口','Connection + compact ports')}}</option><option value="network-stack-mirror">{{t('网络组合镜像','Mirrored network stack')}}</option><option value="network-stack-reverse">{{t('端口在上，连接在下','Ports above connection')}}</option><option value="network-business">{{t('连接与接口独立','Connection & interfaces')}}</option><option value="services">{{t('三列服务','Three services')}}</option></select></label><label>{{t('色彩风格','Colour style')}}<select data-color-select aria-label="色彩风格" :value="colorStyle" @change="changeQuery('palette',$event)"><option value="classic">{{t('原版 · 低饱和','Original · muted')}}</option><option value="vivid">{{t('鲜明 · 推荐','Vivid · recommended')}}</option><option value="bold">{{t('浓彩 · 更鲜艳','Bold · richer')}}</option></select></label><label>{{t('数据场景','Scenario')}}<select aria-label="数据场景" :value="demo.scenario" @change="changeQuery('scenario',$event)"><option v-for="scene in scenes" :key="scene[0]" :value="scene[0]">{{t(scene[1],scene[2])}}</option></select></label><label>FastNet<select aria-label="FastNet 场景" :value="fastnet.scene" @change="setFastnetScene(($event.target as HTMLSelectElement).value)"><option v-for="scene in fastnetScenes" :key="scene.id" :value="scene.id">{{scene.label}}</option></select></label><label>{{t('地址分配方','Address allocator')}}<select aria-label="地址分配方" :value="demo.authority" @change="changeQuery('authority',$event)"><option value="external">{{t('其他主路由','Another router')}}</option><option value="local">{{t('本机','This router')}}</option><option value="unknown">{{t('待确认','Unknown')}}</option></select></label><button @click="toggleLanguage">中文 / English</button></div></details><button ref="editButton" class="lab-edit-button" @click="editing?cancel():begin()" :aria-pressed="editing" :aria-label="t('自定义首页','Customize home')"><StudioGlyph name="tune"/><span>{{t('自定义首页','Customize home')}}</span></button></div></header>
    <div class="lab-intro"><div><span class="lab-eyebrow">{{t('首页概览','HOME OVERVIEW')}}</span><h1>{{t('连接、设备与生活','Your connected home')}}</h1></div><span class="lab-intro-status"><i :class="{ready:connectionState(data.network,errors.network)==='netSuccess'}"></i>{{t('我的网络','My network')}}<small>{{t('设计示例','Design sample')}}</small></span></div>

    <section v-if="editing" class="direct-toolbar" aria-label="首页布局编辑器">
      <span class="direct-edit-label"><i></i>{{t('整理首页','Arrange home')}}</span>
      <button @click="openSlot(null)" data-open-library>＋ {{t('添加模块','Add module')}}</button>
      <button @click="toggleOverview" :aria-pressed="overview" data-overview>{{overview?t('真实页面','Real page'):t('布局总览','Overview')}}</button>
      <button @click="undo" :disabled="!history.length" data-undo>↶ <span>{{t('撤销','Undo')}}</span></button>
      <button @click="openPanel('more')">{{t('更多','More')}} ···</button>
      <div class="direct-toolbar-end"><button @click="cancel" data-cancel>{{t('取消','Cancel')}}</button><button @click="apply" class="primary" data-apply>{{t('完成','Done')}}</button></div>
    </section>
    <DirectEditPanel v-if="panel==='display'&&settingsPlan" :title="name(settingsId)+' · '+t('显示设置','Display settings')" @close="closePanel">
      <PresentationSettings :id="settingsId" :value="settingsValue" @update="settingsValue=$event"/>
      <section class="presentation-result" data-display-preview><strong>{{t('当前实际呈现','Actual presentation')}}</strong><p>{{settingsDescription}}</p><p v-if="settingsPlan.status==='blocked'" role="status">{{issueText(settingsPlan.issue)}}</p><ul v-else class="presentation-impact"></ul><p>{{t('只调整此模块的展示；常用项清单与业务设置不变。','Only the display changes. Favorites and business settings stay intact.')}}</p></section>
      <template #actions><button class="primary" :disabled="settingsPlan.status==='blocked'" @click="adoptSettings" data-adopt-display>{{t('采用设置','Adopt settings')}}</button></template>
    </DirectEditPanel>
    <DirectEditPanel v-if="panel==='module-actions'&&menuModule" :title="name(menuModule)" @close="closePanel"><button class="direct-panel-action" @click="menuModule&&openSettings(menuModule)" data-edit-display>{{t('显示设置','Display settings')}}</button><button class="direct-panel-action" @click="openSlot(pendingSlot)">{{t('替换模块','Replace module')}}</button><button class="direct-panel-action" @click="menuModule&&hideCard(menuModule);closePanel()">{{t('从首页移除','Hide module')}}</button></DirectEditPanel>
    <DirectEditPanel v-if="panel==='adapt'&&pendingPlan" :title="t('放入前调整','Adjust before placing')" @close="closePanel">
      <p class="direct-panel-note">{{t('确认以下调整后一起放入；取消时布局保持不变。','Review the changes before placing. Cancel keeps your layout intact.')}}</p>
      <ul class="presentation-impact" data-adaptation-preview><li v-for="id in changedPreferences(pendingPlan)" :key="id">{{name(id)}} · {{formLabel(preferences[id].mode)}} → {{formLabel(pendingPlan.next.widgetPreferences[id].mode)}}</li><li v-if="pendingPlan.hidden.length">{{t('将隐藏：','Will hide: ')}}{{pendingPlan.hidden.map(name).join('、')}}</li></ul>
      <button v-if="pendingAlternative&&pendingAlternative.status!=='blocked'" @click="pendingPlan=pendingAlternative;pendingAlternative=null" data-adaptation-summary>{{t('改用摘要方案','Use summary alternative')}}</button>
      <p class="presentation-help">{{t('位置和展示设置将一起提交，可一次撤销。','Position and display form one undoable change.')}}</p>
      <template #actions><div class="slot-confirm"><button @click="closePanel()" data-cancel-adaptation>{{t('取消','Cancel')}}</button><button class="primary" @click="confirmAdaptation" data-confirm-adaptation>{{t('采用调整并放入','Adjust and place')}}</button></div></template>
    </DirectEditPanel>
    <DirectEditPanel v-if="panel==='more'" :title="t('预览与恢复','Preview and restore')" @close="closePanel">
      <p class="direct-panel-note">{{t('布局只保留在本次预览，刷新恢复示例。','Preview memory only. Reload resets the sample.')}}</p>
      <button class="direct-panel-action" @click="restoreRecommended();closePanel()" data-reset>{{t('恢复推荐布局','Restore recommended')}}</button>
      <button class="direct-panel-action" @click="addRow();panel='row'" data-add-row>＋ {{t('添加空行','Add empty row')}}</button>
      <button class="direct-panel-action" @click="hideAll();closePanel()">{{t('全部收起','Hide all')}}</button>
      <div class="ordered-width"><label>{{t('内容预览宽度','Content preview width')}}<input type="range" min="292" :max="Math.max(292,availableWidth)" step="1" :value="Math.min(previewWidth||availableWidth,availableWidth)" @input="previewWidth=Number(($event.target as HTMLInputElement).value)" aria-label="内容预览宽度"/><output>{{Math.round(contentWidth)}}px</output></label><button @click="previewWidth=0">{{t('自动宽度','Auto width')}}</button><span>{{t('缩窄不改写你的布局，变宽后恢复','Narrow previews preserve the wide layout')}}</span></div>
    </DirectEditPanel>
    <DirectEditPanel v-if="panel==='targets'&&moving" :title="t('移动到哪里','Choose a destination')" @close="closePanel">
      <p class="direct-panel-note">{{sourceName}} · {{t('选择目标，确认前不会改变布局。','Choose a destination. Nothing changes until confirmed.')}}</p>
      <button v-for="target in moveTargets" :key="targetKey(target)" class="direct-target-option" :data-target-option="targetKey(target)" :aria-disabled="!!targetInfo(target).issue" @click="place(target)"><span>{{targetInfo(target).label}}</span><small v-if="targetInfo(target).issue">{{issueText(targetInfo(target).issue)}}</small></button>
    </DirectEditPanel>
    <div v-if="moving" class="direct-moving-bar" role="status"><span>{{sourceName}} <small>{{t('选择位置','Choose destination')}}</small></span><button @click="panel='targets'">{{t('目标列表','Destinations')}}</button><button @click="cancelMoving">{{t('取消移动','Cancel move')}}</button></div>
    <div v-if="dragPoint" class="direct-drag-ghost" :style="{left:dragPoint.x+'px',top:dragPoint.y+'px'}">⠿ {{sourceName}}<small>{{hoverInfo}}</small></div>
    <DirectEditPanel v-if="panel==='row'" :title="t('这一行的布局','This row’s layout')" @close="closePanel">
      <template v-if="selectedRow">
        <div class="ordered-template-list" aria-label="行布局样式"><button v-for="preset in templates" :key="preset.id" @click="setTemplate(preset.id)" :class="{active:selectedRow.template===preset.id}" :aria-pressed="selectedRow.template===preset.id" :data-template="preset.id"><span class="ordered-miniature" :class="preset.id"><i v-for="(n,index) in preset.slots" :key="index"><b v-for="k in n" :key="k"></b></i></span>{{t(preset.zh,preset.en)}}</button></div>
        <section v-if="requestedTemplate" class="slot-change-preview" data-structure-preview>
          <h3>{{t('布局预览','Layout preview')}} · {{requestedTemplateName}}</h3>
          <p v-if="visibleIds([selectedRow]).length>templateCapacity">{{t('请选择保留的模块；未保留项仅从首页隐藏。','Choose what to keep. Other modules will only be hidden.')}}</p>
          <template v-if="visibleIds([selectedRow]).length>templateCapacity"><label v-for="id in visibleIds([selectedRow])" :key="id" class="slot-retained"><input type="checkbox" :checked="retainedIds.includes(id)" @change="toggleRetained(id)" :data-retain="id"/>{{name(id)}}</label></template>
          <p v-else>{{visibleIds([selectedRow]).map(name).join(' · ')}}</p>
          <p v-if="structureResult.hidden.length">{{t('将隐藏：','Will hide: ')}}{{structureResult.hidden.map(name).join('、')}}</p>
          <p v-if="structureResult.issue" role="status">{{issueText(structureResult.issue)}}</p>
          <p v-if="structureResult.requiresAdaptation" role="status">{{t('采用此布局将同时调整以下形态，可一次撤销。','This layout also changes the following forms in one undoable step.')}}</p>

          <ul v-if="structureResult.requiresAdaptation&&structurePlan" data-structure-adaptations><li v-for="id in changedPreferences(structurePlan)" :key="id">{{name(id)}} · {{formLabel(preferences[id].mode)}} → {{formLabel(structurePlan.next.widgetPreferences[id].mode)}}</li></ul><div class="slot-confirm"><button @click="requestedTemplate=null">{{t('取消更改','Cancel change')}}</button><button :disabled="!!structureResult.issue" @click="confirmStructure" data-confirm-structure class="primary">{{t('采用此布局','Use layout')}}</button></div>
        </section>
        <p class="direct-panel-note" data-natural-height-note>{{t('默认随内容等高。拖动上下分隔线分配空间，拉行底边增加空间。','Content sets the default height. Drag the stack divider to share space, or the row’s bottom edge to add space.')}}</p>
        <button class="direct-panel-action" @click="resetRowSpace" data-reset-row-space>{{t('恢复自动高度与分配','Restore automatic height and allocation')}}</button>
        <div v-if="selectedRow.columns.length>1" class="direct-ratio-presets"><small>{{t('列宽比例','Column proportions')}}</small><button v-for="ratio in widthPresets" :key="ratio.join('/')" @click="setWeights(ratio)" :aria-pressed="selectedRow.columns.map(c=>c.weight).join('/')===ratio.join('/')" :data-width-preset="ratio.join('/')"><i v-for="(weight,index) in ratio" :key="index" :style="{flex:weight}"></i><span>{{ratio.join(' / ')}}</span></button></div>
        <div class="ordered-row-actions"><button @click="hideRow" data-hide-row>{{t('收起整行','Hide row')}}</button></div>
        <p class="ordered-editor-hint">{{t('内容最小尺寸始终保留。调整后可一次撤销。','Content minimums are always preserved. Each adjustment can be undone.')}}</p>
      </template>
    </DirectEditPanel>
    <LayoutNotice :notice="noticeState" :editing="editing" @dismiss="dismissNotice"/>
    <DirectEditPanel v-if="panel==='finish'&&finishPreview" :title="t('完成前整理','Review before finishing')" @close="closePanel">
      <p class="direct-panel-note">{{t('仅整理本次留下的空位，不合并不同行。确认前首页布局不变。','Only empty slots are simplified. Rows are never merged. Nothing is applied until you confirm.')}}</p>
      <div v-for="row in draft" :key="row.id" class="slot-finish-row" :data-finish-row="row.id"><strong>{{t('第','Row ')+(draft.indexOf(row)+1)+t('行','')}}</strong><small>{{rowDescription(row)}} → {{rowDescription(finishPreview.rows.find(r=>r.id===row.id))}}</small></div>
      <section v-for="problem in finishPreview.problems" :key="problem.row" class="slot-change-preview" data-finish-problem>
        <p>{{t('第','Row ')+(draft.findIndex(r=>r.id===problem.row)+1)+t('行：',' · ')}}{{issueText(problem.issue)}}</p>

        <button @click="fixRow(problem.row)" data-fix-row>{{t('定位并修改布局','Locate and fix layout')}}</button>
      </section>
      <div class="slot-confirm"><button @click="closePanel()" data-cancel-finish>{{t('继续编辑','Keep editing')}}</button><button :disabled="!!finishPreview.issue" @click="confirmFinish" data-confirm-finish class="primary">{{t('确认整理并完成','Confirm and finish')}}</button></div>
    </DirectEditPanel>

    <section v-if="editing&&overview" ref="overviewElement" class="slot-overview" :class="{'row-sort-active':rowSorting.active.value}" data-layout-overview @pointerover="trackTarget" @focusin="trackTarget">
      <div class="slot-overview-heading"><h2>{{t('把首页排成你喜欢的样子','Make room for what matters')}}</h2><p role="status">{{moving?hoverInfo:t('拖动行标题排列整行；点选或拖动模块名称调整模块。','Drag a row title to reorder rows; select or drag a module name to move modules.')}}</p></div>
      <div v-if="rowSorting.preview.value" class="row-sort-placeholder" data-row-placeholder :style="{top:rowSorting.preview.value.top+'px',height:rowSorting.preview.value.height+'px'}"><span>{{t('放到这里','Drop here')}} · {{rowSorting.preview.value.to+1}}</span></div>
      <section v-for="(row,ri) in draft" :key="row.id" class="slot-overview-row" :class="{'row-sort-source':rowSorting.preview.value?.id===row.id}" :style="rowSorting.style(row.id)" :data-overview-row="row.id">
        <button v-if="moving" class="slot-overview-gap" :data-placement="'edge:'+row.id" @click="place({kind:'edge',before:row.id})" :aria-label="targetInfo({kind:'edge',before:row.id}).label"><span>＋ {{targetInfo({kind:'edge',before:row.id}).label}}</span></button>
        <header><button class="row-sort-handle" @pointerdown="rowSorting.start($event,row.id)" @click="selectRow(row.id)" @keydown="rowKey($event,row.id,ri)" :data-row-handle="row.id" :aria-label="t('拖动整行','Drag row')+' '+(ri+1)" :title="t('拖动排序；Alt + 上下键或 Home / End','Drag to sort; Alt + arrows or Home / End')"><span aria-hidden="true">⠿</span><strong>{{t('第','Row ')+(ri+1)+t('行','')}}</strong></button><span>{{t('自适应高度','Content height')}} · {{row.columns.map(c=>c.weight).join(' / ')}}</span><details class="row-sort-menu" :data-row-menu="row.id"><summary :aria-label="t('整行操作','Row actions')+' '+(ri+1)">···</summary><div><button @click="sortRow(row.id,ri-1,true)" :disabled="ri===0" data-row-up>{{t('上移','Move up')}}</button><button @click="sortRow(row.id,ri+1,true)" :disabled="ri===draft.length-1" data-row-down>{{t('下移','Move down')}}</button><button @click="sortRow(row.id,0,true)" :disabled="ri===0" data-row-first>{{t('置顶','Move to first')}}</button><button @click="sortRow(row.id,draft.length-1,true)" :disabled="ri===draft.length-1" data-row-last>{{t('置底','Move to last')}}</button></div></details><button @click="selected=row.id;openPanel('row')" :data-select-row="row.id">{{t('布局','Layout')}} ▦</button></header>
        <div class="slot-overview-grid" :class="{stacked:row.columns.some(c=>c.cards.length===2)}" :style="{gridTemplateColumns:row.columns.map(c=>'minmax(0,'+c.weight+'fr)').join(' '),height:row.columns.some(c=>c.cards.length===2)?'184px':'88px'}">
          <div v-for="(column,ci) in row.columns" :key="ci" class="slot-overview-column" :style="{gridTemplateRows:column.cards.length===1?'minmax(0,1fr)':'repeat(2,minmax(0,1fr))'}">
            <div v-for="(id,si) in column.cards" :key="si" class="slot-overview-card" :class="{chosen:selectedCard===id,drop:hoverTarget==='slot:'+row.id+':'+ci+':'+si}">
              <template v-if="id"><button class="slot-overview-name" @pointerdown="startMovePointer($event,{kind:'card',id})" @click="selectCard(row.id,id)" :data-drag-card="id"><StudioGlyph :name="widget(id).glyph"/><span class="overview-module-title" :title="name(id)">{{name(id)}}</span><small class="overview-form-label" :data-overview-form="id">{{actualFormLabel(id)}}</small></button><div class="slot-overview-actions"><button v-if="selectedCard===id" @click="startMoving({kind:'card',id})" :data-move-card="id">{{t('移动','Move')}}</button><button v-else @click="locateReal(id)" :aria-label="t('查看','Preview ')+name(id)">↗</button><button @click="openModuleMenu(id,{row:row.id,column:ci,slot:si})" :data-module-menu="id" :aria-label="t('模块操作','Module actions ')+name(id)">···</button></div></template>
              <button v-else class="slot-overview-empty" :data-overview-empty="'slot:'+row.id+':'+ci+':'+si" @click="openSlot({kind:'slot',row:row.id,column:ci,slot:si})">＋ {{t('添加','Add')}}</button>
              <button v-if="moving" class="slot-overview-drop" :data-placement="'slot:'+row.id+':'+ci+':'+si" :aria-label="targetInfo({kind:'slot',row:row.id,column:ci,slot:si}).label" :aria-disabled="!!targetInfo({kind:'slot',row:row.id,column:ci,slot:si}).issue" @click="place({kind:'slot',row:row.id,column:ci,slot:si})"><span>{{id?name(id):t('空位','Empty')}}</span><small>{{targetInfo({kind:'slot',row:row.id,column:ci,slot:si}).issue?issueText(targetInfo({kind:'slot',row:row.id,column:ci,slot:si}).issue):t('放入此处','Place here')}}</small></button>
            </div>
          </div>
          <template v-if="moving"><button v-for="index in row.columns.length+1" :key="index" class="slot-overview-insert" :style="{left:overviewInsertion(row,index-1)}" :data-placement="'beside:'+row.id+':'+(index-1)" :aria-label="targetInfo({kind:'beside',row:row.id,index:index-1}).label" :aria-disabled="!!targetInfo({kind:'beside',row:row.id,index:index-1}).issue" @click="place({kind:'beside',row:row.id,index:index-1})"><span>＋</span></button></template>
        </div>
      </section>
      <button v-if="moving" class="slot-overview-end" data-placement="edge:end" @click="place({kind:'edge',before:null})">＋ {{t('放到最后','Place at end')}}</button>
      <button v-if="!draft.length" class="slot-overview-end" @click="addRow()">＋ {{t('添加空行','Add empty row')}}</button>
    </section>
    <div class="lab-available" :class="{'slot-parked':overview}" ref="availableElement"><div class="lab-content ordered-content" ref="contentElement" :style="{maxWidth:previewWidth?previewWidth+'px':undefined}">
      <section v-for="row in actualRows" :key="row.id" class="ordered-row" :class="{selected:editing&&row.source===selected}" :style="rowStyle(row)" :data-ordered-row="row.id" :data-row-source="row.source">
        <button v-if="moving&&isFirstFragment(row)" class="direct-gap-target" :data-placement="'edge:'+row.source" @click="place({kind:'edge',before:row.source})">＋ {{t('独占新行 / 整行移到这里','New row / move row here')}}</button>
        <div v-if="editing" class="direct-row-tools"><button @click="selected=row.source;openPanel('row')" :aria-label="t('第','Row ')+(draft.findIndex(r=>r.id===row.source)+1)+t('行布局',' layout')" :data-select-row="row.source">{{t('布局','Layout')}} ▦</button></div>
        <div v-for="(column,ci) in row.columns" :key="column.source" class="ordered-column" :style="sizedColumnStyle(row,column,ci)" :data-ordered-column="ci">
          <div v-for="(id,si) in column.cards" :key="id||'empty-'+si" class="ordered-card" :class="{drop:hoverTarget===slotKey(row,column,si),chosen:selectedCard===id}" :data-ordered-card="id||undefined">
            <template v-if="id">
              <div v-if="editing" class="ordered-card-controls slot-card-tools"><button @click="openSettings(id)" :aria-label="name(id)+t('显示设置',' display settings')" :data-display-settings="id">⚙</button><button @click="openSlot(slotTarget(row,column,si))" :aria-label="t('替换','Replace ')+name(id)" :data-replace-module="id">{{t('替换','Replace')}}</button><button @click="hideCard(id)" :aria-label="t('从首页移除','Hide ')+name(id)" :data-hide-module="id">×</button></div>
              <LayoutPreviewCard :role="column.compact?'stacked':'ordinary'" :chart-stretch="chartStretches(row)" :inert="editing?'':undefined" :id="id" :preference="renderPreferences[id]" :data="data" :errors="errors" :devices="devices" :theme="theme" :natural="row.natural" contained @navigate="go" @device="openDevice" @retry="retry"/>
            </template>
            <button v-else class="direct-empty-slot" @click="openSlot(slotTarget(row,column,si))" :data-empty-slot="slotKey(row,column,si)">＋ {{t('添加模块','Add module')}}</button>
            <button v-if="moving" class="direct-placement" :class="{invalid:targetInfo(slotTarget(row,column,si)).issue}" :data-placement="slotKey(row,column,si)" :aria-label="targetInfo(slotTarget(row,column,si)).label" :aria-disabled="!!targetInfo(slotTarget(row,column,si)).issue" @click="place(slotTarget(row,column,si))"><span>{{targetInfo(slotTarget(row,column,si)).label}}<small v-if="targetInfo(slotTarget(row,column,si)).issue">{{issueText(targetInfo(slotTarget(row,column,si)).issue)}}</small></span></button>
          </div>
          <button v-if="sizeControls(row)&&column.cards.length===2" class="row-stack-grip" role="separator" aria-orientation="horizontal" :aria-label="t('调整上下空间','Adjust stacked card space')" :aria-valuenow="Math.round(stackSpace(row,ci).upper)" :aria-valuemin="Math.round(stackSpace(row,ci).min)" :aria-valuemax="Math.round(stackSpace(row,ci).max)" :style="{top:stackBoundary(row,ci)+'px'}" @pointerdown="startStack($event,row,ci)" @keydown.up.prevent="nudgeStack(row,ci,-16)" @keydown.down.prevent="nudgeStack(row,ci,16)" :title="t('拖动分配上下空间；↑ ↓ 微调','Drag to share space; ↑ ↓ to adjust')" data-stack-grip><span>•••</span></button>
        </div>
        <template v-if="sizeControls(row)&&row.columns.length>1"><button v-for="(_,index) in row.columns.slice(0,-1)" :key="index" class="ordered-column-handle" role="separator" tabindex="0" aria-orientation="vertical" :aria-valuenow="row.columns[index].weight" :aria-valuemin="widthBounds(row,index)[0]" :aria-valuemax="widthBounds(row,index)[1]" :style="{left:'clamp(9px,'+divider(row,index)+'px,calc(100% - 9px))'}" :aria-label="t('调整列宽','Adjust column widths')+' '+(index+1)" @pointerdown="startWidth($event,row,index)" @keydown.left.prevent="nudgeWidth(row,index,-5)" @keydown.right.prevent="nudgeWidth(row,index,5)"><span>⋮</span><output v-if="resizingRow===row.source">{{row.columns.map(c=>c.weight).join(' / ')}}</output></button></template>
        <button v-if="sizeControls(row)" class="row-bottom-grip" role="separator" aria-orientation="horizontal" :aria-label="t('调整整行空间','Adjust row space')" :aria-valuenow="Math.round(rowSpace(row))" :aria-valuemin="naturalSizing.sizes.value[row.id]?.height||0" :aria-valuemax="Math.max(ROW_SPACE_LIMIT,naturalSizing.sizes.value[row.id]?.height||0)" @pointerdown="startRowSpace($event,row)" @keydown.up.prevent="nudgeRowSpace(row,-32)" @keydown.down.prevent="nudgeRowSpace(row,32)" :title="t('拖动行底边增加空间；↑ ↓ 微调','Drag the bottom edge for more room; ↑ ↓ to adjust')" data-row-space-grip><span>•••</span></button>
        <span v-if="sizeControls(row)&&sizeHint" class="row-size-hint" role="status" data-size-hint>{{sizeHint}}</span>
      </section>
      <button v-if="moving" class="direct-last-target" data-placement="edge:end" @click="place({kind:'edge',before:null})">＋ {{t('放到最后，独占新行 / 整行移到最后','New row at end / move row to end')}}</button>
      <section v-if="!actualRows.length" class="ordered-empty"><StudioGlyph name="grid"/><h2>{{t('首页，由你安排','A home of your own')}}</h2><p>{{t('添加常看的模块，也可以随时恢复推荐布局。','Add the things you check most, or restore the recommended layout.')}}</p><button @click="editing?openSlot(null):begin()">{{t('添加模块','Add modules')}}</button></section>
    </div></div>
    <footer class="lab-footer"><span>Quickstart <i>/</i> {{t('有序布局 · 设计预览','Ordered layout · Design preview')}}</span><HomeApplications collapsed :fallback-entries="quickEntrances" @navigate="go"/></footer>
    <DirectEditPanel v-if="panel==='slot'" :title="pendingSlot?t('选择槽位模块','Choose module'):t('添加一行模块','Add module in a new row')" @close="closePanel">
      <p class="direct-panel-note">{{t('选择后先预览；已有模块会移到这里，原位置留空。','Preview before confirming. Existing modules move here and leave their old slot empty.')}}</p>
      <div class="slot-module-list"><button v-for="item in widgetDefinitions" :key="item.id" @click="choice=item.id;choiceSummary=false" :disabled="!choiceAvailable(item.id)" :aria-pressed="choice===item.id" :data-choice="item.id" :data-fit="choiceStatus(item.id)"><StudioGlyph :name="item.glyph"/><span>{{name(item.id)}}<small>{{choiceStatus(item.id)==='blocked'?issueText(choicePlan(item.id).issue):choiceStatus(item.id)==='needs-confirmation'?t('调整后可放入 · ','Adjust to fit · ')+sourceLabel(item.id):sourceLabel(item.id)}}</small></span><span>{{originOf(draft,item.id)?'↗':'＋'}}</span></button></div>
      <template #actions><section v-if="choice" class="slot-change-preview" data-choice-preview>
        <h3>{{name(choice)}} → {{pendingSlot?slotLabel(pendingSlot):t('页面末尾新行','New row at end')}}</h3>
        <p v-if="originOf(draft,choice)">{{t('将从原位置移到这里，不创建副本。','Moves from its current position; no duplicate.')}}</p>
        <p v-if="choiceResult(choice).hidden.length">{{choiceResult(choice).hidden.map(name).join('、')}}{{t('将从首页隐藏，设置保留。',' will be hidden. Settings are kept.')}}</p>
        <ul class="presentation-impact"><li v-for="id in selectedChoicePlan?changedPreferences(selectedChoicePlan):[]" :key="id">{{name(id)}} → {{formLabel(selectedChoicePlan!.next.widgetPreferences[id].mode)}}</li></ul>
        <button v-if="choiceAlternative(choice)&&choiceAlternative(choice)!.status!=='blocked'" @click="choiceSummary=!choiceSummary" :aria-pressed="choiceSummary" data-placement-summary>{{choiceSummary?t('保持原形态','Keep current form'):t('选择摘要方案','Choose summary alternative')}}</button>
        <button class="primary" :disabled="selectedChoicePlan?.status==='blocked'" @click="confirmChoice" data-confirm-choice>{{selectedChoicePlan?.status==='needs-confirmation'||selectedChoicePlan&&changedPreferences(selectedChoicePlan).length?t('放入并采用调整','Place with adjustments'):t('确认放置','Place module')}}</button>
      </section><div v-else class="slot-confirm"><button v-if="pendingSlot&&at(draft,pendingSlot)" @click="removePending" data-remove-pending>{{t('从首页移除','Hide module')}}</button><button @click="closePanel()">{{t('返回首页','Back to home')}}</button></div></template>
    </DirectEditPanel>
    <NativeBusinessHost/>
  </main>
</template>
<script setup lang="ts">
import {computed,nextTick,onMounted,onUnmounted,ref,watch} from 'vue'
import {fastnet,fastnetScenes,setFastnetScene} from './fastnetModel'
import LayoutNotice from '../home/LayoutNotice.vue'
import {useLayoutNotice} from '../home/useLayoutNotice'
import {ROW_SPACE_LIMIT,clamp,splitSpace} from './rowSizing'
import {useNaturalSizing} from './useNaturalSizing'
import {useRoute,useRouter} from 'vue-router'
import StudioGlyph from './StudioGlyph.vue'
import DirectEditPanel from './DirectEditPanel.vue'
import PresentationSettings from './PresentationSettings.vue'
import {documentOf,preferenceSession,planDocument,applyPlan,validateDisplayRows,fingerprint,summaryAlternative,type HomeDocument,type LayoutContext,type DisplayPlan} from './homeDocument'
import {defaultPreferences,placementMinimum,resolvePresentation,columnWidths,presentationHeight,type WidgetPreference} from './widgetPresentation'
import {placement,targets,targetKey,type MoveSource,type MoveTarget} from './overviewPlacement'
import LayoutPreviewCard from './LayoutPreviewCard.vue'
import CompactPreviewCard from './CompactPreviewCard.vue'
import HomeApplications from './HomeApplications.vue'
import {useStudioModel} from './studioModel'
import NativeBusinessHost from './NativeBusinessHost.vue'
import {useNativeBusinessReads} from './useNativeBusinessReads'
import {provide} from 'vue'
import {demo} from './fixtures'
import {connectionState} from './networkBusiness'
import {quickEntrances} from './canvasLayout'
import {homeApplicationSession} from './homeApplications'
import {getTheme} from '../utils/theme'
import {assignSlot,at,originOf,reshape,changeSize,moveToNewRow,moveRowTo,validate,normalize,type Normalized,type Slot,type LayoutResult} from './slotLayout'
import {useRowSorting} from './useRowSorting'
import {widgetDefinitions,widget,type WidgetId} from './rowLayout'
import {templates,clone,naturalRows,compactIds,orderedSession,recommended,makeRow,visibleIds,layoutIssue,rowMinimum,changeTemplate,renderRows,type Template,type OrderedRow,type ActualRow,type ActualColumn} from './orderedLayout'
const route=useRoute(),router=useRouter(),english=window.vue_lang==='en',t=(zh:string,en:string)=>english?en:zh,name=(id:WidgetId)=>t(widget(id).zh,widget(id).en)
orderedSession.value=naturalRows(orderedSession.value)
const {noticeState,notice,showNotice,dismissNotice}=useLayoutNotice()
const editing=ref(false),draft=ref(clone(orderedSession.value)),preferences=ref(clone(preferenceSession.value)),revision=ref(0),history=ref<HomeDocument[]>([]),selected=ref(draft.value[0]?.id||''),previewWidth=ref(0),theme=ref(getTheme())
const layout=computed(()=>editing.value?draft.value:orderedSession.value),visible=computed(()=>visibleIds(layout.value)),hidden=computed(()=>widgetDefinitions.filter(w=>!visible.value.includes(w.id)))
const selectedIndex=computed(()=>draft.value.findIndex(r=>r.id===selected.value)),selectedRow=computed(()=>draft.value[selectedIndex.value])
const widthPresets=computed(()=>{const values=selectedRow.value?.columns.length===3?[[34,33,33],[50,25,25],[25,50,25]]:[[50,50],[60,40],[65,35],[40,60]],current=selectedRow.value?.columns.map(c=>c.weight)||[];return values.some(v=>v.join('/')===current.join('/'))?values:[current,...values]})
const readerLayout=computed(()=>visible.value.map(id=>({id,enabled:true})))
const {data,errors,devices,retry}=useStudioModel(readerLayout,['traffic'],true,true)
useNativeBusinessReads(data,errors)
provide('quickstart-native-business',true)
const availableElement=ref<HTMLElement|null>(null),contentElement=ref<HTMLElement|null>(null),overviewElement=ref<HTMLElement|null>(null),availableWidth=ref(1160),contentWidth=ref(1160)
const actualRows=computed(()=>renderRows(previewRows.value,contentWidth.value,homeApplicationSession.favorites.length,editing.value,renderPreferences.value))
const editButton=ref<HTMLButtonElement|null>(null),library=ref(false),libraryElement=ref<HTMLElement|null>(null)
const panel=ref(''),selectedCard=ref<WidgetId|null>(null)
const pendingSlot=ref<Extract<MoveTarget,{kind:'slot'}>|null>(null),resizingRow=ref('')
const choice=ref<WidgetId|null>(null),requestedTemplate=ref<Template|null>(null),retainedIds=ref<WidgetId[]>([])
const finishPreview=ref<Normalized|null>(null)
const overview=ref(false)
const pendingPlan=ref<DisplayPlan|null>(null),pendingAlternative=ref<DisplayPlan|null>(null),choiceSummary=ref(false),choiceBase=ref<HomeDocument|null>(null),choiceContext=ref<LayoutContext|null>(null)
const menuModule=ref<WidgetId|null>(null),settingsId=ref<WidgetId>('applications'),settingsValue=ref<WidgetPreference>({mode:'auto',iconRows:0}),settingsBase=ref<HomeDocument|null>(null),settingsContext=ref<LayoutContext|null>(null)
const settingsPlan=computed(()=>{if(!settingsBase.value||!settingsContext.value)return null;const prefs=clone(settingsBase.value.widgetPreferences);prefs[settingsId.value]=clone(settingsValue.value);return planDocument(settingsBase.value,{rows:settingsBase.value.rows,issue:'',hidden:[],changed:false},settingsContext.value,prefs)})
const previewRows=computed(()=>editing.value&&panel.value==='display'&&settingsPlan.value?.status!=='blocked'&&settingsPlan.value?settingsPlan.value.next.rows:layout.value)
const renderPreferences=computed(()=>editing.value&&panel.value==='display'&&settingsPlan.value?.status!=='blocked'&&settingsPlan.value?settingsPlan.value.next.widgetPreferences:editing.value?preferences.value:preferenceSession.value)
const naturalSizing=useNaturalSizing(contentElement,actualRows,()=>[actualRows.value,renderPreferences.value,data,errors,overview.value],()=>overview.value)
const sizeHint=ref('')
function logicalRow(row:ActualRow){return previewRows.value.find(r=>r.id===row.source)!}
function intactRow(row:ActualRow){const source=logicalRow(row);return contentWidth.value>=650&&source&&row.columns.length===source.columns.length&&row.columns.every(c=>c.cards.length===source.columns[c.source].cards.length)}
function sizeControls(row:ActualRow){return editing.value&&!moving.value&&!overview.value&&row.source===selected.value&&intactRow(row)}
function hasRowSpace(row:ActualRow){const source=logicalRow(row);return intactRow(row)&&(source.minHeight!==undefined||source.columns.some(c=>c.upperShare!==undefined))}
function rowSpace(row:ActualRow){return Math.max(naturalSizing.sizes.value[row.id]?.height||row.chartHeight||row.height,intactRow(row)?logicalRow(row).minHeight||0:0)}
function stackSpace(row:ActualRow,ci:number){const minima=naturalSizing.sizes.value[row.id]?.columns[ci]||[148,148],column=row.columns[ci],share=logicalRow(row).columns[column.source].upperShare,available=rowSpace(row)-18,stretch=column.cards.map(id=>id&&resolvePresentation(id,renderPreferences.value[id],{width:1000,height:100000,role:'stacked',favoriteCount:4,natural:true}).stretch),automatic=stretch[0]?available-minima[1]:stretch[1]?minima[0]:minima[0]+Math.max(0,available-minima[0]-minima[1])/2;return splitSpace(rowSpace(row),minima,share??automatic/available*100)}
function rowElement(row:ActualRow){return contentElement.value?.querySelector<HTMLElement>('[data-ordered-row="'+row.id+'"]')}
function stackBoundary(row:ActualRow,ci:number){return stackSpace(row,ci).upper+9}
function sizedColumnStyle(row:ActualRow,column:ActualColumn,ci:number){if(intactRow(row)&&column.cards.length===2&&logicalRow(row).columns[column.source].upperShare!==undefined&&naturalSizing.sizes.value[row.id]){const space=stackSpace(row,ci);return {gridTemplateRows:space.upper+'px '+space.lower+'px'}}return columnStyle(column)}
function resetRowSpace(){interruptMove();mutate(rows=>{const row=rows.find(r=>r.id===selected.value);if(row){delete row.minHeight;row.columns.forEach(c=>delete c.upperShare)}return rows});sizeHint.value=''}
function layoutContext():LayoutContext{return {width:Math.max(240,contentWidth.value),favoriteCount:homeApplicationSession.favorites.length,favoritesKey:homeApplicationSession.favorites.join('|')}}
function homeDraft(){return documentOf(draft.value,preferences.value,revision.value)}
const settingsDescription=computed(()=>{if(!settingsPlan.value||!settingsContext.value)return '';const rows=settingsPlan.value.next.rows,row=rows.find(r=>visibleIds([r]).includes(settingsId.value));if(!row)return '';const ci=row.columns.findIndex(c=>c.cards.includes(settingsId.value)),width=columnWidths(row,settingsContext.value.width,settingsPlan.value.next.widgetPreferences)[ci],r=resolvePresentation(settingsId.value,settingsValue.value,{width,height:100000,role:row.columns[ci].cards.length===2?'stacked':'ordinary',favoriteCount:settingsContext.value.favoriteCount,mobile:settingsContext.value.width<650,natural:true});return (settingsId.value==='applications'?r.columns+t('列 × ',' columns × ')+r.rows+t('行',' rows')+' · ':'')+Math.round(width)+t('px 宽 · 自适应高度','px wide · Content height')})
async function openSettings(id:WidgetId){openPanel('display');settingsId.value=id;settingsValue.value=clone(preferences.value[id]);settingsBase.value=homeDraft();settingsContext.value=layoutContext();overview.value=false;await nextTick();contentElement.value?.querySelector<HTMLElement>(`[data-display-settings="${id}"]`)?.scrollIntoView({block:'center'})}
function openModuleMenu(id:WidgetId,target:Slot){pendingSlot.value={kind:'slot',...target};menuModule.value=id;openPanel('module-actions')}
function adoptDocument(next:HomeDocument){const before=homeDraft();if(fingerprint(before)===fingerprint(next))return false;history.value=[...history.value.slice(-29),before];draft.value=clone(next.rows);preferences.value=clone(next.widgetPreferences);revision.value++;if(!next.rows.some(r=>r.id===selected.value))selected.value=next.rows[0]?.id||'';notice.value='';return true}
function adoptSettings(){if(!settingsPlan.value)return;const result=applyPlan(homeDraft(),settingsPlan.value,layoutContext());if(result.issue){warn(result.issue);return}adoptDocument(result.document);closePanel()}
const formLabel=(form:string)=>({auto:t('自动推荐','Automatic'),standard:t('标准','Standard'),summary:t('摘要','Summary'),grid:t('图标网格','Icon grid'),strip:t('启动条','Launch strip')}[form]||form)
function changedPreferences(plan:DisplayPlan){return widgetDefinitions.map(w=>w.id).filter(id=>JSON.stringify(preferences.value[id])!==JSON.stringify(plan.next.widgetPreferences[id]))}
function choosePlan(raw:LayoutResult,id?:WidgetId){const direct=planDocument(homeDraft(),raw,layoutContext()),alternative=summaryAlternative(homeDraft(),raw,layoutContext(),id);return {direct,alternative}}
function requestResult(raw:LayoutResult,id?:WidgetId){const {direct,alternative}=choosePlan(raw,id);if(direct.status==='blocked'&&(!alternative||alternative.status==='blocked')){warn(direct.issue);return false}if(direct.status==='ready')return adoptDocument(direct.next);const plan=direct.status==='blocked'?alternative!:direct;openPanel('adapt');pendingPlan.value=plan;pendingAlternative.value=direct.status==='blocked'?null:alternative;return false}
function confirmAdaptation(){if(!pendingPlan.value)return;const result=applyPlan(homeDraft(),pendingPlan.value,layoutContext());if(result.issue){warn(result.issue);return}adoptDocument(result.document);closePanel()}
function normalizeRows(rows:OrderedRow[]){return normalize(rows,()=>true,value=>validateDisplayRows(value,preferences.value,layoutContext()))}
const rowSorting=useRowSorting({rows:()=>draft.value,root:()=>overviewElement.value,enabled:()=>editing.value&&overview.value&&!panel.value,beforeStart:()=>{interruptMove();selectedCard.value=null;overviewElement.value?.querySelectorAll('details[open]').forEach(e=>e.removeAttribute('open'))},commit:(id,to)=>void sortRow(id,to),announce:(from,to,cancelled)=>{notice.value=cancelled?t('已取消排序，布局未改变。','Sort cancelled. Layout unchanged.'):t('第','Row ')+(from+1)+t('行 → 第',' → position ')+(to+1)+t('位，松手放置。','; release to place.')}})
function selectRow(id:string){if(rowSorting.consumeClick())return;selected.value=id;selectedCard.value=null}
async function sortRow(id:string,to:number,scroll=false){interruptMove();overviewElement.value?.querySelectorAll('details[open]').forEach(e=>e.removeAttribute('open'));const remaining=draft.value.filter(r=>r.id!==id),from=draft.value.findIndex(r=>r.id===id);if(from<0||to<0||to>remaining.length)return;const result=moveRowTo(draft.value,id,remaining[to]?.id||null);const changed=commitResult(result);selected.value=id;notice.value=changed?t('整行位置已调整，可撤销。','Row reordered. Undo is available.'):t('位置未改变。','Position unchanged.');await nextTick();const handle=overviewElement.value?.querySelector<HTMLElement>(`[data-row-handle="${id}"]`);handle?.focus({preventScroll:true});if(scroll)handle?.scrollIntoView({block:'nearest'})}
function rowKey(event:KeyboardEvent,id:string,index:number){if(!event.altKey)return;const to=event.key==='ArrowUp'?index-1:event.key==='ArrowDown'?index+1:event.key==='Home'?0:event.key==='End'?draft.value.length-1:undefined;if(to===undefined)return;event.preventDefault();void sortRow(id,to,true)}
function toggleOverview(){interruptMove();closePanel();overview.value=!overview.value;window.scrollTo({top:0})}
async function locateReal(id:WidgetId){interruptMove();overview.value=false;selectedCard.value=id;selected.value=originOf(draft.value,id)?.row||'';await nextTick();contentElement.value?.querySelector<HTMLElement>(`[data-replace-module="${id}"]`)?.scrollIntoView({block:'center'})}
function overviewInsertion(row:OrderedRow,index:number){if(!index)return '0%';if(index===row.columns.length)return '100%';return row.columns.slice(0,index).reduce((n,c)=>n+c.weight,0)+'%'}
function rowDescription(row?:OrderedRow){if(!row)return t('移除空行','Remove empty row');const item=templates.find(t=>t.id===row.template)!;return t(item.zh,item.en)+' · '+visibleIds([row]).map(name).join(' / ')+' · '+t('自适应高度','Content height')}

function confirmFinish(){if(finishPreview.value&&!finishPreview.value.issue)finishApply(finishPreview.value.rows)}
async function fixRow(id:string){selected.value=id;openPanel('row');await nextTick();contentElement.value?.querySelector<HTMLElement>(`[data-row-source="${id}"]`)?.scrollIntoView({block:'center'})}
const templateCapacity=computed(()=>templates.find(t=>t.id===requestedTemplate.value)?.slots.reduce((a,b)=>a+b,0)||0)
const requestedTemplateName=computed(()=>{const item=templates.find(v=>v.id===requestedTemplate.value);return item?t(item.zh,item.en):''})
const structureResult=computed(()=>{
 if(!requestedTemplate.value)return {rows:draft.value,issue:'',changed:false,hidden:[]} as LayoutResult
 const raw=reshape(draft.value,selected.value,requestedTemplate.value,visibleIds([selectedRow.value]).length>templateCapacity.value?retainedIds.value:undefined)
 if(raw.issue&&!raw.issue.startsWith('height:'))return raw
 const rows=naturalRows(raw.candidateRows||raw.rows),check=validateDisplayRows(rows,preferences.value,layoutContext())
 if(check.issue){const alternative=summaryAlternative(homeDraft(),raw,layoutContext());return {...raw,issue:alternative&&alternative.status!=='blocked'&&!check.issue.startsWith('height:')?'':check.issue,requiresAdaptation:!!alternative&&alternative.status!=='blocked'&&!check.issue.startsWith('height:'),requiredHeight:check.requiredHeight,problemRow:check.row}}
 return {...raw,rows,candidateRows:undefined,issue:'',requiredHeight:undefined,changed:JSON.stringify(rows)!==JSON.stringify(draft.value)}
})
function slotLabel(s:Slot){const index=draft.value.findIndex(r=>r.id===s.row);return t('第','Row ')+(index+1)+t('行 · 第',' · Column ')+(s.column+1)+(draft.value[index]?.columns[s.column].cards.length===2?(s.slot===0?t('列上方',' top'):t('列下方',' bottom')):t('列',''))}
function sourceLabel(id:WidgetId){const source=originOf(draft.value,id);return source?t('已在首页 · 移到这里 · ','On home · Move here · ')+slotLabel(source):t('未展示 · 添加','Hidden · Add')}
function choiceResult(id:WidgetId){const rows=choiceBase.value?.rows||draft.value;return pendingSlot.value?.kind==='slot'?assignSlot(rows,pendingSlot.value,id):moveToNewRow(rows,id,null)}
function choicePlan(id:WidgetId){return planDocument(choiceBase.value||homeDraft(),choiceResult(id),choiceContext.value||layoutContext())}
function choiceAlternative(id:WidgetId){return summaryAlternative(choiceBase.value||homeDraft(),choiceResult(id),choiceContext.value||layoutContext(),id)}
function choiceAvailable(id:WidgetId){return choicePlan(id).status!=='blocked'||!!choiceAlternative(id)&&choiceAlternative(id)!.status!=='blocked'}
function choiceStatus(id:WidgetId){const p=choicePlan(id);return p.status==='blocked'&&choiceAvailable(id)?'needs-confirmation':p.status}
const selectedChoicePlan=computed(()=>!choice.value?null:choiceSummary.value||choicePlan(choice.value).status==='blocked'?choiceAlternative(choice.value):choicePlan(choice.value))
function openSlot(target:MoveTarget|null){interruptMove();pendingSlot.value=target?.kind==='slot'?target:null;choice.value=null;choiceSummary.value=false;openPanel('slot');choiceBase.value=homeDraft();choiceContext.value=layoutContext()}
function confirmChoice(){const plan=selectedChoicePlan.value;if(!plan)return;const result=applyPlan(homeDraft(),plan,layoutContext());if(result.issue){warn(result.issue);return}adoptDocument(result.document);closePanel()}
function removePending(){const id=pendingSlot.value?at(draft.value,pendingSlot.value):null;closePanel();if(id)void hideCard(id)}
function commitResult(result:LayoutResult){return requestResult(result)}
function toggleRetained(id:WidgetId){retainedIds.value=retainedIds.value.includes(id)?retainedIds.value.filter(v=>v!==id):[...retainedIds.value,id]}
const structurePlan=computed(()=>{const plans=choosePlan(structureResult.value);return plans.direct.status==='blocked'?plans.alternative||plans.direct:plans.direct})
function confirmStructure(){const plan=structurePlan.value;if(!plan||plan.status==='blocked')return;const applied=applyPlan(homeDraft(),plan,layoutContext());if(applied.issue){warn(applied.issue);return}adoptDocument(applied.document);requestedTemplate.value=null}
function restoreRecommended(){const plan=planDocument(homeDraft(),{rows:recommended(),issue:'',hidden:[],changed:false},layoutContext(),defaultPreferences());if(plan.status!=='blocked')adoptDocument(plan.next)}
function actualFormLabel(id:WidgetId){const row=actualRows.value.find(r=>r.columns.some(c=>c.cards.includes(id))),column=row?.columns.find(c=>c.cards.includes(id));if(!row||!column)return '';const width=columnWidths({id:row.id,template:'single',height:row.height,columns:row.columns},contentWidth.value,renderPreferences.value)[row.columns.indexOf(column)],resolved=resolvePresentation(id,renderPreferences.value[id],{width,height:100000,role:column.compact?'stacked':'ordinary',favoriteCount:4,natural:true});return formLabel(resolved.id)}
function chooseModule(id:WidgetId){const target=pendingSlot.value;closeLibrary();startMoving({kind:'module',id});if(target)void place(target)}
async function locate(id:WidgetId){closeLibrary();selectedCard.value=id;selected.value=draft.value.find(r=>visibleIds([r]).includes(id))?.id||'';await nextTick();const element=contentElement.value?.querySelector<HTMLElement>(`[data-drag-card="${id}"]`);element?.scrollIntoView({block:'center',behavior:'auto'});element?.focus({preventScroll:true})}
function openSize(row:string){if(suppressClick)return;selected.value=row;openPanel('row')}
function openPanel(value:string){interruptMove();panel.value=value;selectedCard.value=null;requestedTemplate.value=null}
function closePanel(){const id=panel.value==='display'?settingsId.value:null;panel.value='';settingsBase.value=null;settingsContext.value=null;pendingPlan.value=null;pendingAlternative.value=null;choiceBase.value=null;choiceContext.value=null;if(id)void nextTick(()=>contentElement.value?.querySelector<HTMLElement>(`[data-display-settings="${id}"]`)?.focus({preventScroll:true}))}
function blankClick(event:MouseEvent){naturalSizing.schedule();if(suppressClick)return;if(!(event.target as HTMLElement).closest('button,input,select,a,summary,.direct-panel,.ordered-library')){const next=editing.value?(event.target as HTMLElement).closest<HTMLElement>('[data-row-source]')?.dataset.rowSource||'':'';if(next!==selected.value)sizeHint.value='';selected.value=next;selectedCard.value=null}}
function begin(){draft.value=clone(orderedSession.value);preferences.value=clone(preferenceSession.value);revision.value++;history.value=[];editing.value=true;overview.value=false;selected.value=draft.value[0]?.id||'';selectedCard.value=null;notice.value='';sizeHint.value=''}
function cancel(){interruptMove();closePanel();library.value=false;editing.value=false;overview.value=false;draft.value=clone(orderedSession.value);preferences.value=clone(preferenceSession.value);revision.value++;notice.value=t('已取消，恢复本次预览已应用的布局。','Cancelled. Applied preview layout restored.');editButton.value?.focus()}
function apply(){interruptMove();const result=normalizeRows(draft.value);if(result.issue||result.changed){finishPreview.value=result;openPanel('finish');return}finishApply(result.rows)}
function finishApply(rows:OrderedRow[]){const problem=validateDisplayRows(rows,preferences.value,layoutContext());if(problem.issue){warn(problem.issue);return}orderedSession.value=clone(rows);preferenceSession.value=clone(preferences.value);draft.value=clone(rows);closePanel();library.value=false;editing.value=false;overview.value=false;notice.value=t('布局已应用到本次预览；刷新恢复示例。','Layout applied in this preview. Reload resets it.');editButton.value?.focus()}
function warn(issue:string){showNotice(issueText(issue),'error')}
function mutate(change:(rows:OrderedRow[])=>OrderedRow[]){const before=homeDraft(),next=change(clone(draft.value)),plan=planDocument(before,{rows:next,issue:'',hidden:[],changed:false},layoutContext(),preferences.value,false);if(plan.status==='blocked'){warn(plan.issue);return false}return adoptDocument(plan.next)}
function undo(){interruptMove();closePanel();const previous=history.value.pop();if(previous){draft.value=clone(previous.rows);preferences.value=clone(previous.widgetPreferences);revision.value++;if(!previous.rows.some(r=>r.id===selected.value))selected.value=previous.rows[0]?.id||'';notice.value=t('已撤销上一次布局修改。','Last layout change undone.')}}
function setTemplate(template:Template){requestedTemplate.value=template;const capacity=templates.find(v=>v.id===template)!.slots.reduce((a,b)=>a+b,0);retainedIds.value=visibleIds([selectedRow.value]).slice(0,capacity)}
function setWeights(weights:number[]){commitResult(changeSize(draft.value,selected.value,{weights}))}
function setFirstWeight(value:number){const columns=selectedRow.value.columns;setWeights(columns.length===2?[value,100-value]:[value,Math.floor((100-value)/2),100-value-Math.floor((100-value)/2)])}
async function hideCard(id:WidgetId){const source=originOf(draft.value,id);if(!source)return;selected.value=source.row;commitResult(assignSlot(draft.value,source,null));selectedCard.value=null;await nextTick();const key=`slot:${source.row}:${source.column}:${source.slot}`;if(overview.value)document.querySelector<HTMLElement>(`[data-overview-empty="${key}"]`)?.focus({preventScroll:true});else contentElement.value?.querySelector<HTMLElement>(`[data-empty-slot="${key}"]`)?.focus({preventScroll:true})}
function toggleModule(id:WidgetId){if(visible.value.includes(id)){hideCard(id);return}const row=makeRow([id],'single',Math.max(320,widget(id).singleHeight));if(mutate(rows=>[...rows,row]))selected.value=row.id}
function addRow(){const row=makeRow();mutate(rows=>[...rows,row]);selected.value=row.id}
function hideRow(){mutate(rows=>rows.filter(r=>r.id!==selected.value));closePanel()}
function hideAll(){mutate(()=>[])}
function selectCard(row:string,id:WidgetId){if(suppressClick)return;selected.value=row;selectedCard.value=selectedCard.value===id?null:id;notice.value=''}
const moving=ref<MoveSource|null>(null),hoverTarget=ref(''),dragPoint=ref<{x:number;y:number}|null>(null)
const moveTargets=computed(()=>targets(draft.value)),sourceName=computed(()=>!moving.value?'':name(moving.value.id))
function issueText(issue:string){return issue==='compact'?t('此形态不支持上下位置，请换形态或选择普通位置。','This form does not support a stacked position. Change form or position.'):issue==='retained'||issue==='choose-retained'?t('请按槽位数量选择保留模块。','Choose modules within this layout’s capacity.'):issue==='incomplete-stack'?t('上下组合仍有空位，请补齐或更换行布局。','Fill the stacked slots or choose another layout.'):issue==='hard-spec'?t('整理后的规格不适配，请更换布局。','The simplified layout does not fit.'):issue==='capacity'?t('此行无法继续插入，请选择新行或更换布局。','This row is full. Use a new row or change its layout.'):issue==='structure'?t('比例合计须为100%，且与布局槽位一致。','Proportions must total 100% and match the layout.'):issue==='row-target'?t('整行请放到行间','Move whole rows between rows'):issue==='target'?t('目标已变化，请重新选择','Destination changed; choose again'):issue.startsWith('height:')?t('需要行高至少 ','Needs row height of at least ')+issue.split(':')[1]+' px':issue.startsWith('width:')?t('需要更宽的槽位：','Needs a wider slot: ')+issue.split(':')[1]+' px':t('无法采用此组合：','Cannot use this layout: ')+issue}
function targetInfo(target:MoveTarget){
 const index=target.kind!=='edge'?draft.value.findIndex(r=>r.id===target.row):target.before?draft.value.findIndex(r=>r.id===target.before):draft.value.length
 const structure=target.kind==='slot'?slotLabel(target):target.kind==='beside'?t('并排插入第','Insert beside row ')+(index+1)+t('行',''):target.before?t('第','Before row ')+(index+1)+t('行之前',''):t('页面末尾','End of page')
 const id=target.kind==='slot'?draft.value[index]?.columns[target.column]?.cards[target.slot]:null
 const result=moving.value?choosePlan(placement(draft.value,moving.value,target),moving.value.id):null
 const action=target.kind==='edge'?t('独占新行','New single row'):target.kind==='beside'?t('预览扩列','Preview new column'):id?t('替换','Replace ')+name(id)+t('，原模块隐藏','; current module will be hidden'):t('移入空位','Move into empty slot')
 const available=result&&(result.direct.status!=='blocked'||result.alternative&&result.alternative.status!=='blocked')
 return {label:structure+' · '+action+(result&&available&&(result.direct.status!=='ready')?t(' · 调整后可放入',' · Adjust to fit'):''),issue:result&&!available?result.direct.issue:''}
}
const hoverInfo=computed(()=>{const target=moveTargets.value.find(t=>targetKey(t)===hoverTarget.value);if(!target)return t('选择位置；来源空位会保留，可撤销。','Choose a destination. The source slot stays empty; undo is available.');const info=targetInfo(target);if(info.issue)return issueText(info.issue);const next=moving.value?placement(draft.value,moving.value,target).rows:[];return info.label+(target.kind!=='edge'?' → '+rowDescription(next.find(r=>r.id===target.row)):'')})
function trackTarget(event:Event){if(!moving.value)return;const key=(event.target as HTMLElement).closest<HTMLElement>('[data-placement]')?.dataset.placement;if(key)hoverTarget.value=key}
const slotTarget=(row:ActualRow,column:ActualColumn,si:number):MoveTarget=>({kind:'slot',row:row.source,column:column.source,slot:column.slots[si]})
const slotKey=(row:ActualRow,column:ActualColumn,si:number)=>targetKey(slotTarget(row,column,si))
const isFirstFragment=(row:ActualRow)=>actualRows.value.find(r=>r.source===row.source)?.id===row.id
function startMoving(source:MoveSource){if(suppressClick){suppressClick=false;return}rowSorting.cancel();finishGesture?.();overview.value=true;moving.value=source;panel.value='';notice.value=''}
async function cancelMoving(){const source=moving.value;cancelMoveGesture?.();stopMoving();notice.value=t('已取消移动，布局未改变。','Move cancelled. Layout unchanged.');await nextTick();if(source)overviewElement.value?.querySelector<HTMLElement>(`[data-drag-card="${source.id}"]`)?.focus({preventScroll:true})}
function stopMoving(){moving.value=null;hoverTarget.value='';dragPoint.value=null;if(panel.value==='targets')panel.value=''}
async function place(target:MoveTarget){if(!moving.value)return;const info=targetInfo(target);if(info.issue){warn(info.issue);return}const source=moving.value,raw=placement(draft.value,source,target);const changed=requestResult(raw,source.id);if(panel.value!=='adapt')stopMoving();if(changed)notice.value=t('位置已调整，可撤销。','Position updated. Undo is available.');await nextTick();if(panel.value!=='adapt')overviewElement.value?.querySelector<HTMLElement>(`[data-drag-card="${source.id}"]`)?.focus({preventScroll:true})}
let cancelMoveGesture:(()=>void)|undefined,suppressClick=false
function startMovePointer(event:PointerEvent,source:MoveSource){
 if(event.button!==0||event.isPrimary===false||finishGesture||panel.value)return
 rowSorting.cancel()
 cancelMoveGesture?.();const start={x:event.clientX,y:event.clientY},pointer=event.pointerId,viewport=[innerWidth,innerHeight];let active=false
 const move=(e:PointerEvent)=>{if(e.pointerId!==pointer)return;if(!active&&Math.hypot(e.clientX-start.x,e.clientY-start.y)<6)return;if(!active){startMoving(source);active=true}e.preventDefault();dragPoint.value={x:Math.max(contentElement.value?.getBoundingClientRect().left||0,Math.min(innerWidth-180,e.clientX+12)),y:Math.max(0,Math.min(innerHeight-80,e.clientY+12))};hoverTarget.value=(document.elementFromPoint(e.clientX,e.clientY)?.closest('[data-placement]') as HTMLElement)?.dataset.placement||''}
 const clean=()=>{window.removeEventListener('pointermove',move);window.removeEventListener('pointerup',up);window.removeEventListener('pointercancel',cancel);window.removeEventListener('blur',cancel);cancelMoveGesture=undefined}
 const cancel=()=>{clean();if(active){suppressClick=true;setTimeout(()=>suppressClick=false,0);stopMoving()}}
 const up=(e:PointerEvent)=>{if(e.pointerId!==pointer)return;clean();if(active){suppressClick=true;setTimeout(()=>suppressClick=false,0);if(viewport[0]!==innerWidth||viewport[1]!==innerHeight){stopMoving();return}const target=moveTargets.value.find(t=>targetKey(t)===hoverTarget.value);if(target)void place(target);else stopMoving();dragPoint.value=null}}
 cancelMoveGesture=cancel;window.addEventListener('pointermove',move,{passive:false});window.addEventListener('pointerup',up);window.addEventListener('pointercancel',cancel);window.addEventListener('blur',cancel)
}
function rowStyle(row:ActualRow){return {height:hasRowSpace(row)&&naturalSizing.sizes.value[row.id]?rowSpace(row)+'px':row.natural?'auto':row.height+'px',minHeight:row.chartHeight?row.chartHeight+'px':undefined,'--natural-chart-height':(row.chartHeight||320)+'px',gridTemplateColumns:row.columns.map(c=>'minmax('+Math.min(contentWidth.value,Math.max(...c.cards.map(id=>id?placementMinimum(id,renderPreferences.value[id],c.cards.length===2).minWidth:260)))+'px,'+c.weight+'fr)').join(' ')}}
function columnStyle(column:Pick<ActualColumn,'cards'|'split'>){if(column.cards.length===1)return {gridTemplateRows:'auto'};const stretch=column.cards.map(id=>id&&resolvePresentation(id,renderPreferences.value[id],{width:1000,height:100000,role:'stacked',favoriteCount:4,natural:true}).stretch);return {gridTemplateRows:stretch[0]?'minmax(min-content,1fr) max-content':stretch[1]?'max-content minmax(min-content,1fr)':'auto auto'}}
function chartStretches(row:ActualRow){return hasRowSpace(row)||row.columns.length>1&&row.columns.some(c=>c.compact)&&((renderPreferences.value.traffic.ruleVersion??2)>=2||row.columns.some(c=>c.compact&&c.cards.some(id=>id&&((renderPreferences.value[id].ruleVersion??2)>=2||resolvePresentation(id,renderPreferences.value[id],{width:1000,height:100000,role:'stacked',favoriteCount:4,natural:true}).stretch))))}
function divider(row:ActualRow,index:number){const element=contentElement.value?.querySelector<HTMLElement>('[data-ordered-row="'+row.id+'"]'),columns=element?.querySelectorAll<HTMLElement>('.ordered-column');if(!element||!columns?.[index])return 0;return columns[index].getBoundingClientRect().right-element.getBoundingClientRect().left+9}
let finishGesture:(()=>void)|undefined
let sizeClickPending=false
function eatSizeClick(event:MouseEvent){if(sizeClickPending){sizeClickPending=false;if(event.detail>0){event.preventDefault();event.stopImmediatePropagation()}}}
function gesture(event:PointerEvent,move:(dx:number,dy:number)=>void,done:(cancelled:boolean)=>void){
 if(event.button!==0)return;event.preventDefault();finishGesture?.();const x=event.clientX,y=event.clientY,pointer=event.pointerId,viewport=[innerWidth,innerHeight,contentElement.value?.clientWidth];let moved=false
 const onMove=(e:PointerEvent)=>{if(e.pointerId!==pointer)return;const dx=e.clientX-x,dy=e.clientY-y;moved ||= Math.hypot(dx,dy)>3;if(moved)move(dx,dy)}
 const finish=(cancelled:boolean)=>{window.removeEventListener('pointermove',onMove);window.removeEventListener('pointerup',up);window.removeEventListener('pointercancel',cancel);window.removeEventListener('blur',cancel);finishGesture=undefined;if(moved){sizeClickPending=true;suppressClick=true;setTimeout(()=>suppressClick=false,0)}done(cancelled);resizingRow.value=''}
 const up=(e:PointerEvent)=>{if(e.pointerId===pointer)finish(viewport[0]!==innerWidth||viewport[1]!==innerHeight||viewport[2]!==contentElement.value?.clientWidth)},cancel=()=>finish(true)
 finishGesture=cancel;window.addEventListener('pointermove',onMove);window.addEventListener('pointerup',up);window.addEventListener('pointercancel',cancel);window.addEventListener('blur',cancel)
}
function finishSize(before:OrderedRow[],cancelled:boolean){if(cancelled){draft.value=before;return}const issue=validateDisplayRows(draft.value,preferences.value,layoutContext()).issue;if(issue){draft.value=before;warn(issue);return}if(JSON.stringify(before)!==JSON.stringify(draft.value)){history.value=[...history.value.slice(-29),documentOf(before,preferences.value,revision.value)];revision.value++}}
function widthBounds(actual:ActualRow,index:number){const sum=actual.columns.reduce((s,c)=>s+c.weight,0),total=actual.columns[index].weight+actual.columns[index+1].weight,usable=contentWidth.value-18*(actual.columns.length-1),min=(c:ActualColumn)=>Math.max(...c.cards.map(id=>id?placementMinimum(id,preferences.value[id],c.cards.length===2).minWidth:260));return [Math.ceil(min(actual.columns[index])/usable*sum*10)/10,Math.floor((total-min(actual.columns[index+1])/usable*sum)*10)/10]}
function startWidth(event:PointerEvent,actual:ActualRow,index:number){if(event.button!==0||event.isPrimary===false||finishGesture||cancelMoveGesture||rowSorting.busy.value)return;const before=clone(draft.value),row=draft.value.find(r=>r.id===actual.source)!,a=actual.columns[index].source,b=actual.columns[index+1].source,start=row.columns[a].weight,total=start+row.columns[b].weight,width=contentWidth.value-18*(actual.columns.length-1),sum=actual.columns.reduce((s,c)=>s+c.weight,0),bounds=widthBounds(actual,index);resizingRow.value=row.id;gesture(event,dx=>{const raw=Math.round((start+dx/width*sum)/5)*5,value=Math.max(bounds[0],Math.min(bounds[1],raw));row.columns[a].weight=value;row.columns[b].weight=Number((total-value).toFixed(1));if(value!==raw)notice.value=t('已到卡片最小宽度。','Card minimum width reached.')},cancelled=>finishSize(before,cancelled))}
function nudgeWidth(actual:ActualRow,index:number,delta:number){interruptMove();selected.value=actual.source;const bounds=widthBounds(actual,index);mutate(rows=>{const row=rows.find(r=>r.id===actual.source)!,a=actual.columns[index].source,b=actual.columns[index+1].source,total=row.columns[a].weight+row.columns[b].weight,value=Math.max(bounds[0],Math.min(bounds[1],row.columns[a].weight+delta));row.columns[a].weight=value;row.columns[b].weight=Number((total-value).toFixed(1));return rows})}
function minimumHint(){sizeHint.value=t('已到内容最小高度，可拉高整行增加调整空间。','Content minimum reached. Drag the row’s bottom edge for more room.')}
function spaceValue(actual:ActualRow,raw:number){const min=naturalSizing.sizes.value[actual.id]?.height||rowSpace(actual),value=clamp(Math.round(raw),min,Math.max(min,ROW_SPACE_LIMIT));sizeHint.value=value!==Math.round(raw)?raw<min?t('已到内容最小高度。','Content minimum reached.'):t('已到可调整空间上限。','Maximum adjustable space reached.'):'';return value}
function startRowSpace(event:PointerEvent,actual:ActualRow){
 if(event.button!==0||event.isPrimary===false||finishGesture||cancelMoveGesture||rowSorting.busy.value)return
 const before=clone(draft.value),row=draft.value.find(r=>r.id===actual.source)!,start=rowElement(actual)?.getBoundingClientRect().height||rowSpace(actual)
 resizingRow.value=row.id;sizeHint.value=''
 gesture(event,(_,dy)=>{const value=spaceValue(actual,start+dy);if(value!==Math.round(start))row.minHeight=value;else if(before.find(r=>r.id===row.id)!.minHeight===undefined)delete row.minHeight;else row.minHeight=before.find(r=>r.id===row.id)!.minHeight},cancelled=>{finishSize(before,cancelled);if(cancelled)sizeHint.value=''})
}
function nudgeRowSpace(actual:ActualRow,delta:number){interruptMove();const start=rowElement(actual)?.getBoundingClientRect().height||rowSpace(actual),value=spaceValue(actual,start+delta);if(value===Math.round(start))return;mutate(rows=>{rows.find(r=>r.id===actual.source)!.minHeight=value;return rows})}
function setStackDelta(actual:ActualRow,ci:number,start:number,dy:number,rows:OrderedRow[]){
 const metrics=stackSpace(actual,ci),value=clamp(start+dy,metrics.min,metrics.max)
 if(value!==start+dy)minimumHint();else sizeHint.value=''
 if(Math.abs(value-start)<.5)return false
 rows.find(r=>r.id===actual.source)!.columns[actual.columns[ci].source].upperShare=Number(clamp(value/metrics.available*100,.01,99.99).toFixed(4));return true
}
function startStack(event:PointerEvent,actual:ActualRow,ci:number){
 if(event.button!==0||event.isPrimary===false||finishGesture||cancelMoveGesture||rowSorting.busy.value)return
 const before=clone(draft.value),element=rowElement(actual)?.querySelectorAll<HTMLElement>(':scope > .ordered-column')[ci],start=element?.querySelector<HTMLElement>(':scope > .ordered-card')?.getBoundingClientRect().height||stackSpace(actual,ci).upper
 resizingRow.value=actual.source;sizeHint.value=''
 gesture(event,(_,dy)=>{const next=clone(before);setStackDelta(actual,ci,start,dy,next);draft.value=next},cancelled=>{finishSize(before,cancelled);if(cancelled)sizeHint.value=''})
}
function nudgeStack(actual:ActualRow,ci:number,delta:number){interruptMove();const element=rowElement(actual)?.querySelectorAll<HTMLElement>(':scope > .ordered-column')[ci],start=element?.querySelector<HTMLElement>(':scope > .ordered-card')?.getBoundingClientRect().height||stackSpace(actual,ci).upper;mutate(rows=>{setStackDelta(actual,ci,start,delta,rows);return rows})}
const target=(path:string,device?:string)=>({path,query:{...route.query,layout:'ordered',variant:'rows',device,filter:path==='/devicemanagement'?'all':undefined}})
function go(path:string){if(editing.value){notice.value=t('请先应用或取消布局草稿，再打开详情。','Apply or cancel the layout draft before opening details.');return}void router.push(target(path))}
function openDevice(id:string){if(editing.value){notice.value=t('请先应用或取消布局草稿，再打开设备。','Apply or cancel the draft before opening a device.');return}void router.push(target('/devicemanagement',id))}
const colorStyle=computed(()=>['classic','vivid','bold'].includes(String(route.query.palette))?String(route.query.palette):'vivid')
function toggleTheme(){const value=theme.value==='dark'?'light':'dark';document.body.setAttribute('theme',value);void router.replace({query:{...route.query,theme:value}})}
function toggleLanguage(){if(editing.value){notice.value=t('切换语言会刷新示例，请先结束布局编辑。','Changing language reloads the sample. Finish editing first.');return}location.href=router.resolve({path:'/',query:{...route.query,layout:'ordered',variant:'rows',lang:english?'zh-cn':'en'}}).href}
function changeQuery(key:string,event:Event){if(key!=='palette')demo.recovered=false;void router.replace({query:{...route.query,[key]:(event.target as HTMLSelectElement).value}})}
function changeSample(event:Event){if(editing.value){notice.value=t('切换示例会刷新布局，请先应用或取消草稿。','Changing the sample reloads the layout. Apply or cancel first.');return}location.href=router.resolve({path:'/',query:{...route.query,layout:'ordered',variant:'rows',preset:(event.target as HTMLSelectElement).value}}).href}
let libraryOpener:HTMLElement|null=null
watch(library,async open=>{if(open){interruptMove();panel.value='';selectedCard.value=null;libraryOpener=document.activeElement as HTMLElement;await nextTick();libraryElement.value?.querySelector<HTMLButtonElement>('button')?.focus()}else libraryOpener?.focus({preventScroll:true})})
function closeLibrary(){library.value=false;pendingSlot.value=null}
function libraryKeyboard(event:KeyboardEvent){if(event.key==='Escape'){event.stopPropagation();closeLibrary()}if(event.key==='Tab'){const controls=Array.from(libraryElement.value?.querySelectorAll<HTMLElement>('button,input')||[]),first=controls[0],last=controls[controls.length-1];if(event.shiftKey&&document.activeElement===first){event.preventDefault();last?.focus()}else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first?.focus()}}}
watch(()=>route.query.theme,()=>theme.value=getTheme())
function escapeMove(event:KeyboardEvent){if(event.key==='Escape'&&(rowSorting.busy.value||moving.value||cancelMoveGesture||finishGesture)){event.preventDefault();event.stopImmediatePropagation();const source=moving.value;interruptMove();if(source)void nextTick(()=>overviewElement.value?.querySelector<HTMLElement>(`[data-drag-card="${source.id}"]`)?.focus({preventScroll:true}));notice.value=t('已取消操作，布局未改变。','Action cancelled. Layout unchanged.')}else if(event.key==='Escape'&&(library.value||panel.value)){event.preventDefault();event.stopImmediatePropagation();if(library.value)closeLibrary();else closePanel()}}
function interruptMove(){rowSorting.cancel();cancelMoveGesture?.();finishGesture?.();stopMoving()}
function environmentChanged(){interruptMove();if(['display','adapt','slot'].includes(panel.value)){closePanel();notice.value=t('预览条件已变化，请重新选择。','Preview conditions changed. Choose again.')}}
watch(()=>[demo.scenario,demo.authority],environmentChanged)
watch(revision,()=>{cancelMoveGesture?.();rowSorting.cancel()})
let observer:ResizeObserver|undefined
onMounted(()=>{window.addEventListener('keydown',escapeMove,true);window.addEventListener('resize',environmentChanged);observer=new ResizeObserver(entries=>{for(const e of entries){if(e.target===availableElement.value)availableWidth.value=e.contentRect.width;if(e.target===contentElement.value)contentWidth.value=e.contentRect.width}naturalSizing.schedule()});if(availableElement.value)observer.observe(availableElement.value);if(contentElement.value)observer.observe(contentElement.value);Object.defineProperty(window,'__quickstartOrdered',{configurable:true,get:()=>({editing:editing.value,overview:overview.value,panel:panel.value,applied:clone(orderedSession.value),draft:clone(draft.value),preferences:clone(preferences.value),appliedPreferences:clone(preferenceSession.value),document:homeDraft(),visible:visible.value,width:contentWidth.value,actual:actualRows.value,sizing:naturalSizing.debug(),minimum:selectedRow.value?rowMinimum(selectedRow.value):0,history:history.value.length,moving:clone(moving.value),rowSort:rowSorting.debug(),gestureActive:Boolean(rowSorting.busy.value||cancelMoveGesture||finishGesture)})})})
onUnmounted(()=>{observer?.disconnect();interruptMove();window.removeEventListener('keydown',escapeMove,true);window.removeEventListener('resize',environmentChanged);finishGesture?.();delete (window as any).__quickstartOrdered})
const scenes=[['network-dns','DNS错误','DNS error'],['network-dns-dhcp','DHCP与DNS错误','DHCP / DNS error'],['network-source','软件源错误','Source error'],['network-offline','未联网','Offline'],['network-detecting','联网检测中','Checking connection'],['network-unknown','状态待确认','Unknown connection'],['network-no-address','无地址信息','No addresses'],['network-loading','读取连接中','Loading connection'],['network-many','48个DNS与128个端口','Many DNS / ports'],['ports-loading','读取接口中','Loading interfaces'],['network-source-write-error','软件源提交失败','Source save rejected'],['ports-empty','没有接口','No interfaces'],['ports-error','接口读取失败','Interfaces unavailable'],['network-write-error','配置提交失败','Configuration rejected'],['system-error','系统读取失败','System unavailable'],['long-names','长设备名称','Long names'],['many-disks','24块磁盘','24 disks'],['normal','正常','Normal'],['many','128台设备','128 devices'],['empty','没有设备','Empty'],['loading','加载中','Loading'],['error','设备API失败','Inventory error'],['stale','旧快照','Stale snapshot'],['old','旧后端','Old backend'],['permission','权限失败','Access denied'],['no-speed','服务未安装','Service missing'],['disabled','服务未启用','Service disabled'],['speed-error','服务异常','Service error'],['unsupported','固件不支持','Unsupported'],['network-error','网络模块失败','Network unavailable'],['storage-error','存储失败','Storage unavailable'],['services-error','服务失败','Services unavailable'],['apps-empty','没有扩展应用','No extra apps'],['apps-error','应用清单失败','Apps unavailable'],['apps-loading','应用加载','Apps loading'],['apps-many','大量应用','Many apps']]
scenes.unshift(['network-reference','原版截图对照','Original screenshot reference'])
scenes.unshift(['system-missing','系统指标缺失（原口径）','Missing metrics (original convention)'],['system-zero','系统指标零值（原口径）','Zero metrics (original convention)'],['system-high','系统高负载','High system metrics'],['many-shares','24条共享','24 shares'],['business-all','六组件：服务全部运行','All services running'],['business-disks','磁盘：RAID与异常状态','RAID and disk states'],['business-missing','服务未配置／未安装','Unconfigured / not installed'],['business-stopped','Docker已安装未启用','Docker stopped'],['business-ipv6','仅IPv6域名','IPv6 domain only'],['business-raid-error','RAID读取失败','RAID read failed'],['business-empty','没有磁盘','No disks'],['business-write-error','业务提交被拒绝','Business write rejected'],['business-http-error','业务HTTP提交失败','Business HTTP write failed'],['business-migration-warning','Docker非空目录确认','Docker overwrite review'],['business-install','服务需安装','Service installation needed'])
</script>
<style src="./canvas.css" scoped></style>
<style src="./layoutDesignHome.css" scoped></style>
<style src="./orderedHome.css" scoped></style>
<style src="./directEdit.css"></style>
<style src="./slotEdit.css"></style>
<style src="./presentationEdit.css"></style>
<style src="./homeTypography.css"></style>
<style src="./naturalHome.css"></style>
<style src="./rowSizing.css"></style>
<style src="./widgetActionGroup.css"></style>
<style src="./homeWidgetChrome.css"></style>
<style src="./homeVisualTokens.css"></style>
<style src="./homeWidgetDetails.css"></style>
<style src="./homeColorStyles.css"></style>
<style src="../home/homeSemanticColors.css"></style>
