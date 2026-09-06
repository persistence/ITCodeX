import { Navigate, Route, Routes } from 'react-router-dom'
import { CollectionsPage } from '@/pages/collections/CollectionsPage'
import { CreateCollectionPage } from '@/pages/collections/CreateCollectionPage'
import { CollectionStudio } from '@/pages/studio/CollectionStudio'
import { ScriptsPage } from '@/pages/scripts/ScriptsPage'
import { SettingsPage } from '@/pages/settings/SettingsPage'

export function AppRouter() {
  return (
    <Routes>
      <Route path="/" element={<CollectionsPage />} />
      <Route path="/collections/new" element={<CreateCollectionPage />} />
      <Route path="/collections/:name/*" element={<CollectionStudio />} />
      <Route path="/scripts" element={<ScriptsPage />} />
      <Route path="/settings" element={<SettingsPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
