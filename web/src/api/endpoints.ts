import api from './client';

// --- Identity BC ---
export const tenantApi = {
  list: () => api.get('/tenants'),
  get: (id: number) => api.get(`/tenants/${id}`),
  create: (data: Record<string, unknown>) => api.post('/tenants', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/tenants/${id}`, data),
};

export const userApi = {
  list: () => api.get('/users'),
  get: (id: number) => api.get(`/users/${id}`),
  create: (data: Record<string, unknown>) => api.post('/users', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/users/${id}`, data),
};

export const agentApi = {
  list: () => api.get('/agents'),
  get: (id: number) => api.get(`/agents/${id}`),
  create: (data: Record<string, unknown>) => api.post('/agents', data),
};

export const skillGroupApi = {
  list: () => api.get('/skill-groups'),
  get: (id: number) => api.get(`/skill-groups/${id}`),
  create: (data: Record<string, unknown>) => api.post('/skill-groups', data),
  getMembers: (id: number) => api.get(`/skill-groups/${id}/members`),
  addMember: (id: number, data: Record<string, unknown>) => api.post(`/skill-groups/${id}/members`, data),
  removeMember: (id: number, agentId: number) => api.delete(`/skill-groups/${id}/members/${agentId}`),
};

// --- Routing BC ---
export const ivrFlowApi = {
  list: () => api.get('/ivr-flows'),
  get: (id: number) => api.get(`/ivr-flows/${id}`),
  create: (data: Record<string, unknown>) => api.post('/ivr-flows', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/ivr-flows/${id}`, data),
  publish: (id: number) => api.post(`/ivr-flows/${id}/publish`),
  lock: (id: number) => api.post(`/ivr-flows/${id}/lock`),
  unlock: (id: number) => api.post(`/ivr-flows/${id}/unlock`),
  clone: (id: number) => api.post(`/ivr-flows/${id}/clone`),
  versions: (id: number) => api.get(`/ivr-flows/${id}/versions`),
  rollback: (id: number, version: number) => api.post(`/ivr-flows/${id}/rollback/${version}`),
};

// --- Telephony BC ---
export const carrierApi = {
  list: () => api.get('/carriers'),
  create: (data: Record<string, unknown>) => api.post('/carriers', data),
};

export const sipTrunkApi = {
  list: () => api.get('/sip-trunks'),
  create: (data: Record<string, unknown>) => api.post('/sip-trunks', data),
};

export const phoneNumberApi = {
  list: () => api.get('/phone-numbers'),
  create: (data: Record<string, unknown>) => api.post('/phone-numbers', data),
};

// --- Call BC ---
export const callApi = {
  list: (params?: Record<string, unknown>) => api.get('/calls', { params }),
  get: (id: number) => api.get(`/calls/${id}`),
  dial: (data: Record<string, unknown>) => api.post('/calls/dial', data),
  internalDial: (data: Record<string, unknown>) => api.post('/calls/internal-dial', data),
  getEvents: (id: number) => api.get(`/calls/${id}/events`),
  getIvrTracking: (id: number) => api.get(`/calls/${id}/ivr-tracking`),
};

export const callControlApi = {
  answer: (id: number) => api.post(`/calls/${id}/answer`),
  end: (id: number) => api.post(`/calls/${id}/end`),
  hold: (id: number) => api.post(`/calls/${id}/hold`),
  retrieve: (id: number) => api.post(`/calls/${id}/retrieve`),
  sendDTMF: (id: number, data: Record<string, unknown>) => api.post(`/calls/${id}/dtmf`, data),
  blindTransfer: (id: number, data: Record<string, unknown>) => api.post(`/calls/${id}/blind-transfer`, data),
  attendedTransfer: (id: number, data: Record<string, unknown>) => api.post(`/calls/${id}/attended-transfer`, data),
  consult: (id: number, data: Record<string, unknown>) => api.post(`/calls/${id}/consult`, data),
  consultTransfer: (id: number) => api.post(`/calls/${id}/consult-transfer`),
  consultCancel: (id: number) => api.post(`/calls/${id}/consult-cancel`),
  conference: (id: number) => api.post(`/calls/${id}/conference`),
  disposition: (id: number, data: Record<string, unknown>) => api.post(`/calls/${id}/disposition`, data),
  monitor: (id: number, data?: Record<string, unknown>) => api.post(`/calls/${id}/monitor`, data),
  whisper: (id: number, data?: Record<string, unknown>) => api.post(`/calls/${id}/whisper`, data),
  barge: (id: number, data?: Record<string, unknown>) => api.post(`/calls/${id}/barge`, data),
  intercept: (id: number, data?: Record<string, unknown>) => api.post(`/calls/${id}/intercept`, data),
  coach: (id: number, data?: Record<string, unknown>) => api.post(`/calls/${id}/coach`, data),
  inbound: (data: Record<string, unknown>) => api.post('/calls/inbound', data),
  back2back: (data: Record<string, unknown>) => api.post('/calls/back2back', data),
  encrypted: (data: Record<string, unknown>) => api.post('/calls/encrypted', data),
  requestCallback: (data: Record<string, unknown>) => api.post('/callbacks', data),
};

// --- Campaign BC ---
export const campaignApi = {
  list: () => api.get('/campaigns'),
  get: (id: number) => api.get(`/campaigns/${id}`),
  create: (data: Record<string, unknown>) => api.post('/campaigns', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/campaigns/${id}`, data),
  start: (id: number) => api.post(`/campaigns/${id}/start`),
  pause: (id: number) => api.post(`/campaigns/${id}/pause`),
  resume: (id: number) => api.post(`/campaigns/${id}/resume`),
  abort: (id: number) => api.post(`/campaigns/${id}/abort`),
  importCases: (id: number, data: Record<string, unknown>) => api.post(`/campaigns/${id}/cases/import`, data),
  stats: (id: number) => api.get(`/campaigns/${id}/stats`),
};

