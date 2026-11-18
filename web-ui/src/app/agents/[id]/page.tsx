import AgentStudioClient from '@/components/AgentStudioClient';

export function generateStaticParams() {
  return [
    'CodeGen',
    'product',
    'architect',
    'design',
    'developer',
    'security',
    'qa',
    'performance',
    'devops',
    'compliance',
    'coordinator',
  ].map((id) => ({ id }));
}

export default function AgentPage({ params }: { params: { id: string } }) {
  return <AgentStudioClient id={params.id} />;
}
