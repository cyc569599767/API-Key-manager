<script lang="ts" setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import {
  BackupData,
  BatchDeleteAPIKeys,
  BatchExportAPIKeys,
  BatchImportAPIKeys,
  CreateAPIKey,
  GetCopyCredentials,
  DisableAPIKey,
  EnableAPIKey,
  GetAPIKeyDetail,
  GetDashboardStats,
  GetFilterOptions,
  ListAPIKeys,
  ListAuditLogs,
  RestoreData,
  TestAPIKey,
  UpdateAPIKey,
} from '../wailsjs/go/main/App'
import { ClipboardSetText } from '../wailsjs/runtime/runtime'
import { model } from '../wailsjs/go/models'

type FormState = {
  id?: number
  name: string
  provider: string
  environment: string
  groupName: string
  budgetTag: string
  baseUrl: string
  apiKey: string
  testEndpoint: string
  timeoutMs: number
  rateLimitRpm: number
  failureStrategy: string
}

const keys = ref<model.APIKey[]>([])
const selectedDetail = ref<model.APIKeyDetail | null>(null)
const stats = ref(new model.DashboardStats())
const audits = ref<model.AuditLog[]>([])
const providers = ref<string[]>([])
const groupNames = ref<string[]>([])
const providerOptions = ['OpenAI', 'Anthropic', 'Gemini', 'DeepSeek', 'Qwen', 'Azure OpenAI', 'OpenAI Compatible']
const failureStrategyOptions = ['连续 2 次告警', '连续 3 次告警', '单次失败告警', '仅记录审计']
const defaultModels: Record<string, string> = {
  OpenAI: 'gpt-5.5',
  Anthropic: 'claude-haiku-4-5-20251001',
}
const loading = ref(false)
const testing = ref(false)
const saving = ref(false)
const batchImporting = ref(false)
const batchDeleting = ref(false)
const rowTestingIds = ref<number[]>([])
const batchExporting = ref(false)
const backingUp = ref(false)
const restoring = ref(false)
const errorMessage = ref('')
const showModal = ref(false)
const showBatchModal = ref(false)
const editing = ref(false)
const selectedIds = ref<number[]>([])
const batchText = ref('')
const batchFormat = ref('auto')
const batchDuplicateStrategy = ref('skip')
const batchResult = ref<model.BatchImportAPIKeysResult | null>(null)

const filters = reactive<model.APIKeyFilter>({ search: '', provider: 'all', status: 'all', environment: 'all', groupName: 'all', page: 1, pageSize: 20 })
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const form = reactive<FormState>(emptyForm())
const batchDefaults = reactive<FormState>(emptyForm())

const selectedKey = computed(() => selectedDetail.value?.key || null)
const latestResult = computed(() => selectedDetail.value?.latestTestResult || null)
const activeKeys = computed(() => stats.value.total - stats.value.disabled)
const healthRate = computed(() => activeKeys.value === 0 ? 0 : Math.round((stats.value.available / activeKeys.value) * 100))
const selectedIdSet = computed(() => new Set(selectedIds.value))
const allVisibleSelected = computed(() => keys.value.length > 0 && keys.value.every((key) => selectedIdSet.value.has(key.id)))
const selectedCount = computed(() => selectedIds.value.length)
const totalPages = computed(() => Math.max(1, Math.ceil(pagination.total / pagination.pageSize)))

watch(filters, () => {
  pagination.page = 1
  refreshList()
}, { deep: true })
watch(() => form.provider, (provider) => applyDefaultModel(form, provider))
watch(() => batchDefaults.provider, (provider) => applyDefaultModel(batchDefaults, provider))

onMounted(async () => {
  await refreshAll()
})

function emptyForm(): FormState {
  return {
    name: '',
    provider: 'OpenAI',
    environment: 'Default',
    groupName: '',
    budgetTag: '',
    baseUrl: 'https://api.openai.com/v1',
    apiKey: '',
    testEndpoint: defaultModelForProvider('OpenAI'),
    timeoutMs: 10000,
    rateLimitRpm: 120,
    failureStrategy: '连续 2 次告警',
  }
}

function defaultModelForProvider(provider: string) {
  return defaultModels[provider] || 'gpt-5.5'
}

function applyDefaultModel(target: FormState, provider: string) {
  if (!target.testEndpoint || Object.values(defaultModels).includes(target.testEndpoint)) {
    target.testEndpoint = defaultModelForProvider(provider)
  }
}