// --- CRM BC ---
export const customerApi = {
  list: (params?: Record<string, unknown>) => api.get('/customers', { params }),
  get: (id: number) => api.get(`/customers/${id}`),
  create: (data: Record<string, unknown>) => api.post('/customers', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/customers/${id}`, data),
  batchImport: (data: FormData) => api.post('/customers/import', data),
};

// --- Ticket BC ---
export const ticketApi = {
  list: (params?: Record<string, unknown>) => api.get('/tickets', { params }),
  get: (id: number) => api.get(`/tickets/${id}`),
  create: (data: Record<string, unknown>) => api.post('/tickets', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/tickets/${id}`, data),
  addComment: (id: number, data: Record<string, unknown>) => api.post(`/tickets/${id}/comments`, data),
};

export const ticketTemplateApi = {
  list: () => api.get('/ticket-templates'),
  get: (id: number) => api.get(`/ticket-templates/${id}`),
  create: (data: Record<string, unknown>) => api.post('/ticket-templates', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/ticket-templates/${id}`, data),
  publish: (id: number) => api.post(`/ticket-templates/${id}/publish`),
};

// --- IM BC ---
export const imChannelApi = {
  list: () => api.get('/im/channels'),
  create: (data: Record<string, unknown>) => api.post('/im/channels', data),
};

export const imSessionApi = {
  list: (params?: Record<string, unknown>) => api.get('/im/sessions', { params }),
  get: (id: number) => api.get(`/im/sessions/${id}`),
  close: (id: number) => api.post(`/im/sessions/${id}/close`),
  messages: (id: number) => api.get(`/im/sessions/${id}/messages`),
  send: (id: number, data: Record<string, unknown>) => api.post(`/im/sessions/${id}/messages`, data),
};

// --- AI BC ---
export const digitalEmployeeApi = {
  list: () => api.get('/digital-employees'),
  get: (id: number) => api.get(`/digital-employees/${id}`),
  create: (data: Record<string, unknown>) => api.post('/digital-employees', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/digital-employees/${id}`, data),
  createScene: (id: number, data: Record<string, unknown>) => api.post(`/digital-employees/${id}/scenes`, data),
  listScenes: (id: number) => api.get(`/digital-employees/${id}/scenes`),
};

