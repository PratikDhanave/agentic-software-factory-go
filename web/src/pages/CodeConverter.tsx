import React, { useState, useEffect } from 'react';
import { Plus, Copy, Download, Zap, ChevronDown } from 'lucide-react';
import { client, ConversionTask } from '../api/client';

const languageLabels = {
  swift: 'iOS (Swift)',
  kotlin: 'Kotlin (Android)',
};

const syntaxHighlight = (code: string, language: string) => {
  // Simple keyword highlighting for demo
  const keywords: { [key: string]: string[] } = {
    swift: ['func', 'class', 'struct', 'var', 'let', 'if', 'for', 'return', 'import'],
    kotlin: ['fun', 'class', 'var', 'val', 'if', 'for', 'return', 'import', 'suspend'],
    dart: ['void', 'class', 'var', 'final', 'if', 'for', 'return', 'import', 'async', 'await'],
  };

  let highlighted = code;
  const lang = language as keyof typeof keywords;

  if (keywords[lang]) {
    keywords[lang].forEach((keyword) => {
      const regex = new RegExp(`\\b${keyword}\\b`, 'g');
      highlighted = highlighted.replace(regex, `<span class="text-blue-600 font-semibold">${keyword}</span>`);
    });
  }

  return highlighted;
};

const CodeBlock: React.FC<{ code: string; language: string; title: string; onCopy?: () => void }> = ({
  code,
  language,
  title,
  onCopy,
}) => {
  return (
    <div className="bg-gray-900 rounded-lg overflow-hidden border border-gray-700">
      <div className="bg-gray-800 px-4 py-3 border-b border-gray-700 flex items-center justify-between">
        <span className="text-gray-300 font-medium">{title}</span>
        {onCopy && (
          <button
            onClick={onCopy}
            className="text-gray-400 hover:text-white transition-colors flex items-center gap-1"
          >
            <Copy size={14} />
            <span className="text-xs">Copy</span>
          </button>
        )}
      </div>
      <pre className="p-4 text-sm text-gray-100 overflow-x-auto font-mono">
        <code>{code}</code>
      </pre>
    </div>
  );
};