async function refreshAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    await Promise.all([refreshList(), refreshStats(), refreshAudits(), refreshFilterOptions()])
  } catch (error) {
    showError(error)
  } finally {
    loading.value = false
  }
}

async function refreshList() {
  const result = await ListAPIKeys({ ...filters, page: pagination.page, pageSize: pagination.pageSize } as model.APIKeyFilter)
  keys.value = result.items || []
  pagination.total = result.total || 0
  pagination.page = result.page || pagination.page
  pagination.pageSize = result.pageSize || pagination.pageSize
  if (!selectedKey.value && keys.value.length > 0) {
    await selectKey(keys.value[0].id)
  }
}

async function refreshStats() {
  stats.value = await GetDashboardStats()
}

async function refreshAudits() {
  audits.value = await ListAuditLogs(10)
}

async function refreshFilterOptions() {
  const options = await GetFilterOptions()
  providers.value = options.providers || []
  groupNames.value = options.groupNames || []
}

function toggleSelectKey(id: number) {
  selectedIds.value = selectedIdSet.value.has(id)
    ? selectedIds.value.filter((selectedId) => selectedId !== id)
    : [...selectedIds.value, id]
}

function toggleSelectAllVisible() {
  const visibleIds = keys.value.map((key) => key.id)
  selectedIds.value = allVisibleSelected.value
    ? selectedIds.value.filter((id) => !visibleIds.includes(id))
    : [...new Set([...selectedIds.value, ...visibleIds])]
}

function goToPage(page: number) {
  const nextPage = Math.min(Math.max(page, 1), totalPages.value)
  if (nextPage === pagination.page) return
  pagination.page = nextPage
  refreshList()
}

function changePageSize() {
  pagination.page = 1
  refreshList()
}

async function selectKey(id: number) {
  selectedDetail.value = await GetAPIKeyDetail(id)
}

function openCreateModal() {
  Object.assign(form, emptyForm())
  editing.value = false
  showModal.value = true
}

function openBatchModal() {
  Object.assign(batchDefaults, emptyForm())
  batchDefaults.apiKey = ''
  batchText.value = ''
  batchFormat.value = 'auto'
  batchDuplicateStrategy.value = 'skip'
  batchResult.value = null
  showBatchModal.value = true
}

function openEditModal() {
  if (!selectedKey.value) return
  Object.assign(form, {
    id: selectedKey.value.id,
    name: selectedKey.value.name,
    provider: selectedKey.value.provider,
    environment: 'Default',
    groupName: selectedKey.value.groupName,
    budgetTag: '',
    baseUrl: selectedKey.value.baseUrl,
    apiKey: '',
    testEndpoint: selectedKey.value.testEndpoint,
    timeoutMs: selectedKey.value.timeoutMs,
    rateLimitRpm: selectedKey.value.rateLimitRpm,
    failureStrategy: selectedKey.value.failureStrategy,
  })
  editing.value = true
  showModal.value = true
}

async function saveKey() {
  saving.value = true
  errorMessage.value = ''
  try {
    if (editing.value && form.id) {
      await UpdateAPIKey(form.id, form as model.UpdateAPIKeyInput)
      selectedDetail.value = await GetAPIKeyDetail(form.id)
    } else {
      const created = await CreateAPIKey(form as model.CreateAPIKeyInput)
      selectedDetail.value = await GetAPIKeyDetail(created.id)
    }
    showModal.value = false
    await Promise.all([refreshList(), refreshStats(), refreshAudits(), refreshFilterOptions()])
  } catch (error) {
    showError(error)
  } finally {
    saving.value = false
  }
}

async function submitBatchImport() {
  batchImporting.value = true
  errorMessage.value = ''
  try {
    const defaults = { ...batchDefaults, apiKey: '' } as model.CreateAPIKeyInput
    const result = await BatchImportAPIKeys({
      rawText: batchText.value,
      defaults,
      format: batchFormat.value,
      duplicateStrategy: batchDuplicateStrategy.value,
    } as model.BatchImportAPIKeysInput)
    batchResult.value = result
    if (result.created > 0) {
      await Promise.all([refreshList(), refreshStats(), refreshAudits(), refreshFilterOptions()])
      if (result.createdKeys?.length) {
        selectedDetail.value = await GetAPIKeyDetail(result.createdKeys[0].id)
      }
    } else {
      await refreshAudits()
    }
    if (result.failed === 0 && result.skipped === 0 && result.created > 0) {
      showBatchModal.value = false
    }
  } catch (error) {
    showError(error)
  } finally {
    batchImporting.value = false
  }
}

