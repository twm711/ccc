import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { lazy, Suspense } from 'react';
import ErrorBoundary from './components/common/ErrorBoundary';
import AppLayout from './components/layout/AppLayout';
import Login from './pages/Login';
import { useAuthStore } from './stores/auth';

// Lazy-load every route page. Without this every chunk lands in the
// initial bundle, which makes the first paint slow especially on first
// login. React.lazy + Suspense splits each page into its own chunk.
const DashboardPage = lazy(() => import('./pages/dashboard/DashboardPage'));
const MyWorkbenchPage = lazy(() => import('./pages/dashboard/MyWorkbenchPage'));
const AgentListPage = lazy(() => import('./pages/agents/AgentListPage'));
const SkillGroupPage = lazy(() => import('./pages/skill-groups/SkillGroupPage'));
const TenantListPage = lazy(() => import('./pages/tenants/TenantListPage'));
const IvrFlowListPage = lazy(() => import('./pages/ivr/IvrFlowListPage'));
const PhoneNumberPage = lazy(() => import('./pages/phone-numbers/PhoneNumberPage'));
const CallRecordPage = lazy(() => import('./pages/call-records/CallRecordPage'));
const VoicemailPage = lazy(() => import('./pages/voicemails/VoicemailPage'));
const CampaignPage = lazy(() => import('./pages/campaigns/CampaignPage'));
const ImSessionPage = lazy(() => import('./pages/im/ImSessionPage'));
const ImChannelPage = lazy(() => import('./pages/im/ImChannelPage'));
const CustomerPage = lazy(() => import('./pages/crm/CustomerPage'));
const TicketListPage = lazy(() => import('./pages/tickets/TicketListPage'));
const TicketTemplatePage = lazy(() => import('./pages/tickets/TicketTemplatePage'));
const KnowledgePage = lazy(() => import('./pages/ai/KnowledgePage'));
const DigitalEmployeePage = lazy(() => import('./pages/ai/DigitalEmployeePage'));
const QaPage = lazy(() => import('./pages/ai/QaPage'));
const AsrHotwordsPage = lazy(() => import('./pages/ai/AsrHotwordsPage'));
const PerformancePage = lazy(() => import('./pages/ai/PerformancePage'));
const AnnotationPage = lazy(() => import('./pages/ai/AnnotationPage'));
const LlmModelPage = lazy(() => import('./pages/ai/LlmModelPage'));
const AdvancedAiPage = lazy(() => import('./pages/ai/AdvancedAiPage'));
const AgentReportPage = lazy(() => import('./pages/reports/AgentReportPage'));
const GroupAgentReportPage = lazy(() => import('./pages/reports/GroupAgentReportPage'));
const SkillGroupReportPage = lazy(() => import('./pages/reports/SkillGroupReportPage'));
const B2BReportPage = lazy(() => import('./pages/reports/B2BReportPage'));
const InternalCallReportPage = lazy(() => import('./pages/reports/InternalCallReportPage'));
const StatusLogPage = lazy(() => import('./pages/reports/StatusLogPage'));
const CsatReportPage = lazy(() => import('./pages/reports/CsatReportPage'));
const CampaignReportPage = lazy(() => import('./pages/reports/CampaignReportPage'));
const TenantSettingsPage = lazy(() => import('./pages/settings/TenantSettingsPage'));
const BreakReasonsPage = lazy(() => import('./pages/settings/BreakReasonsPage'));
const DispositionCodesPage = lazy(() => import('./pages/settings/DispositionCodesPage'));
const BusinessHoursPage = lazy(() => import('./pages/settings/BusinessHoursPage'));
const CustomFieldsPage = lazy(() => import('./pages/settings/CustomFieldsPage'));
const CallTagsPage = lazy(() => import('./pages/settings/CallTagsPage'));
const ScreenPopPage = lazy(() => import('./pages/settings/ScreenPopPage'));
const WebhookPage = lazy(() => import('./pages/settings/WebhookPage'));
const SmsConfigPage = lazy(() => import('./pages/settings/SmsConfigPage'));
const QuickRepliesPage = lazy(() => import('./pages/settings/QuickRepliesPage'));
const CsatConfigPage = lazy(() => import('./pages/settings/CsatConfigPage'));
const AudioFilesPage = lazy(() => import('./pages/settings/AudioFilesPage'));
const AuditLogPage = lazy(() => import('./pages/settings/AuditLogPage'));
const SocialChannelPage = lazy(() => import('./pages/im/SocialChannelPage'));
const CampaignLiveDashboard = lazy(() => import('./pages/campaigns/CampaignLiveDashboard'));

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuthStore();
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