export default function CodeConverter() {
  const [sourceLanguage, setSourceLanguage] = useState<'swift' | 'kotlin'>('swift');
  const [sourceCode, setSourceCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [conversions, setConversions] = useState<ConversionTask[]>([
    {
      id: '1',
      sourceLanguage: 'swift',
      sourceCode: `import UIKit

class ViewController: UIViewController {
    @IBOutlet weak var label: UILabel!

    override func viewDidLoad() {
        super.viewDidLoad()
        label.text = "Hello, World!"
    }
}`,
      flutterCode: `import 'package:flutter/material.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        body: Center(
          child: Text('Hello, World!'),
        ),
      ),
    );
  }
}`,
      status: 'completed',
      assignedAgent: 'converter-1',
      createdAt: new Date(Date.now() - 24 * 60 * 60000).toISOString(),
      completedAt: new Date(Date.now() - 23 * 60 * 60000).toISOString(),
    },
  ]);

  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    loadConversions();
    const interval = setInterval(loadConversions, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadConversions = async () => {
    try {
      const data = await client.listConversions();
      setConversions(data);
    } catch (error) {
      console.error('Failed to load conversions:', error);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!sourceCode.trim()) return;

    setSubmitting(true);

    try {
      const newTask = await client.submitCodeConversion(sourceLanguage, sourceCode);
      setConversions([newTask, ...conversions]);
      setSourceCode('');

      // Simulate agent pickup after 1 second
      setTimeout(() => {
        client.updateConversion(newTask.id, {
          status: 'converting',
          assignedAgent: 'converter-1',
        }).catch(console.error);
      }, 1000);
    } catch (error) {
      console.error('Failed to submit conversion:', error);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDownload = (code: string, filename: string) => {
    const element = document.createElement('a');
    element.setAttribute('href', `data:text/plain;charset=utf-8,${encodeURIComponent(code)}`);
    element.setAttribute('download', filename);
    element.style.display = 'none';
    document.body.appendChild(element);
    element.click();
    document.body.removeChild(element);
  };

  const handleCopy = (code: string) => {
    navigator.clipboard.writeText(code);
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900">Code Converter</h1>
        <p className="text-gray-600 mt-2">Convert iOS and Kotlin code to Flutter automatically</p>
      </div>

      {/* Submission Form */}
      <div className="bg-white rounded-lg shadow-sm p-6">
        <h2 className="text-lg font-bold text-gray-900 mb-4">Submit Code for Conversion</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Source Language</label>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => setSourceLanguage('swift')}
                className={`flex-1 px-4 py-2 rounded-lg border-2 transition-colors ${
                  sourceLanguage === 'swift'
                    ? 'bg-blue-50 border-blue-500 text-blue-700 font-medium'
                    : 'border-gray-200 text-gray-700 hover:border-gray-300'
                }`}
              >
                iOS (Swift)
              </button>
              <button
                type="button"
                onClick={() => setSourceLanguage('kotlin')}
                className={`flex-1 px-4 py-2 rounded-lg border-2 transition-colors ${
                  sourceLanguage === 'kotlin'
                    ? 'bg-blue-50 border-blue-500 text-blue-700 font-medium'
                    : 'border-gray-200 text-gray-700 hover:border-gray-300'
                }`}
              >
                Kotlin (Android)
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Source Code</label>
            <textarea
              value={sourceCode}
              onChange={(e) => setSourceCode(e.target.value)}
              placeholder={`Paste your ${languageLabels[sourceLanguage]} code here...`}
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm h-48"
            />
          </div>

          <div className="flex gap-3">
            <button
              type="submit"
              disabled={submitting || !sourceCode.trim()}
              className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 disabled:bg-gray-400 transition-colors flex items-center gap-2 font-medium"
            >
              <Zap size={18} />
              Convert to Flutter
            </button>
            {sourceCode && (
              <button
                type="button"
                onClick={() => setSourceCode('')}
                className="bg-gray-200 text-gray-700 px-6 py-2 rounded-lg hover:bg-gray-300 transition-colors"
              >
                Clear
              </button>
            )}
          </div>
        </form>
      </div>

      {/* Conversion History */}
      <div className="space-y-4">
        <h2 className="text-lg font-bold text-gray-900">Conversion History</h2>

        {conversions.map((task) => (
          <div key={task.id} className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            {/* Header */}
            <div className="bg-gray-50 px-6 py-4 border-b border-gray-200 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`w-3 h-3 rounded-full ${
                  task.status === 'completed' ? 'bg-green-500' :
                  task.status === 'converting' ? 'bg-blue-500' :
                  task.status === 'failed' ? 'bg-red-500' : 'bg-gray-500'
                }`}></div>
                <div>
                  <p className="font-medium text-gray-900">{languageLabels[task.sourceLanguage]} → Flutter</p>
                  <p className="text-xs text-gray-600">
                    {task.assignedAgent && `Converted by ${task.assignedAgent}`}
                  </p>
                </div>
              </div>
              <div className="text-right">
                <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                  task.status === 'completed' ? 'bg-green-100 text-green-800' :
                  task.status === 'converting' ? 'bg-blue-100 text-blue-800' :
                  task.status === 'failed' ? 'bg-red-100 text-red-800' : 'bg-gray-100 text-gray-800'
                }`}>
                  {task.status.charAt(0).toUpperCase() + task.status.slice(1)}
                </span>
              </div>
            </div>

            {/* Code Comparison */}
            {task.status === 'completed' && task.flutterCode && (
              <div className="p-6">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                  <CodeBlock
                    code={task.sourceCode}
                    language={task.sourceLanguage}
                    title={`${languageLabels[task.sourceLanguage]} Code`}
                    onCopy={() => handleCopy(task.sourceCode)}
                  />
                  <CodeBlock
                    code={task.flutterCode}
                    language="dart"
                    title="Flutter (Dart) Code"
                    onCopy={() => handleCopy(task.flutterCode!)}
                  />
                </div>

                <div className="mt-4 flex gap-3">
                  <button
                    onClick={() => handleDownload(task.flutterCode!, 'converted.dart')}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-colors"
                  >
                    <Download size={16} />
                    Download Flutter Code
                  </button>
                  <button
                    onClick={() => handleCopy(task.flutterCode!)}
                    className="flex items-center gap-2 px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors"
                  >
                    <Copy size={16} />
                    Copy to Clipboard
                  </button>
                </div>
              </div>
            )}

            {/* Pending/Converting State */}
            {(task.status === 'pending' || task.status === 'converting') && (
              <div className="p-6">
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <div className="animate-spin">
                      <Zap size={16} className="text-blue-600" />
                    </div>
                    <p className="text-blue-800 font-medium">
                      {task.status === 'pending' ? 'Waiting for agent...' : 'Converting...'}
                    </p>
                  </div>
                  <p className="text-sm text-blue-700">
                    {task.sourceCode.split('\n').length} lines of code submitted
                  </p>
                </div>
              </div>
            )}

            {/* Failed State */}
            {task.status === 'failed' && (
              <div className="p-6">
                <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                  <p className="text-red-800 font-medium">Conversion failed</p>
                  <p className="text-sm text-red-700 mt-1">Please try again or contact support</p>
                </div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Quick Tips */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
        <h3 className="font-bold text-blue-900 mb-3">💡 Tips for Best Results</h3>
        <ul className="space-y-2 text-sm text-blue-800">
          <li>• Include complete code blocks with all dependencies and imports</li>
          <li>• Add comments explaining complex logic for better conversion accuracy</li>
          <li>• For large projects, convert them file-by-file or by feature</li>
          <li>• Review the converted code for Flutter best practices</li>
          <li>• Test the converted code thoroughly in your Flutter project</li>
        </ul>
      </div>
    </div>
  );
}
