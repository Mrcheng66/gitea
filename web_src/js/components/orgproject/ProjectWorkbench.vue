<script setup lang="ts">
import {reactive} from 'vue';
import type {ProjectWorkbenchProject, ProjectWorkbenchResult} from './types.ts';

const props = defineProps<{
  workbench: ProjectWorkbenchResult,
  onlyMine: boolean,
  baseLink: string,
  labels: Record<string, string>,
}>();

const open = reactive<Record<number, boolean>>(Object.fromEntries(props.workbench.projects.map((project) => [project.id, project.expanded])));

function scopeLink(mine: boolean): string {
  return mine ? `${props.baseLink}?scope=mine` : props.baseLink;
}

function activityLink(): string {
  return `${props.baseLink}?view=activity`;
}

function eventLabel(kind: string): string {
  return props.labels[kind === 'pull_merged' ? 'pullMerged' : kind === 'issue_closed' ? 'issueClosed' : kind] || kind;
}

function firstLine(value: string): string {
  return value.split('\n', 1)[0];
}

function shortDate(value: string): string {
  if (!value) return '—';
  return value.slice(5).replace('-', '/');
}

function relativeTime(value: string): string {
  const elapsed = Date.now() - new Date(value).valueOf();
  if (elapsed < 60 * 60 * 1000) return `${Math.max(1, Math.floor(elapsed / 60000))}m`;
  if (elapsed < 24 * 60 * 60 * 1000) return `${Math.floor(elapsed / 3600000)}h`;
  return `${Math.floor(elapsed / 86400000)}d`;
}

function visibleEvents(project: ProjectWorkbenchProject) {
  return project.activity.progress.slice(0, open[project.id] ? 5 : 1);
}

function initials(name: string): string {
  return name.trim().slice(0, 1).toUpperCase();
}
</script>

