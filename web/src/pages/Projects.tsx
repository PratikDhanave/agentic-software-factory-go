import React, { useState } from 'react';
import { Plus, GripVertical, Trash2, CheckCircle2, Clock, AlertCircle } from 'lucide-react';

interface Card {
  id: string;
  title: string;
  description: string;
  priority: 'low' | 'medium' | 'high';
  assignee?: string;
  labels: string[];
  createdAt: Date;
}

interface Column {
  id: 'todo' | 'in-progress' | 'done';
  title: string;
  color: string;
  icon: React.ReactNode;
  cards: Card[];
}

const priorityColors = {
  low: 'bg-blue-100 text-blue-800',
  medium: 'bg-yellow-100 text-yellow-800',
  high: 'bg-red-100 text-red-800',
};

const priorityIcons = {
  low: <Clock size={14} />,
  medium: <AlertCircle size={14} />,
  high: <AlertCircle size={14} className="text-red-600" />,
};

export default function Projects() {
  const [columns, setColumns] = useState<Column[]>([
    {
      id: 'todo',
      title: 'To do',
      color: 'bg-gray-100 border-gray-300',
      icon: <Plus size={20} className="text-gray-600" />,
      cards: [
        {
          id: '1',
          title: 'Implement WebSocket updates',
          description: 'Add real-time updates for dashboard metrics',
          priority: 'high',
          assignee: 'Alice',
          labels: ['feature', 'realtime'],
          createdAt: new Date(),
        },
        {
          id: '2',
          title: 'Add dark mode support',
          description: 'Implement theme toggle with local storage',
          priority: 'medium',
          labels: ['ui', 'enhancement'],
          createdAt: new Date(),
        },
      ],
    },
    {
      id: 'in-progress',
      title: 'In progress',
      color: 'bg-blue-100 border-blue-300',
      icon: <Clock size={20} className="text-blue-600" />,
      cards: [
        {
          id: '3',
          title: 'Complete service discovery page',
          description: 'Build service listing and health monitoring',
          priority: 'high',
          assignee: 'Bob',
          labels: ['feature'],
          createdAt: new Date(),
        },
      ],
    },
    {
      id: 'done',
      title: 'Done',
      color: 'bg-green-100 border-green-300',
      icon: <CheckCircle2 size={20} className="text-green-600" />,
      cards: [
        {
          id: '4',
          title: 'Dashboard implementation',
          description: 'Create main dashboard with stats',
          priority: 'high',
          assignee: 'Carol',
          labels: ['completed'],
          createdAt: new Date(),
        },
        {
          id: '5',
          title: 'Tenant CRUD operations',
          description: 'Implement create, read, update, delete for tenants',
          priority: 'high',
          assignee: 'Alice',
          labels: ['completed'],
          createdAt: new Date(),
        },
      ],
    },
  ]);

  const [draggedCard, setDraggedCard] = useState<{
    cardId: string;
    sourceColumnId: string;
  } | null>(null);

  const handleDragStart = (e: React.DragEvent, cardId: string, columnId: string) => {
    setDraggedCard({ cardId, sourceColumnId: columnId });
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'move';
  };

  const handleDrop = (e: React.DragEvent, targetColumnId: string) => {
    e.preventDefault();
    if (!draggedCard) return;

    const updatedColumns = columns.map((col) => {
      if (col.id === draggedCard.sourceColumnId) {
        return {
          ...col,
          cards: col.cards.filter((card) => card.id !== draggedCard.cardId),
        };
      }
      if (col.id === targetColumnId) {
        const draggedCardData = columns
          .find((c) => c.id === draggedCard.sourceColumnId)
          ?.cards.find((card) => card.id === draggedCard.cardId);
        if (draggedCardData) {
          return {
            ...col,
            cards: [...col.cards, draggedCardData],
          };
        }
      }
      return col;
    });

    setColumns(updatedColumns);
    setDraggedCard(null);
  };

  const handleDeleteCard = (columnId: string, cardId: string) => {
    const updatedColumns = columns.map((col) => {
      if (col.id === columnId) {
        return {
          ...col,
          cards: col.cards.filter((card) => card.id !== cardId),
        };
      }
      return col;
    });
    setColumns(updatedColumns);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Projects</h1>
        <p className="text-gray-600 mt-2">GitHub-style project board for tracking development</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {columns.map((column) => (
          <div
            key={column.id}
            onDragOver={handleDragOver}
            onDrop={(e) => handleDrop(e, column.id)}
            className={`${column.color} border rounded-lg p-4 min-h-96 flex flex-col`}
          >
            <div className="flex items-center gap-2 mb-4 pb-4 border-b border-gray-200">
              {column.icon}
              <h2 className="font-bold text-gray-900">{column.title}</h2>
              <span className="ml-auto bg-white px-2 py-1 rounded text-sm font-medium text-gray-700">
                {column.cards.length}
              </span>
            </div>

            <div className="space-y-3 flex-1 overflow-y-auto">
              {column.cards.map((card) => (
                <div
                  key={card.id}
                  draggable
                  onDragStart={(e) => handleDragStart(e, card.id, column.id)}
                  className="bg-white rounded-lg border border-gray-200 p-3 shadow-sm hover:shadow-md transition-shadow cursor-move group"
                >
                  <div className="flex items-start gap-2 mb-2">
                    <GripVertical size={16} className="text-gray-400 flex-shrink-0 mt-0.5" />
                    <div className="flex-1">
                      <h3 className="font-medium text-gray-900 group-hover:text-blue-600">
                        {card.title}
                      </h3>
                      <p className="text-xs text-gray-600 mt-1">{card.description}</p>
                    </div>
                    <button
                      onClick={() => handleDeleteCard(column.id, card.id)}
                      className="text-gray-400 hover:text-red-600 opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t border-gray-100">
                    <div className="flex gap-1 flex-wrap">
                      {card.labels.map((label) => (
                        <span
                          key={label}
                          className="inline-block bg-blue-50 text-blue-700 text-xs px-2 py-0.5 rounded"
                        >
                          {label}
                        </span>
                      ))}
                    </div>
                    <div className="flex items-center gap-1">
                      <span className={`flex items-center gap-1 text-xs font-medium ${priorityColors[card.priority]}`}>
                        {priorityIcons[card.priority]}
                      </span>
                    </div>
                  </div>

                  {card.assignee && (
                    <div className="mt-2 pt-2 border-t border-gray-100">
                      <span className="text-xs text-gray-600">Assigned to: </span>
                      <span className="text-xs font-medium text-gray-900">{card.assignee}</span>
                    </div>
                  )}
                </div>
              ))}
            </div>

            <button className="mt-4 w-full p-2 text-gray-600 hover:bg-white hover:bg-opacity-50 rounded border-2 border-dashed border-gray-300 hover:border-gray-400 transition-colors flex items-center justify-center gap-2">
              <Plus size={16} />
              <span className="text-sm font-medium">Add card</span>
            </button>
          </div>
        ))}
      </div>

      {/* Stats Footer */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div>
            <p className="text-gray-600 text-sm">Total Cards</p>
            <p className="text-2xl font-bold text-gray-900">
              {columns.reduce((sum, col) => sum + col.cards.length, 0)}
            </p>
          </div>
          <div>
            <p className="text-gray-600 text-sm">To Do</p>
            <p className="text-2xl font-bold text-gray-900">
              {columns.find((c) => c.id === 'todo')?.cards.length}
            </p>
          </div>
          <div>
            <p className="text-gray-600 text-sm">In Progress</p>
            <p className="text-2xl font-bold text-blue-600">
              {columns.find((c) => c.id === 'in-progress')?.cards.length}
            </p>
          </div>
          <div>
            <p className="text-gray-600 text-sm">Completed</p>
            <p className="text-2xl font-bold text-green-600">
              {columns.find((c) => c.id === 'done')?.cards.length}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