function resetBatchImport() {
  batchText.value = ''
  batchResult.value = null
}

async function confirmBatchDelete() {
  if (selectedIds.value.length === 0) return
  const ids = [...selectedIds.value]
  const confirmed = window.confirm(`确认永久删除选中的 ${ids.length} 个 API Key？\n\n该操作会永久删除 API Key，并级联删除相关测试结果；操作不可恢复。`)
  if (!confirmed) return

  batchDeleting.value = true
  errorMessage.value = ''
  try {
    await BatchDeleteAPIKeys({ ids } as model.BatchDeleteAPIKeysInput)
    selectedIds.value = []
    if (selectedKey.value && ids.includes(selectedKey.value.id)) {
      selectedDetail.value = null
    }
    await Promise.all([refreshList(), refreshStats(), refreshAudits(), refreshFilterOptions()])
  } catch (error) {
    showError(error)
  } finally {
    batchDeleting.value = false
  }
}

async function confirmDeleteKey(id: number, name: string) {
  const confirmed = window.confirm(`确认永久删除“${name}”？\n\n该操作会永久删除 API Key，并级联删除相关测试结果；操作不可恢复。`)
  if (!confirmed) return

  batchDeleting.value = true
  errorMessage.value = ''
  try {
    await BatchDeleteAPIKeys({ ids: [id] } as model.BatchDeleteAPIKeysInput)
    selectedIds.value = selectedIds.value.filter((selectedId) => selectedId !== id)
    if (selectedKey.value?.id === id) {
      selectedDetail.value = null
    }
    await Promise.all([refreshList(), refreshStats(), refreshAudits(), refreshFilterOptions()])
  } catch (error) {
    showError(error)
  } finally {
    batchDeleting.value = false
  }
}

async function runRowTest(id: number) {
  if (rowTestingIds.value.includes(id)) return
  rowTestingIds.value = [...rowTestingIds.value, id]
  errorMessage.value = ''
  try {
    const detail = await TestAPIKey(id)
    const index = keys.value.findIndex((key) => key.id === id)
    if (index !== -1) {
      keys.value[index] = detail.key
    }
    selectedDetail.value = detail
    await Promise.all([refreshStats(), refreshAudits()])
  } catch (error) {
    showError(error)
  } finally {
    rowTestingIds.value = rowTestingIds.value.filter((testingId) => testingId !== id)
  }
}

async function confirmBatchExport() {
  if (selectedIds.value.length === 0) return
  const ids = [...selectedIds.value]
  const confirmed = window.confirm(`确认导出选中的 ${ids.length} 个 API Key？\n\n导出的 TXT 文件将包含明文 baseurl 和 apiKey，请妥善保存。`)
  if (!confirmed) return

  batchExporting.value = true
  errorMessage.value = ''
  try {
    const result = await BatchExportAPIKeys({ ids } as model.BatchExportAPIKeysInput)
    if (!result.canceled) {
      errorMessage.value = `导出完成：${result.exported} 个 API Key 已保存到 ${result.path}${result.skipped ? `，跳过 ${result.skipped} 个` : ''}`
      await refreshAudits()
    }
  } catch (error) {
    showError(error)
  } finally {
    batchExporting.value = false
  }
}

async function confirmBackup() {
  const confirmed = window.confirm('备份文件将包含数据库和加密密钥，拥有该 zip 的人可能恢复并解密你的 API Key。请确认保存到安全位置。')
  if (!confirmed) return

  backingUp.value = true
  errorMessage.value = ''
  try {
    const result = await BackupData()
    if (!result.canceled) {
      errorMessage.value = `备份完成：数据已保存到 ${result.path}`
      await refreshAudits()
    }
  } catch (error) {
    showError(error)
  } finally {
    backingUp.value = false
  }
}

