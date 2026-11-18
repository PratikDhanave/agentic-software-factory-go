import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import WorkflowGenerator from '@/components/WorkflowGenerator';
import * as apiModule from '@/lib/api';

// Mock the API client
jest.mock('@/lib/api', () => ({
  apiClient: {
    generateWorkflow: jest.fn(),
  },
}));

describe('WorkflowGenerator', () => {
  const mockApiClient = apiModule.apiClient as jest.Mocked<typeof apiModule.apiClient>;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  test('renders workflow generator form', () => {
    render(<WorkflowGenerator />);

    expect(screen.getByText('Generate Workflow')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Fix authentication/)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/Describe the issue/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Generate Workflow/ })).toBeInTheDocument();
  });

  test('validates empty form submission', async () => {
    render(<WorkflowGenerator />);

    const button = screen.getByRole('button', { name: /Generate Workflow/ });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/Please fill in both title and description/)).toBeInTheDocument();
    });
  });

  test('validates title is required', async () => {
    render(<WorkflowGenerator />);

    const descriptionInput = screen.getByPlaceholderText(/Describe the issue/);
    await userEvent.type(descriptionInput, 'Some description');

    const button = screen.getByRole('button', { name: /Generate Workflow/ });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/Please fill in both title and description/)).toBeInTheDocument();
    });
  });

  test('validates description is required', async () => {
    render(<WorkflowGenerator />);

    const titleInput = screen.getByPlaceholderText(/Fix authentication/);
    await userEvent.type(titleInput, 'Auth Bug');

    const button = screen.getByRole('button', { name: /Generate Workflow/ });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/Please fill in both title and description/)).toBeInTheDocument();
    });
  });

  test('successfully generates workflow with valid input', async () => {
    const mockResponse = {
      workflow_id: 'wf-123',
      workflow_name: 'Test Workflow',
      decisions: [
        {
          agent_role: 'Developer',
          decision: 'Fix the bug',
          confidence: 0.95,
          reasoning: 'This is the standard approach',
        },
      ],
    };

    mockApiClient.generateWorkflow.mockResolvedValue(mockResponse);

    render(<WorkflowGenerator />);

    const titleInput = screen.getByPlaceholderText(/Fix authentication/);
    const descInput = screen.getByPlaceholderText(/Describe the issue/);

    await userEvent.type(titleInput, 'Auth Bug');
    await userEvent.type(descInput, 'OAuth not working');

    const button = screen.getByRole('button', { name: /Generate Workflow/ });
    fireEvent.click(button);

    await waitFor(() => {
      expect(screen.getByText(/Workflow Generated/)).toBeInTheDocument();
      expect(screen.getByText(/wf-123/)).toBeInTheDocument();
    });
  });

  test('displays agent decisions with confidence scores', async () => {
    const mockResponse = {
      workflow_id: 'wf-123',
      workflow_name: 'Test Workflow',
      decisions: [
        {
          agent_role: 'Architect',
          decision: 'Implement OAuth refresh',
          confidence: 0.95,
          reasoning: 'Token refresh is needed',
        },
        {
          agent_role: 'Developer',
          decision: 'Add retry logic',
          confidence: 0.92,
          reasoning: 'Exponential backoff pattern',
        },
      ],
    };

    mockApiClient.generateWorkflow.mockResolvedValue(mockResponse);

    render(<WorkflowGenerator />);

    await userEvent.type(screen.getByPlaceholderText(/Fix authentication/), 'Auth Bug');
    await userEvent.type(screen.getByPlaceholderText(/Describe the issue/), 'OAuth not working');

    fireEvent.click(screen.getByRole('button', { name: /Generate Workflow/ }));

    await waitFor(() => {
      expect(screen.getByText('Architect')).toBeInTheDocument();
      expect(screen.getByText('Developer')).toBeInTheDocument();
      expect(screen.getByText('95% confidence')).toBeInTheDocument();
      expect(screen.getByText('92% confidence')).toBeInTheDocument();
    });
  });

  test('handles API errors gracefully', async () => {
    mockApiClient.generateWorkflow.mockRejectedValue(
      new Error('Network error')
    );

    render(<WorkflowGenerator />);

    await userEvent.type(screen.getByPlaceholderText(/Fix authentication/), 'Auth Bug');
    await userEvent.type(screen.getByPlaceholderText(/Describe the issue/), 'OAuth not working');

    fireEvent.click(screen.getByRole('button', { name: /Generate Workflow/ }));

    await waitFor(() => {
      expect(screen.getByText(/Network error/)).toBeInTheDocument();
    });
  });

  test('shows loading state while generating', async () => {
    mockApiClient.generateWorkflow.mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );

    render(<WorkflowGenerator />);

    await userEvent.type(screen.getByPlaceholderText(/Fix authentication/), 'Auth Bug');
    await userEvent.type(screen.getByPlaceholderText(/Describe the issue/), 'OAuth not working');

    fireEvent.click(screen.getByRole('button', { name: /Generate Workflow/ }));

    expect(screen.getByText(/Generating.../)).toBeInTheDocument();
  });

  test('disables button while loading', async () => {
    mockApiClient.generateWorkflow.mockImplementation(
      () => new Promise(resolve => setTimeout(resolve, 100))
    );

    render(<WorkflowGenerator />);

    await userEvent.type(screen.getByPlaceholderText(/Fix authentication/), 'Auth Bug');
    await userEvent.type(screen.getByPlaceholderText(/Describe the issue/), 'OAuth not working');

    const button = screen.getByRole('button', { name: /Generate Workflow/ });
    fireEvent.click(button);

    expect(button).toBeDisabled();
  });

  test('clears error when form is resubmitted', async () => {
    mockApiClient.generateWorkflow
      .mockRejectedValueOnce(new Error('First error'))
      .mockResolvedValueOnce({
        workflow_id: 'wf-456',
        workflow_name: 'Success',
        decisions: [],
      });

    render(<WorkflowGenerator />);

    // First submission - error
    await userEvent.type(screen.getByPlaceholderText(/Fix authentication/), 'Bug 1');
    await userEvent.type(screen.getByPlaceholderText(/Describe the issue/), 'Description 1');
    fireEvent.click(screen.getByRole('button', { name: /Generate Workflow/ }));

    await waitFor(() => {
      expect(screen.getByText(/First error/)).toBeInTheDocument();
    });

    // Clear and retry
    const titleInput = screen.getByPlaceholderText(/Fix authentication/) as HTMLInputElement;
    await userEvent.clear(titleInput);
    await userEvent.type(titleInput, 'Bug 2');

    fireEvent.click(screen.getByRole('button', { name: /Generate Workflow/ }));

    await waitFor(() => {
      expect(screen.queryByText(/First error/)).not.toBeInTheDocument();
      expect(screen.getByText(/Workflow Generated/)).toBeInTheDocument();
    });
  });
});