export const qaApi = {
  listRules: () => api.get('/qa/rules'),
  createRule: (data: Record<string, unknown>) => api.post('/qa/rules', data),
  updateRule: (id: number, data: Record<string, unknown>) => api.put(`/qa/rules/${id}`, data),
  deleteRule: (id: number) => api.delete(`/qa/rules/${id}`),
  listSchemes: () => api.get('/qa/schemes'),
  createScheme: (data: Record<string, unknown>) => api.post('/qa/schemes', data),
  runInspection: (data: Record<string, unknown>) => api.post('/qa/run', data),
  listResults: (params?: Record<string, unknown>) => api.get('/qa/results', { params }),
  getResult: (id: number) => api.get(`/qa/results/${id}`),
  appeal: (id: number, data: Record<string, unknown>) => api.post(`/qa/results/${id}/appeal`, data),
  review: (id: number, data: Record<string, unknown>) => api.post(`/qa/results/${id}/review`, data),
};

export const knowledgeApi = {
  listCategories: () => api.get('/knowledge/categories'),
  createCategory: (data: Record<string, unknown>) => api.post('/knowledge/categories', data),
  listArticles: (params?: Record<string, unknown>) => api.get('/knowledge/articles', { params }),
  getArticle: (id: number) => api.get(`/knowledge/articles/${id}`),
  createArticle: (data: Record<string, unknown>) => api.post('/knowledge/articles', data),
  updateArticle: (id: number, data: Record<string, unknown>) => api.put(`/knowledge/articles/${id}`, data),
};

// --- Report BC ---
export const dashboardApi = {
  overview: () => api.get('/dashboard/overview'),
  agents: () => api.get('/dashboard/agents'),
  skillGroups: () => api.get('/dashboard/skill-groups'),
  funnel: () => api.get('/dashboard/funnel'),
  trend: (params?: Record<string, unknown>) => api.get('/dashboard/trend', { params }),
};

export const reportApi = {
  agents: (params?: Record<string, unknown>) => api.get('/reports/agents', { params }),
  groupAgents: (params?: Record<string, unknown>) => api.get('/reports/group-agents', { params }),
  skillGroups: (params?: Record<string, unknown>) => api.get('/reports/skill-groups', { params }),
  b2b: (params?: Record<string, unknown>) => api.get('/reports/back2back', { params }),
  internal: (params?: Record<string, unknown>) => api.get('/reports/internal-calls', { params }),
  statusLog: (params?: Record<string, unknown>) => api.get('/reports/agent-status-log', { params }),
  campaigns: (params?: Record<string, unknown>) => api.get('/reports/campaigns', { params }),
  exportAgents: (params?: Record<string, unknown>) => api.get('/reports/agents/export', { params, responseType: 'blob' }),
  exportSkillGroups: (params?: Record<string, unknown>) => api.get('/reports/skill-groups/export', { params, responseType: 'blob' }),
};

// --- Recording BC ---
export const recordingApi = {
  list: (params?: Record<string, unknown>) => api.get('/recordings', { params }),
  get: (id: number) => api.get(`/recordings/${id}`),
  stream: (id: number) => `/api/v1/recordings/${id}/stream`,
  download: (id: number) => `/api/v1/recordings/${id}/download`,
};

// --- Flash SMS ---
export const flashSmsApi = {
  send: (data: Record<string, unknown>) => api.post('/flash-sms', data),
};

