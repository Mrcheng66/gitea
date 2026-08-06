<script setup lang="ts">
import {ref} from 'vue';
import type {OrgProjectTeam} from './types.ts';

const props = defineProps<{teams: OrgProjectTeam[], emptyText: string}>();
const selected = ref(new Set(props.teams.filter((team) => team.selected).map((team) => team.id)));

function toggle(teamID: number, checked: boolean) {
  if (checked) selected.value.add(teamID);
  else selected.value.delete(teamID);
}
</script>

<template>
  <div class="org-project-team-list">
    <label v-for="team in teams" :key="team.id" class="org-project-team-option" :class="{disabled: team.deleted}">
      <input
        type="checkbox"
        name="team_ids"
        :value="team.id"
        :checked="selected.has(team.id)"
        :disabled="team.deleted"
        @change="toggle(team.id, ($event.target as HTMLInputElement).checked)"
      >
      <span><strong>{{ team.name }}</strong><small v-if="team.description" class="text light-2">{{ team.description }}</small></span>
    </label>
    <p v-if="!teams.length" class="text light-2">{{ emptyText }}</p>
  </div>
</template>