<template>
  <section class="project-workbench" aria-labelledby="project-workbench-title">
    <header class="project-workbench-header">
      <div>
        <h1 id="project-workbench-title">{{ labels.title }}</h1>
        <p>{{ labels.subtitle }}</p>
      </div>
      <nav class="project-workbench-scope" aria-label="Project scope">
        <a :class="{active: !onlyMine}" :href="scopeLink(false)">{{ labels.team }}</a>
        <a :class="{active: onlyMine}" :href="scopeLink(true)">{{ labels.mine }}</a>
      </nav>
    </header>

    <div class="project-attention" aria-label="Project attention summary">
      <strong>{{ labels.attention }}</strong>
      <span v-if="workbench.attention.blocked" class="danger"><i/>{{ labels.blocked }} {{ workbench.attention.blocked }}</span>
      <span v-if="workbench.attention.overdue" class="warning"><i/>{{ labels.overdue }} {{ workbench.attention.overdue }}</span>
      <span v-if="workbench.attention.due_soon"><i/>{{ labels.dueSoon }} {{ workbench.attention.due_soon }}</span>
      <span v-if="workbench.attention.stale"><i/>{{ labels.stale }} {{ workbench.attention.stale }}</span>
      <span v-if="workbench.attention.unowned"><i/>{{ labels.unowned }} {{ workbench.attention.unowned }}</span>
      <span v-if="!Object.values(workbench.attention).some(Boolean)" class="healthy"><i/>{{ labels.clear }}</span>
    </div>

    <div v-if="workbench.projects.length" class="project-workbench-layout">
      <main class="project-execution-list">
        <article v-for="project in workbench.projects" :key="`${project.organization}-${project.id}`" class="project-execution" :class="[`risk-${project.risk_key}`, {expanded: open[project.id]}]">
          <header class="project-execution-header">
            <div class="project-execution-identity">
              <div class="project-kicker">
                <a :href="project.organization_url">{{ project.organization }}</a>
                <span>{{ project.stage }}</span>
                <span v-if="project.risk_key !== 'normal'" class="project-risk">{{ project.risk }}</span>
              </div>
              <a class="project-name" :href="project.link">{{ project.name }}</a>
              <p v-if="project.description">{{ project.description }}</p>
            </div>
            <div class="project-responsibility">
              <div class="project-responsibility-row">
                <span class="project-section-label">{{ labels.owner }}</span>
                <a v-if="project.owner" :href="project.owner.link" class="project-person">
                  <b>{{ initials(project.owner.full_name) }}</b>{{ project.owner.full_name }}
                </a>
                <span v-else class="project-unowned">{{ labels.unowned }}</span>
              </div>
              <div class="project-responsibility-row">
                <span class="project-section-label">{{ labels.participants }}</span>
                <span v-if="project.participants.length" class="project-participants">
                  <a v-for="person in project.participants.slice(0, 4)" :key="person.id" :href="person.link" :title="person.full_name">{{ initials(person.full_name) }}</a>
                </span>
                <span v-else>—</span>
              </div>
            </div>
            <div class="project-deadline" :class="{overdue: project.overdue}">
              <span>{{ labels.target }}</span>
              <strong>{{ shortDate(project.target_date) }}</strong>
            </div>
          </header>

          <div class="project-progress-row">
            <div class="project-progress-track"><span :style="{width: `${Math.min(100, Math.max(0, project.progress))}%`}"/></div>
            <strong>{{ Math.round(project.progress) }}%</strong>
            <span class="project-progress-evidence">{{ project.activity.release_count }} {{ labels.releases }} · {{ project.activity.merged_pulls }} {{ labels.merged }} · {{ project.activity.open_pulls }} {{ labels.open }}</span>
          </div>

          <section class="project-evidence">
            <header><span class="project-section-label">{{ labels.realProgress }}</span></header>
            <p v-if="project.activity_error" class="project-evidence-empty">{{ labels.activityUnavailable }}</p>
            <p v-else-if="!project.activity.progress.length" class="project-evidence-empty">{{ labels.noEvidence }}</p>
            <ol v-else>
              <li v-for="event in visibleEvents(project)" :key="`${event.kind}-${event.repository_id}-${event.link}`">
                <span class="event-kind" :class="`event-${event.kind}`">{{ eventLabel(event.kind) }}</span>
                <a :href="event.link">{{ firstLine(event.title) }}</a>
                <a class="event-repository" :href="event.repository_link">{{ event.repository_full_name }}</a>
                <span class="event-meta">{{ event.author_name }} · {{ relativeTime(event.occurred_at) }}</span>
              </li>
            </ol>
            <button v-if="project.activity.progress.length > 1" type="button" @click="open[project.id] = !open[project.id]">
              {{ open[project.id] ? labels.collapse : labels.expand }}
            </button>
          </section>

          <div class="project-decision-row">
            <div>
              <span class="project-section-label">{{ labels.currentProblem }}</span>
              <p :class="{empty: !project.current_problem}">{{ project.current_problem || '—' }}</p>
            </div>
            <div>
              <span class="project-section-label">{{ labels.nextAction }}</span>
              <p :class="{empty: !project.next_action}">{{ project.next_action || '—' }}</p>
              <small v-if="project.next_action_owner || project.next_action_due">
                {{ project.next_action_owner?.full_name }}<template v-if="project.next_action_due"> · {{ shortDate(project.next_action_due) }}</template>
              </small>
            </div>
          </div>
        </article>
      </main>

      <aside class="project-workbench-aside">
        <section>
          <header><h2>{{ labels.people }}</h2><span>{{ workbench.people.length }}</span></header>
          <ul class="project-people-list">
            <li v-for="person in workbench.people" :key="person.id">
              <a :href="person.link"><b>{{ initials(person.full_name) }}</b><span><strong>{{ person.full_name }}</strong><small>{{ person.projects.slice(0, 2).join('、') }}</small></span></a>
              <span>{{ person.owned }} / {{ person.participating }}</span>
            </li>
          </ul>
        </section>
        <section>
          <header><h2>{{ labels.quickLinks }}</h2></header>
          <nav class="project-quick-links">
            <a :href="activityLink()">{{ labels.allActivity }} <span>→</span></a>
            <a v-for="project in workbench.projects.slice(0, 3)" :key="project.link" :href="project.link">{{ project.name }} <span>→</span></a>
          </nav>
        </section>
      </aside>
    </div>

    <div v-else class="project-workbench-empty">
      <span>⌁</span>
      <h2>{{ labels.empty }}</h2>
      <p>{{ labels.configure }}</p>
    </div>
  </section>
</template>
