<script setup lang="ts">
import {SvgIcon} from '../../svg.ts';
import type {OrgProjectActivitySummary} from './types.ts';

type Locale = {
  openPulls: string,
  mergedPulls: string,
  releases: string,
  repositories: string,
  repository: string,
  recentCommits: string,
  latestRelease: string,
  empty: string,
  since: string,
};

defineProps<{summary: OrgProjectActivitySummary, locale: Locale}>();
</script>

<template>
  <div class="org-project-activity">
    <p class="text light-2">
      {{ locale.since }} <relative-time :datetime="summary.since" prefix=""/>
    </p>
    <div class="org-project-activity-summary">
      <section class="ui segment">
        <SvgIcon name="octicon-git-pull-request" :size="20"/>
        <span>{{ locale.openPulls }}</span>
        <strong>{{ summary.open_pulls }}</strong>
      </section>
      <section class="ui segment">
        <SvgIcon name="octicon-git-merge" :size="20"/>
        <span>{{ locale.mergedPulls }}</span>
        <strong>{{ summary.merged_pulls }}</strong>
      </section>
      <section class="ui segment">
        <SvgIcon name="octicon-tag" :size="20"/>
        <span>{{ locale.releases }}</span>
        <strong>{{ summary.release_count }}</strong>
        <small v-if="summary.latest_release_at" class="text light-2 org-project-activity-latest">
          {{ locale.latestRelease }} <relative-time :datetime="summary.latest_release_at" prefix=""/>
        </small>
      </section>
    </div>

    <section v-if="summary.repositories.length" class="ui segment">
      <h3 class="ui header">{{ locale.repositories }}</h3>
      <div class="org-project-table-wrap">
        <table class="ui table org-project-activity-table">
          <thead><tr><th>{{ locale.repository }}</th><th>{{ locale.openPulls }}</th><th>{{ locale.mergedPulls }}</th><th>{{ locale.releases }}</th><th>{{ locale.latestRelease }}</th></tr></thead>
          <tbody>
            <tr v-for="repository in summary.repositories" :key="repository.id">
              <td><a :href="repository.link">{{ repository.full_name }}</a></td>
              <td>{{ repository.open_pulls }}</td>
              <td>{{ repository.merged_pulls }}</td>
              <td>{{ repository.release_count }}</td>
              <td><relative-time v-if="repository.latest_release_at" :datetime="repository.latest_release_at" prefix=""/><span v-else>—</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="ui segment">
      <h3 class="ui header">{{ locale.recentCommits }}</h3>
      <div v-if="summary.commits.length" class="ui relaxed divided list">
        <article v-for="commit in summary.commits" :key="`${commit.repository_id}-${commit.sha}`" class="item org-project-activity-commit">
          <SvgIcon name="octicon-git-commit" :size="18"/>
          <div class="content">
            <a class="header" :href="commit.link">{{ commit.message }}</a>
            <div class="description">
              <a :href="commit.repository_link">{{ commit.repository_full_name }}</a>
              <code>{{ commit.short_sha }}</code>
              <span>{{ commit.author_name }}</span>
              <relative-time :datetime="commit.committed_at" prefix=""/>
            </div>
          </div>
        </article>
      </div>
      <div v-else class="ui placeholder segment">{{ locale.empty }}</div>
    </section>
  </div>
</template>