// --- Annotation (Phase 10) ---
export const annotationApi = {
  listTasks: () => api.get('/annotation-tasks'),
  getTask: (id: number) => api.get(`/annotation-tasks/${id}`),
  createTask: (data: Record<string, unknown>) => api.post('/annotation-tasks', data),
  startTask: (id: number) => api.post(`/annotation-tasks/${id}/start`),
  completeTask: (id: number) => api.post(`/annotation-tasks/${id}/complete`),
  cancelTask: (id: number) => api.post(`/annotation-tasks/${id}/cancel`),
  submitAnnotation: (id: number, data: Record<string, unknown>) => api.post(`/annotation-tasks/${id}/annotations`, data),
  listResults: (id: number) => api.get(`/annotation-tasks/${id}/annotations`),
};

export const llmModelApi = {
  list: () => api.get('/llm-models'),
  get: (id: number) => api.get(`/llm-models/${id}`),
  create: (data: Record<string, unknown>) => api.post('/llm-models', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/llm-models/${id}`, data),
  delete: (id: number) => api.delete(`/llm-models/${id}`),
  getDefault: () => api.get('/llm-models/default'),
};

// --- Settings ---
export const agentPresenceApi = {
  list: () => api.get('/agent-presence'),
  changeStatus: (data: Record<string, unknown>) => api.post('/agent-presence/status', data),
};

export const webhookConfigApi = {
  list: () => api.get('/webhook-configs'),
  create: (data: Record<string, unknown>) => api.post('/webhook-configs', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/webhook-configs/${id}`, data),
  delete: (id: number) => api.delete(`/webhook-configs/${id}`),
};

export const csatApi = {
  getConfig: () => api.get('/csat/config'),
  updateConfig: (data: Record<string, unknown>) => api.put('/csat/config', data),
  listResults: (params?: Record<string, unknown>) => api.get('/csat/results', { params }),
};

export const asrHotwordsApi = {
  list: () => api.get('/asr-hotwords'),
  create: (data: Record<string, unknown>) => api.post('/asr-hotwords', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/asr-hotwords/${id}`, data),
  delete: (id: number) => api.delete(`/asr-hotwords/${id}`),
};

export const performanceApi = {
  list: (params?: Record<string, unknown>) => api.get('/performance-scorecards', { params }),
  generate: (data: Record<string, unknown>) => api.post('/performance-scorecards/generate', data),
};

// --- Social Channels ---
export const socialChannelApi = {
  list: () => api.get('/social-channels'),
  create: (data: Record<string, unknown>) => api.post('/social-channels', data),
  update: (id: number, data: Record<string, unknown>) => api.put(`/social-channels/${id}`, data),
  delete: (id: number) => api.delete(`/social-channels/${id}`),
};

// --- Advanced AI ---
export const voiceCloneApi = {
  listTasks: () => api.get('/ai/voice-clone/tasks'),
  createTask: (data: Record<string, unknown>) => api.post('/ai/voice-clone/tasks', data),
  getTask: (id: number) => api.get(`/ai/voice-clone/tasks/${id}`),
};

export const conversationAnalyticsApi = {
  analyze: (data: Record<string, unknown>) => api.post('/ai/conversation-analytics/analyze', data),
};

export const trainingApi = {
  generateQuestions: (data: Record<string, unknown>) => api.post('/ai/training/generate-questions', data),
  evaluate: (data: Record<string, unknown>) => api.post('/ai/training/evaluate', data),
};

// --- Supervisor ---
export const supervisorApi = {
  activeCalls: () => api.get('/supervisor/active-calls'),
};

// --- Webchat ---
export const webchatApi = {
  createSession: (data: Record<string, unknown>) => api.post('/webchat/sessions', data),
  getMessages: (sessionId: string) => api.get(`/webchat/sessions/${sessionId}/messages`),
  sendMessage: (sessionId: string, data: Record<string, unknown>) => api.post(`/webchat/sessions/${sessionId}/messages`, data),
};
