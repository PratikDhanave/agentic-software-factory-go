import React from 'react';

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ReactNode;
  trend: string;
  color: 'blue' | 'green' | 'purple' | 'orange';
}

const colorClasses = {
  blue: 'bg-blue-50 border-blue-200',
  green: 'bg-green-50 border-green-200',
  purple: 'bg-purple-50 border-purple-200',
  orange: 'bg-orange-50 border-orange-200',
};

export default function StatCard({
  title,
  value,
  icon,
  trend,
  color,
}: StatCardProps) {
  return (
    <div className={`${colorClasses[color]} rounded-lg border p-6`}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-gray-600 text-sm font-medium">{title}</p>
          <p className="text-3xl font-bold text-gray-900 mt-2">{value}</p>
          <p className="text-gray-600 text-xs mt-2">{trend}</p>
        </div>
        <div className="p-3 bg-white rounded-lg">{icon}</div>
      </div>
    </div>
  );
}
