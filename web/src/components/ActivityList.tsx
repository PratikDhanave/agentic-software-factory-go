import React from 'react';
import { formatDistanceToNow } from 'date-fns';

interface Activity {
  id: string;
  type: 'tenant_created' | 'agent_registered' | 'service_registered' | 'credential_issued' | 'policy_created';
  description: string;
  timestamp: Date;
  actor: string;
}

const mockActivities: Activity[] = [
  {
    id: '1',
    type: 'tenant_created',
    description: 'Created new tenant "team-platform"',
    timestamp: new Date(Date.now() - 5 * 60000),
    actor: 'admin',
  },
  {
    id: '2',
    type: 'agent_registered',
    description: 'Agent "analyzer-1" registered to team-platform',
    timestamp: new Date(Date.now() - 15 * 60000),
    actor: 'system',
  },
  {
    id: '3',
    type: 'service_registered',
    description: 'Service "code-review" discovered',
    timestamp: new Date(Date.now() - 30 * 60000),
    actor: 'agent',
  },
  {
    id: '4',
    type: 'credential_issued',
    description: 'Credential issued for agent "analyzer-1"',
    timestamp: new Date(Date.now() - 45 * 60000),
    actor: 'system',
  },
  {
    id: '5',
    type: 'policy_created',
    description: 'Policy created for team-platform tenant',
    timestamp: new Date(Date.now() - 60 * 60000),
    actor: 'admin',
  },
];

const activityIcons: Record<Activity['type'], string> = {
  tenant_created: '🏢',
  agent_registered: '👤',
  service_registered: '🔗',
  credential_issued: '🔑',
  policy_created: '📋',
};

export default function ActivityList() {
  return (
    <div className="space-y-3">
      {mockActivities.map((activity) => (
        <div
          key={activity.id}
          className="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-50 border border-gray-100 transition-colors"
        >
          <span className="text-2xl">{activityIcons[activity.type]}</span>
          <div className="flex-1">
            <p className="text-gray-900 font-medium">{activity.description}</p>
            <p className="text-xs text-gray-600">
              {formatDistanceToNow(activity.timestamp, { addSuffix: true })} by {activity.actor}
            </p>
          </div>
        </div>
      ))}
    </div>
  );
}
