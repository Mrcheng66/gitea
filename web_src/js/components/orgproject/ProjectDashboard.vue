<script setup lang="ts">
import type {OrgProjectSummary} from './types.ts';

const props = defineProps<{
  summary: OrgProjectSummary,
  baseLink: string,
  labels: {
    blocked: string,
    overdue: string,
    dueSoon: string,
    averageProgress: string,
    blockedDescription: string,
    overdueDescription: string,
    dueSoonDescription: string,
    averageProgressDescription: string,
  },
}>();

const cards = [
  {key: 'blocked', value: () => props.summary.blocked, label: props.labels.blocked, description: props.labels.blockedDescription, href: `${props.baseLink}?filter_risk=blocked`, tone: 'danger'},
  {key: 'overdue', value: () => props.summary.overdue, label: props.labels.overdue, description: props.labels.overdueDescription, href: `${props.baseLink}?due=overdue`, tone: 'danger'},
  {key: 'due-soon', value: () => props.summary.due_soon, label: props.labels.dueSoon, description: props.labels.dueSoonDescription, href: `${props.baseLink}?due=week`, tone: 'warning'},
  {key: 'average-progress', value: () => `${Math.round(props.summary.average_progress)}%`, label: props.labels.averageProgress, description: props.labels.averageProgressDescription, tone: 'neutral'},
];
</script>

<template>
  <div class="org-project-summary-grid">
    <component
      :is="card.href ? 'a' : 'section'"
      v-for="card in cards"
      :key="card.key"
      class="org-project-summary-card"
      :class="`is-${card.tone}`"
      :href="card.href"
    >
      <strong>{{ card.value() }}</strong>
      <span><b>{{ card.label }}</b><small>{{ card.description }}</small></span>
    </component>
  </div>
</template>
