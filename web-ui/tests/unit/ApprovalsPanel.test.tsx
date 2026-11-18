import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ApprovalsPanel from '@/components/ApprovalsPanel';
import * as apiModule from '@/lib/api';

jest.mock('@/lib/api', () => ({
  apiClient: {
    getPendingApprovals: jest.fn(),
    submitApproval: jest.fn(),
  },
}));

describe('ApprovalsPanel', () => {
  const mockApiClient = apiModule.apiClient as jest.Mocked<typeof apiModule.apiClient>;

  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  test('renders approvals panel', () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [],
      total_pending: 0,
    });

    render(<ApprovalsPanel />);

    expect(screen.getByText(/Pending Approvals/)).toBeInTheDocument();
  });

  test('displays empty state when no approvals', async () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [],
      total_pending: 0,
    });

    render(<ApprovalsPanel />);

    await waitFor(() => {
      expect(screen.getByText(/No pending approvals/)).toBeInTheDocument();
    });
  });

  test('displays pending approvals list', async () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [
        {
          session_id: 'sess-1',
          agent_role: 'Developer',
          current_choice: 'implement_fix',
          confidence: 0.65,
          options: ['implement_fix', 'rollback', 'investigate'],
          requires_action: true,
        },
        {
          session_id: 'sess-2',
          agent_role: 'QA',
          current_choice: 'approve',
          confidence: 0.68,
          options: ['approve', 'reject'],
          requires_action: true,
        },
      ],
      total_pending: 2,
    });

    render(<ApprovalsPanel />);

    await waitFor(() => {
      expect(screen.getByText('Developer')).toBeInTheDocument();
      expect(screen.getByText('QA')).toBeInTheDocument();
      expect(screen.getByText(/65% confidence/)).toBeInTheDocument();
      expect(screen.getByText(/68% confidence/)).toBeInTheDocument();
    });
  });

  test('displays approve and reject buttons for each approval', async () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [
        {
          session_id: 'sess-1',
          agent_role: 'Developer',
          current_choice: 'implement_fix',
          confidence: 0.65,
          options: ['implement_fix', 'rollback'],
          requires_action: true,
        },
      ],
      total_pending: 1,
    });

    render(<ApprovalsPanel />);

    await waitFor(() => {
      const approveButtons = screen.getAllByRole('button', { name: /Approve/ });
      const rejectButtons = screen.getAllByRole('button', { name: /Reject/ });
      expect(approveButtons.length).toBeGreaterThan(0);
      expect(rejectButtons.length).toBeGreaterThan(0);
    });
  });

  test('submits approval when approve button is clicked', async () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [
        {
          session_id: 'sess-1',
          agent_role: 'Developer',
          current_choice: 'implement_fix',
          confidence: 0.65,
          options: ['implement_fix', 'rollback'],
          requires_action: true,
        },
      ],
      total_pending: 1,
    });

    mockApiClient.submitApproval.mockResolvedValue({
      session_id: 'sess-1',
      status: 'approved',
      message: 'Approved successfully',
    });

    render(<ApprovalsPanel />);

    await waitFor(() => {
      expect(screen.getByText('Developer')).toBeInTheDocument();
    });

    const approveButton = screen.getAllByRole('button', { name: /Approve/ })[0];
    fireEvent.click(approveButton);

    await waitFor(() => {
      expect(mockApiClient.submitApproval).toHaveBeenCalledWith(
        expect.objectContaining({
          session_id: 'sess-1',
          approved: true,
        })
      );
    });
  });

  test('submits rejection when reject button is clicked', async () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [
        {
          session_id: 'sess-1',
          agent_role: 'QA',
          current_choice: 'approve',
          confidence: 0.68,
          options: ['approve', 'reject'],
          requires_action: true,
        },
      ],
      total_pending: 1,
    });

    mockApiClient.submitApproval.mockResolvedValue({
      session_id: 'sess-1',
      status: 'rejected',
      message: 'Rejected',
    });

    render(<ApprovalsPanel />);

    await waitFor(() => {
      expect(screen.getByText('QA')).toBeInTheDocument();
    });

    const rejectButton = screen.getAllByRole('button', { name: /Reject/ })[0];
    fireEvent.click(rejectButton);

    await waitFor(() => {
      expect(mockApiClient.submitApproval).toHaveBeenCalledWith(
        expect.objectContaining({
          session_id: 'sess-1',
          approved: false,
        })
      );
    });
  });

  test('polls for approvals on interval', async () => {
    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [],
      total_pending: 0,
    });

    render(<ApprovalsPanel />);

    await waitFor(() => {
      expect(mockApiClient.getPendingApprovals).toHaveBeenCalled();
    });

    // Fast-forward 5 seconds (default polling interval)
    jest.advanceTimersByTime(5000);

    await waitFor(() => {
      expect(mockApiClient.getPendingApprovals).toHaveBeenCalledTimes(2);
    });
  });

  test('handles API errors gracefully', async () => {
    mockApiClient.getPendingApprovals.mockRejectedValue(
      new Error('Failed to fetch approvals')
    );

    render(<ApprovalsPanel />);

    await waitFor(() => {
      // Component should still render without crashing
      expect(screen.getByText(/Pending Approvals/)).toBeInTheDocument();
    });
  });

  test('displays loading state initially', () => {
    mockApiClient.getPendingApprovals.mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );

    render(<ApprovalsPanel />);

    expect(screen.getByText(/Loading/)).toBeInTheDocument();
  });

  test('updates approvals when data changes', async () => {
    const { rerender } = render(<ApprovalsPanel />);

    mockApiClient.getPendingApprovals.mockResolvedValue({
      approvals: [
        {
          session_id: 'sess-1',
          agent_role: 'Developer',
          current_choice: 'implement',
          confidence: 0.70,
          options: ['implement', 'skip'],
          requires_action: true,
        },
      ],
      total_pending: 1,
    });

    jest.advanceTimersByTime(5000);

    await waitFor(() => {
      expect(screen.getByText('Developer')).toBeInTheDocument();
    });
  });
});
