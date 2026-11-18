'use client';

import FactoryWorkbench from './ide/FactoryWorkbench';

export default function FactoryAssemblyLine({ onRunStateChange }: { onRunStateChange?: () => void }) {
  return <FactoryWorkbench onRunStateChange={onRunStateChange} />;
}
