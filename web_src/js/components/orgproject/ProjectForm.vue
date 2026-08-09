<script setup lang="ts">
import {computed, ref} from 'vue';
import ProjectFormField from './ProjectFormField.vue';
import {groupProjectFields, initialProjectValues, serializeProjectValues, sortedActiveFields} from './types.ts';
import type {OrgProjectField, OrgProjectMember, OrgProjectSchema, ProjectFormGroupKey} from './types.ts';

type ProjectFormGroupLabels = Record<ProjectFormGroupKey, {title: string, description: string}>;

const props = withDefaults(defineProps<{
  schema: OrgProjectSchema,
  initialValues: Record<string, unknown>,
  members: OrgProjectMember[],
  layout?: 'plain' | 'grouped',
  labels?: ProjectFormGroupLabels,
}>(), {
  layout: 'plain',
  labels: () => ({
    plan: {title: '', description: ''},
    status: {title: '', description: ''},
    other: {title: '', description: ''},
  }),
});

const values = ref<Record<string, any>>(initialProjectValues(props.schema, props.initialValues));
const fields = computed(() => sortedActiveFields(props.schema));
const groupedFields = computed(() => groupProjectFields(fields.value));
const visibleGroups = computed(() => (Object.entries(groupedFields.value) as Array<[ProjectFormGroupKey, OrgProjectField[]]>).filter(([, groupFields]) => groupFields.length));
const serializedValues = computed(() => serializeProjectValues(props.schema, values.value));
</script>

<template>
  <template v-if="layout === 'grouped'">
    <section v-for="([groupKey, groupFields]) in visibleGroups" :key="groupKey" class="ui segment org-project-create-panel" :data-project-field-group="groupKey">
      <div class="org-project-create-panel-heading">
        <div>
          <h2>{{ labels[groupKey].title }}</h2>
          <p v-if="labels[groupKey].description">{{ labels[groupKey].description }}</p>
        </div>
      </div>
      <div class="org-project-form-fields">
        <ProjectFormField v-for="field in groupFields" :key="field.key" v-model="values[field.key]" :field="field" :members="members"/>
      </div>
    </section>
  </template>
  <div v-else class="org-project-form-fields">
    <ProjectFormField v-for="field in fields" :key="field.key" v-model="values[field.key]" :field="field" :members="members"/>
  </div>
  <input type="hidden" name="values" :value="serializedValues">
</template>
