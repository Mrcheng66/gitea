<script setup lang="ts">
import type {OrgProjectField, OrgProjectMember} from './types.ts';

const props = defineProps<{
  field: OrgProjectField,
  members: OrgProjectMember[],
}>();

const value = defineModel<any>({required: true});

function inputID() {
  return `org-project-field-${props.field.key}`;
}
</script>

<template>
  <div class="field" :class="{required: field.required, 'org-project-form-wide': field.type === 'long_text' || field.type === 'multi_select' || field.type === 'member_array'}">
    <label :for="inputID()">{{ field.label }}</label>
    <textarea v-if="field.type === 'long_text'" :id="inputID()" v-model="value" rows="4" :required="field.required"/>
    <select v-else-if="field.type === 'single_select'" :id="inputID()" v-model="value" class="ui fluid selection dropdown org-project-dropdown" :required="field.required">
      <option v-if="!field.required" value="">—</option>
      <option v-for="option in field.options" :key="option.key" :value="option.key">{{ option.label }}</option>
    </select>
    <select v-else-if="field.type === 'multi_select'" :id="inputID()" v-model="value" class="ui fluid search multiple selection dropdown org-project-dropdown" multiple>
      <option v-for="option in field.options" :key="option.key" :value="option.key">{{ option.label }}</option>
    </select>
    <select v-else-if="field.type === 'member'" :id="inputID()" v-model.number="value" class="ui fluid search selection dropdown org-project-dropdown" :required="field.required">
      <option v-if="!field.required" :value="null">—</option>
      <option v-for="member in members" :key="member.id" :value="member.id">{{ member.full_name || member.name }}</option>
    </select>
    <select v-else-if="field.type === 'member_array'" :id="inputID()" v-model="value" class="ui fluid search multiple selection dropdown org-project-dropdown" multiple>
      <option v-for="member in members" :key="member.id" :value="member.id">{{ member.full_name || member.name }}</option>
    </select>
    <div v-else-if="field.type === 'boolean'" class="ui checkbox">
      <input :id="inputID()" v-model="value" type="checkbox">
      <label :for="inputID()">{{ field.label }}</label>
    </div>
    <input v-else-if="field.type === 'integer'" :id="inputID()" v-model.number="value" type="number" step="1" :required="field.required">
    <input v-else-if="field.type === 'decimal'" :id="inputID()" v-model.number="value" type="number" step="any" :required="field.required">
    <input v-else-if="field.type === 'percent'" :id="inputID()" v-model.number="value" type="number" step="0.01" min="0" max="100" :required="field.required">
    <input v-else-if="field.type === 'date'" :id="inputID()" v-model="value" class="org-project-date-input" type="date" :required="field.required">
    <input v-else-if="field.type === 'date_time'" :id="inputID()" v-model="value" class="org-project-date-input" type="datetime-local" :required="field.required">
    <input v-else :id="inputID()" v-model.trim="value" type="text" :required="field.required">
  </div>
</template>