const Loading = () => (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 300 }}>
    <Spin size="large" />
  </div>
);

export default function App() {
  return (
    <ErrorBoundary>
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Suspense fallback={<Loading />}>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
            <Route index element={<Navigate to="/dashboard" replace />} />
            <Route path="dashboard" element={<DashboardPage />} />
            <Route path="me" element={<MyWorkbenchPage />} />
            {/* Agent Management */}
            <Route path="agents" element={<AgentListPage />} />
            <Route path="skill-groups" element={<SkillGroupPage />} />
            {/* Flow Management */}
            <Route path="ivr" element={<IvrFlowListPage />} />
            <Route path="audio-files" element={<AudioFilesPage />} />
            {/* Telephony */}
            <Route path="phone-numbers" element={<PhoneNumberPage />} />
            <Route path="call-records" element={<CallRecordPage />} />
            <Route path="voicemails" element={<VoicemailPage />} />
            {/* Campaign */}
            <Route path="campaigns" element={<CampaignPage />} />
            <Route path="campaigns/live" element={<CampaignLiveDashboard />} />
            {/* IM */}
            <Route path="im/sessions" element={<ImSessionPage />} />
            <Route path="im/channels" element={<ImChannelPage />} />
            <Route path="im/social" element={<SocialChannelPage />} />
            {/* Business */}
            <Route path="crm/customers" element={<CustomerPage />} />
            <Route path="tickets" element={<TicketListPage />} />
            <Route path="ticket-templates" element={<TicketTemplatePage />} />
            <Route path="knowledge" element={<KnowledgePage />} />
            {/* Reports */}
            <Route path="reports/agents" element={<AgentReportPage />} />
            <Route path="reports/group-agents" element={<GroupAgentReportPage />} />
            <Route path="reports/skill-groups" element={<SkillGroupReportPage />} />
            <Route path="reports/b2b" element={<B2BReportPage />} />
            <Route path="reports/internal" element={<InternalCallReportPage />} />
            <Route path="reports/status-log" element={<StatusLogPage />} />
            <Route path="reports/csat" element={<CsatReportPage />} />
            <Route path="reports/campaigns" element={<CampaignReportPage />} />
            {/* AI */}
            <Route path="ai/digital-employees" element={<DigitalEmployeePage />} />
            <Route path="ai/qa" element={<QaPage />} />
            <Route path="ai/hotwords" element={<AsrHotwordsPage />} />
            <Route path="ai/performance" element={<PerformancePage />} />
            <Route path="ai/annotations" element={<AnnotationPage />} />
            <Route path="ai/llm-models" element={<LlmModelPage />} />
            <Route path="ai/advanced" element={<AdvancedAiPage />} />
            {/* Settings */}
            <Route path="settings/tenant" element={<TenantSettingsPage />} />
            <Route path="settings/break-reasons" element={<BreakReasonsPage />} />
            <Route path="settings/disposition-codes" element={<DispositionCodesPage />} />
            <Route path="settings/business-hours" element={<BusinessHoursPage />} />
            <Route path="settings/custom-fields" element={<CustomFieldsPage />} />
            <Route path="settings/call-tags" element={<CallTagsPage />} />
            <Route path="settings/screen-pop" element={<ScreenPopPage />} />
            <Route path="settings/webhooks" element={<WebhookPage />} />
            <Route path="settings/sms" element={<SmsConfigPage />} />
            <Route path="settings/quick-replies" element={<QuickRepliesPage />} />
            <Route path="settings/csat" element={<CsatConfigPage />} />
            {/* Platform */}
            <Route path="tenants" element={<TenantListPage />} />
            <Route path="audit-logs" element={<AuditLogPage />} />
          </Route>
        </Routes>
        </Suspense>
      </BrowserRouter>
    </ConfigProvider>
    </ErrorBoundary>
  );
}