async function confirmRestore() {
  const confirmed = window.confirm('还原会用备份中的 keymanager.sqlite 和 secret.key 覆盖当前数据。当前所有 API Key、测试结果、审计记录和设置都将被替换。建议先备份当前数据。是否继续？')
  if (!confirmed) return
  const finalConfirmed = window.confirm('最后确认：当前数据将被覆盖且不可撤销。继续还原？')
  if (!finalConfirmed) return

  restoring.value = true
  errorMessage.value = ''
  try {
    const result = await RestoreData()
    if (!result.canceled) {
      selectedIds.value = []
      selectedDetail.value = null
      await refreshAll()
      errorMessage.value = `还原完成：已从 ${result.path} 恢复数据`
    }
  } catch (error) {
    showError(error)
  } finally {
    restoring.value = false
  }
}

async function toggleDisabled() {
  if (!selectedKey.value) return
  errorMessage.value = ''
  try {
    if (selectedKey.value.status === 'disabled') {
      await EnableAPIKey(selectedKey.value.id)
    } else {
      await DisableAPIKey(selectedKey.value.id)
    }
    selectedDetail.value = await GetAPIKeyDetail(selectedKey.value.id)
    await Promise.all([refreshList(), refreshStats(), refreshAudits()])
  } catch (error) {
    showError(error)
  }
}

async function runTest() {
  if (!selectedKey.value) return
  testing.value = true
  errorMessage.value = ''
  try {
    const detail = await TestAPIKey(selectedKey.value.id)
    selectedDetail.value = detail
    const index = keys.value.findIndex((key) => key.id === detail.key.id)
    if (index !== -1) {
      keys.value[index] = detail.key
    }
    await Promise.all([refreshStats(), refreshAudits()])
  } catch (error) {
    showError(error)
  } finally {
    testing.value = false
  }
}

async function copyAPIKey() {
  if (!selectedKey.value) return
  const apiKey = await GetCopyCredentials(selectedKey.value.id)
  await ClipboardSetText(apiKey)
  await refreshAudits()
}

function showError(error: unknown) {
  errorMessage.value = error instanceof Error ? error.message : String(error)
}

function statusText(status: string) {
  return {
    available: '可用',
    error: '异常',
    untested: '未测试',
    disabled: '已禁用',
  }[status] || status
}

function formatTime(value: any) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString('zh-CN', { hour12: false })
}

function assistantReply(result: model.TestResult | null) {
  if (!result) return '—'
  if (result.responseBodyJson) {
    try {
      const payload = JSON.parse(result.responseBodyJson)
      const content = payload?.choices?.[0]?.message?.content ?? payload?.content?.[0]?.text ?? payload?.message?.content
      if (content) return String(content)
    } catch {
      return result.responseBodyJson
    }
  }
  return result.errorMessage || '—'
}

function auditTone(action: string) {
  if (action.includes('failed') || action.includes('disabled')) return 'danger'
  if (action.includes('tested') || action.includes('enabled')) return 'success'
  if (action.includes('copied')) return 'accent'
  return 'warning'
}
</script>

