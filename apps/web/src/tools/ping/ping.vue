<script setup lang="ts">
import PingTool from '@/components/PingTool.vue'
</script>

<template>
  <PingTool kind="icmp" path="/v1/ping/stream" />
</template>
