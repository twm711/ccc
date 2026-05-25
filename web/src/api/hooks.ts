import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { agentApi, tenantApi, skillGroupApi, callApi, dashboardApi, customerApi, ticketApi } from './endpoints';

export function useAgents() {
  return useQuery({ queryKey: ['agents'], queryFn: () => agentApi.list().then(r => r.data) });
}

export function useAgent(id: number) {
  return useQuery({ queryKey: ['agents', id], queryFn: () => agentApi.get(id).then(r => r.data), enabled: id > 0 });
}

export function useCreateAgent() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (data: Record<string, unknown>) => agentApi.create(data), onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }) });
}

export function useTenants() {
  return useQuery({ queryKey: ['tenants'], queryFn: () => tenantApi.list().then(r => r.data) });
}

export function useSkillGroups() {
  return useQuery({ queryKey: ['skill-groups'], queryFn: () => skillGroupApi.list().then(r => r.data) });
}

export function useCalls() {
  return useQuery({ queryKey: ['calls'], queryFn: () => callApi.list().then(r => r.data) });
}

export function useDashboard() {
  return useQuery({ queryKey: ['dashboard'], queryFn: () => dashboardApi.get().then(r => r.data), refetchInterval: 5000 });
}

export function useCustomers() {
  return useQuery({ queryKey: ['customers'], queryFn: () => customerApi.list().then(r => r.data) });
}

export function useTickets() {
  return useQuery({ queryKey: ['tickets'], queryFn: () => ticketApi.list().then(r => r.data) });
}

export function useCreateTicket() {
  const qc = useQueryClient();
  return useMutation({ mutationFn: (data: Record<string, unknown>) => ticketApi.create(data), onSuccess: () => qc.invalidateQueries({ queryKey: ['tickets'] }) });
}
