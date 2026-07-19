import { createContext, useContext } from 'react'
import type { Topic } from './useTopics'

/** Entry points the dashboard subtree uses to open the topic popups. */
export interface TopicModalsApi {
  openDetail: (topic: Topic) => void
  openCreate: () => void
}

export const TopicModalsContext = createContext<TopicModalsApi | null>(null)

export function useTopicModals(): TopicModalsApi {
  const ctx = useContext(TopicModalsContext)
  if (!ctx) throw new Error('useTopicModals must be used within <ModalHost>')
  return ctx
}