<template>
  <main class="app-shell">
    <section class="hero">
      <div class="hero-content">
        <div class="eyebrow"><span class="pulse"></span>API Key Vault · {{ stats.total }} keys monitored</div>
        <h1>API Key 统一管理</h1>
        <p>集中存储自定义 key 和代理地址，并检测可用性、延迟、限流和鉴权异常。</p>
        <div class="hero-metrics">
          <article><small>健康率</small><strong>{{ healthRate }}%</strong></article>
          <article><small>可用</small><strong class="success">{{ stats.available }}</strong></article>
          <article><small>异常</small><strong class="danger">{{ stats.error }}</strong></article>
          <article><small>未测试</small><strong class="warning">{{ stats.untested }}</strong></article>
        </div>
      </div>
      <div class="network-card">
        <div class="network-glow"></div>
        <article class="status-card">
          <span class="badge success">Live Check</span>
          <strong>{{ stats.available }} / {{ activeKeys }}</strong>
          <p>OpenAI-compatible endpoints monitored</p>
          <div class="latency-line"><span></span><span></span><span></span></div>
        </article>
        <article class="status-card floating">
          <span class="badge danger">Failing</span>
          <strong>{{ stats.error }}</strong>
          <p>需要更新或禁用</p>
        </article>
      </div>
    </section>

    <section class="workspace">
      <div class="inventory panel">
        <header class="panel-header">
          <div>
            <h2>API Key 清单</h2>
            <p>统一管理自定义 API key、base_url 和可用性检测状态。</p>
          </div>
          <div class="header-actions">
            <button class="danger-button small" :disabled="selectedCount === 0 || batchDeleting" @click="confirmBatchDelete">{{ batchDeleting ? '删除中...' : selectedCount ? `批量删除 (${selectedCount})` : '批量删除' }}</button>
            <button class="secondary small" :disabled="selectedCount === 0 || batchExporting" @click="confirmBatchExport">{{ batchExporting ? '导出中...' : selectedCount ? `批量导出 (${selectedCount})` : '批量导出' }}</button>
            <button class="secondary small" @click="openBatchModal">批量导入</button>
            <button class="primary small" @click="openCreateModal">添加 Key</button>
          </div>
        </header>
        <div class="stats-row">
          <article><small>总数</small><strong>{{ stats.total }}</strong></article>
          <article><small>可用</small><strong class="success">{{ stats.available }}</strong></article>
          <article><small>异常</small><strong class="danger">{{ stats.error }}</strong></article>
          <article><small>未测试</small><strong class="warning">{{ stats.untested }}</strong></article>
        </div>
        <div class="filters">
          <select v-model="filters.provider">
            <option value="all">Provider: 全部</option>
            <option v-for="provider in providers" :key="provider" :value="provider">{{ provider }}</option>
          </select>
          <select v-model="filters.groupName">
            <option value="all">分组: 全部</option>
            <option v-for="groupName in groupNames" :key="groupName" :value="groupName">{{ groupName }}</option>
          </select>
          <select v-model="filters.status">
            <option value="all">状态: 全部</option>
            <option value="available">可用</option>
            <option value="error">异常</option>
            <option value="untested">未测试</option>
            <option value="disabled">已禁用</option>
          </select>
        </div>
        <label class="global-search inventory-search">
          <span>⌕</span>
          <input v-model="filters.search" placeholder="搜索 key 名称、provider、base_url 或所属分组" />
        </label>
        <div class="table-wrap">
          <table>
            <thead>
              <tr><th><input type="checkbox" :checked="allVisibleSelected" :disabled="keys.length === 0" @change="toggleSelectAllVisible" /></th><th>名称</th><th>所属分组</th><th>Provider</th><th>Base URL</th><th>状态</th><th>操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="key in keys" :key="key.id" :class="{ selected: selectedKey?.id === key.id }" @click="selectKey(key.id)">
                <td><input type="checkbox" :checked="selectedIdSet.has(key.id)" @click.stop @change="toggleSelectKey(key.id)" /></td>
                <td><strong>{{ key.name }}</strong><small>{{ key.maskedKey }}</small></td>
                <td class="group-cell">{{ key.groupName || '—' }}</td>
                <td>{{ key.provider }}</td>
                <td class="mono">{{ key.baseUrl }}</td>
                <td><span class="status" :class="key.status">{{ statusText(key.status) }}</span></td>
                <td class="row-actions">
                  <button class="small" :disabled="key.status === 'disabled' || rowTestingIds.includes(key.id)" @click.stop="runRowTest(key.id)">{{ rowTestingIds.includes(key.id) ? '测试中...' : '测试' }}</button>
                  <button class="small danger-button" :disabled="batchDeleting" @click.stop="confirmDeleteKey(key.id, key.name)">删除</button>
                </td>
              </tr>
              <tr v-if="!loading && keys.length === 0"><td colspan="7" class="empty">暂无 API Key，点击“添加 Key”开始。</td></tr>
            </tbody>
          </table>
        </div>
        <div class="pagination-bar">
          <span>共 {{ pagination.total }} 条，已选择 {{ selectedCount }} 条</span>
          <div class="page-actions">
            <label class="page-size">每页
              <select v-model.number="pagination.pageSize" @change="changePageSize">
                <option :value="10">10</option>
                <option :value="20">20</option>
                <option :value="50">50</option>
                <option :value="100">100</option>
              </select>
            </label>
            <button class="small" :disabled="pagination.page <= 1" @click="goToPage(pagination.page - 1)">上一页</button>
            <strong>{{ pagination.page }} / {{ totalPages }}</strong>
            <button class="small" :disabled="pagination.page >= totalPages" @click="goToPage(pagination.page + 1)">下一页</button>
          </div>
        </div>
      </div>

      <aside class="detail panel compact-detail">
        <template v-if="selectedKey">
          <section class="detail-head">
            <small>当前选中</small>
            <h2>{{ selectedKey.provider }} · {{ selectedKey.name }}</h2>
            <span class="status" :class="selectedKey.status">{{ statusText(selectedKey.status) }}</span>
          </section>
          <section class="card">
            <h3>凭据</h3>
            <dl>
              <dt>API Key</dt><dd class="mono muted-box">{{ selectedKey.maskedKey }}</dd>
              <dt>Base URL</dt><dd class="mono muted-box">{{ selectedKey.baseUrl }}</dd>
              <dt>所属分组</dt><dd class="mono muted-box">{{ selectedKey.groupName || '—' }}</dd>
            </dl>
            <div class="detail-actions">
              <button class="primary" :disabled="testing || selectedKey.status === 'disabled'" @click="runTest">{{ testing ? '测试中...' : '立即测试' }}</button>
              <button @click="copyAPIKey">复制 API Key</button>
              <button @click="openEditModal">编辑</button>
              <button class="danger-button" @click="toggleDisabled">{{ selectedKey.status === 'disabled' ? '启用' : '禁用' }}</button>
            </div>
          </section>
          <section class="card test-result">
            <h3>最近一次可用性测试</h3>
            <p>测试请求会向 {{ selectedKey.testEndpoint }} 模型发送“hi”完成，不会发送业务 prompt。</p>
            <dl>
              <dt>响应状态码</dt><dd>{{ latestResult ? latestResult.httpStatus || 0 : '未测试' }}</dd>
              <dt>首包延迟</dt><dd>{{ latestResult ? `${latestResult.latencyMs} ms` : '—' }}</dd>
              <dt>测试时间</dt><dd>{{ latestResult ? formatTime(latestResult.testedAt) : '—' }}</dd>
              <dt v-if="latestResult?.errorMessage">错误摘要</dt><dd v-if="latestResult?.errorMessage">{{ latestResult.errorMessage }}</dd>
            </dl>
            <div v-if="latestResult" class="conversation-preview">
              <article class="message user-message">
                <small>我</small>
                <p>hi</p>
              </article>
              <article class="message assistant-message">
                <small>对方</small>
                <p>{{ assistantReply(latestResult) }}</p>
              </article>
            </div>
          </section>
          <section class="card audit">
            <header><h3>最近动态</h3><span>最近 10 条</span></header>
            <ul>
              <li v-for="log in audits" :key="log.id">
                <i :class="auditTone(log.action)"></i>
                <div><strong>{{ log.summary }}</strong><small>{{ log.apiKeyName || log.action }}</small></div>
                <time>{{ formatTime(log.createdAt) }}</time>
              </li>
              <li v-if="audits.length === 0" class="empty-audit">暂无审计记录</li>
            </ul>
          </section>
          <section class="card backup-card">
            <header><h3>数据备份</h3><span>数据库 + 加密密钥</span></header>
            <p>备份会导出 keymanager.sqlite 和 secret.key。请妥善保存，泄露后可能导致 API Key 被恢复和解密。</p>
            <div class="backup-actions">
              <button class="secondary" :disabled="backingUp || restoring" @click="confirmBackup">{{ backingUp ? '备份中...' : '备份' }}</button>
              <button class="danger-button" :disabled="backingUp || restoring" @click="confirmRestore">{{ restoring ? '还原中...' : '还原' }}</button>
            </div>
          </section>
        </template>
        <div v-else class="empty-detail">选择或添加一个 API Key 查看详情。</div>
      </aside>
    </section>

    <div v-if="errorMessage" class="toast">
      <span>{{ errorMessage }}</span>
      <button type="button" @click="errorMessage = ''">×</button>
    </div>

    <div v-if="showModal" class="modal-backdrop" @click.self="showModal = false">
      <form class="modal" @submit.prevent="saveKey">
        <header><h2>{{ editing ? '编辑 API Key' : '添加 API Key' }}</h2><button type="button" @click="showModal = false">×</button></header>
        <div class="form-grid">
          <label>名称<input v-model="form.name" required /></label>
          <label>Provider
            <select v-model="form.provider" required>
              <option v-for="provider in providerOptions" :key="provider" :value="provider">{{ provider }}</option>
            </select>
          </label>
          <label>Base URL<input v-model="form.baseUrl" required /></label>
          <label>API Key<input v-model="form.apiKey" :required="!editing" type="password" :placeholder="editing ? '留空则不修改' : ''" /></label>
          <label>测试模型<input v-model="form.testEndpoint" required /></label>
          <label>所属分组<input v-model="form.groupName" /></label>
          <label>超时阈值 ms<input v-model.number="form.timeoutMs" type="number" min="1000" /></label>
          <label>速率限制 req/min<input v-model.number="form.rateLimitRpm" type="number" min="1" /></label>
          <label class="wide">失败策略<input v-model="form.failureStrategy" /></label>
        </div>
        <footer><button type="button" @click="showModal = false">取消</button><button class="primary" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button></footer>
      </form>
    </div>

    <div v-if="showBatchModal" class="modal-backdrop" @click.self="showBatchModal = false">
      <form class="modal batch-modal" @submit.prevent="submitBatchImport">
        <header><h2>批量导入 API Key</h2><button type="button" @click="showBatchModal = false">×</button></header>
        <p class="batch-help">支持每行一个 key、JSON Lines、带表头 CSV/TSV。API Key 仅用于加密入库，不会写入审计日志。</p>
        <label class="batch-textarea">
          粘贴导入内容
          <textarea v-model="batchText" required placeholder="sk-xxxx&#10;sk-yyyy&#10;或：name,provider,baseUrl,apiKey" />
        </label>
        <div class="form-grid">
          <label>格式
            <select v-model="batchFormat">
              <option value="auto">自动识别</option>
              <option value="lines">每行一个 Key</option>
              <option value="jsonl">JSON Lines</option>
              <option value="csv">CSV 表头</option>
              <option value="tsv">TSV 表头</option>
            </select>
          </label>
          <label>重复处理
            <select v-model="batchDuplicateStrategy">
              <option value="skip">跳过重复</option>
              <option value="error">标记失败</option>
            </select>
          </label>
          <label>默认 Provider
            <select v-model="batchDefaults.provider" required>
              <option v-for="provider in providerOptions" :key="provider" :value="provider">{{ provider }}</option>
            </select>
          </label>
          <label>默认 Base URL<input v-model="batchDefaults.baseUrl" required /></label>
          <label>默认测试模型<input v-model="batchDefaults.testEndpoint" required /></label>
          <label>默认分组<input v-model="batchDefaults.groupName" /></label>
          <label>默认超时 ms<input v-model.number="batchDefaults.timeoutMs" type="number" min="1000" /></label>
          <label>默认速率限制<input v-model.number="batchDefaults.rateLimitRpm" type="number" min="1" /></label>
          <label class="wide">默认失败策略
            <select v-model="batchDefaults.failureStrategy">
              <option v-for="strategy in failureStrategyOptions" :key="strategy" :value="strategy">{{ strategy }}</option>
            </select>
          </label>
        </div>
        <section v-if="batchResult" class="batch-result">
          <div class="batch-summary">
            <article><small>总行数</small><strong>{{ batchResult.total }}</strong></article>
            <article><small>成功</small><strong class="success">{{ batchResult.created }}</strong></article>
            <article><small>跳过</small><strong class="warning">{{ batchResult.skipped }}</strong></article>
            <article><small>失败</small><strong class="danger">{{ batchResult.failed }}</strong></article>
          </div>
          <div v-if="batchResult.failures?.length" class="batch-details">
            <h3>失败明细</h3>
            <ul><li v-for="item in batchResult.failures.slice(0, 50)" :key="`f-${item.line}`">第 {{ item.line }} 行 · {{ item.name || item.maskedKey || '未命名' }} · {{ item.reason }}</li></ul>
          </div>
          <div v-if="batchResult.skippedRows?.length" class="batch-details">
            <h3>跳过明细</h3>
            <ul><li v-for="item in batchResult.skippedRows.slice(0, 50)" :key="`s-${item.line}`">第 {{ item.line }} 行 · {{ item.name || item.maskedKey || '未命名' }} · {{ item.reason }}</li></ul>
          </div>
        </section>
        <footer>
          <button type="button" @click="resetBatchImport">清空</button>
          <button type="button" @click="showBatchModal = false">关闭</button>
          <button class="primary" :disabled="batchImporting">{{ batchImporting ? '导入中...' : '开始导入' }}</button>
        </footer>
      </form>
    </div>
  </main>
</template>
