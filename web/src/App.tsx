import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import AppLayout from './components/layout/AppLayout';
import Login from './pages/Login';
import DashboardPage from './pages/dashboard/DashboardPage';
import MyWorkbenchPage from './pages/dashboard/MyWorkbenchPage';
import AgentListPage from './pages/agents/AgentListPage';
import SkillGroupPage from './pages/skill-groups/SkillGroupPage';
import TenantListPage from './pages/tenants/TenantListPage';
import IvrFlowListPage from './pages/ivr/IvrFlowListPage';
import PhoneNumberPage from './pages/phone-numbers/PhoneNumberPage';
import CallRecordPage from './pages/call-records/CallRecordPage';
import VoicemailPage from './pages/voicemails/VoicemailPage';
import CampaignPage from './pages/campaigns/CampaignPage';
import ImSessionPage from './pages/im/ImSessionPage';
import ImChannelPage from './pages/im/ImChannelPage';
import CustomerPage from './pages/crm/CustomerPage';
import TicketListPage from './pages/tickets/TicketListPage';
import TicketTemplatePage from './pages/tickets/TicketTemplatePage';
import KnowledgePage from './pages/ai/KnowledgePage';
import DigitalEmployeePage from './pages/ai/DigitalEmployeePage';
import QaPage from './pages/ai/QaPage';
import AsrHotwordsPage from './pages/ai/AsrHotwordsPage';
import PerformancePage from './pages/ai/PerformancePage';
import AnnotationPage from './pages/ai/AnnotationPage';
import LlmModelPage from './pages/ai/LlmModelPage';
import AdvancedAiPage from './pages/ai/AdvancedAiPage';
import AgentReportPage from './pages/reports/AgentReportPage';
import GroupAgentReportPage from './pages/reports/GroupAgentReportPage';
import SkillGroupReportPage from './pages/reports/SkillGroupReportPage';
import B2BReportPage from './pages/reports/B2BReportPage';
import InternalCallReportPage from './pages/reports/InternalCallReportPage';
import StatusLogPage from './pages/reports/StatusLogPage';
import CsatReportPage from './pages/reports/CsatReportPage';
import CampaignReportPage from './pages/reports/CampaignReportPage';
import TenantSettingsPage from './pages/settings/TenantSettingsPage';
import BreakReasonsPage from './pages/settings/BreakReasonsPage';
import DispositionCodesPage from './pages/settings/DispositionCodesPage';
import BusinessHoursPage from './pages/settings/BusinessHoursPage';
import CustomFieldsPage from './pages/settings/CustomFieldsPage';
import CallTagsPage from './pages/settings/CallTagsPage';
import ScreenPopPage from './pages/settings/ScreenPopPage';
import WebhookPage from './pages/settings/WebhookPage';
import SmsConfigPage from './pages/settings/SmsConfigPage';
import QuickRepliesPage from './pages/settings/QuickRepliesPage';
import CsatConfigPage from './pages/settings/CsatConfigPage';
import AudioFilesPage from './pages/settings/AudioFilesPage';
import AuditLogPage from './pages/settings/AuditLogPage';
import SocialChannelPage from './pages/im/SocialChannelPage';
import CampaignLiveDashboard from './pages/campaigns/CampaignLiveDashboard';
import { useAuthStore } from './stores/auth';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { token } = useAuthStore();
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
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
      </BrowserRouter>
    </ConfigProvider>
  );
}
