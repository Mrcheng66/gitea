<script setup lang="ts">
import type {OrgProjectMetricDisplay} from './types.ts';

defineProps<{metrics: OrgProjectMetricDisplay[]}>();

function displayValue(value: number) {
  return Number.isInteger(value) ? String(value) : value.toLocaleString(undefined, {maximumFractionDigits: 2});
}
</script>

<template>
  <div v-if="metrics.length" class="org-project-metric-grid tw-mb-4">
    <section v-for="metric in metrics" :key="metric.Key" class="ui segment org-project-metric-card">
      <h3 class="ui small header">{{ metric.Label }}</h3>
      <div class="org-project-metric-values">
        <div v-for="(bucket, index) in metric.Buckets" :key="`${metric.Key}-${index}`" class="org-project-metric-value">
          <span v-if="bucket.Bucket !== undefined" class="text light-2">{{ bucket.Bucket }}</span>
          <strong>{{ displayValue(bucket.Value) }}</strong>
        </div>
      </div>
    </section>
  </div>
</template>
