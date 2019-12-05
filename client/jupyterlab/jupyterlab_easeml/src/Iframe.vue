<template>
  <iframe :src=url></iframe>
</template>

<script>
module.exports = {
  data: function () {
    return {
      url: "http://localhost:8081/"
    }
  }
}
</script>

<style scoped>
p {
  font-size: 2em;
  text-align: center;
}
iframe {
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
  flex-grow: 1;
  border: none;
  margin: 0;
  padding: 0;
}
</style>
