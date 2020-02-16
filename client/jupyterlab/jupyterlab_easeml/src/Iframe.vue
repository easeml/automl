<template>
  <div v-if="IsAlive">
  <iframe :src=url></iframe>
  </div>
  <div v-else class="ErrorMessage">
  Make sure to set Easeml server address correctly in:
  <ul>
    <li> Settings > Advanced Settings Editor > Easeml > easemlServer </li>
  </ul>
    easemlServer variable currently set to: {{ url }}
    <br>
    Is server alive: {{IsAlive}}
    <br>
    Checking for API at: {{url}}/api/v1/
    <br>
    Response: {{response}}
    <br>
    Error: {{error}}
  </div>
</template>

<script>

import axios from 'axios';


export default {
  data() {
    return {
      response: "",
      error: ""
    }
  },
  props: {
    'url': {
      type: String,
      required: true,
    }
  },
  mounted(){
    axios.get(this.url+"/api/v1/")
            .then(response => (this.response = response.data))
            .catch(error => {
              console.log(error)
              this.error = error
            })
  },
  computed: {
    IsAlive: function() {
      console.log(this.response.includes("Easeml"));
      return this.response.includes("Easeml")
    }
  }
}
</script>

<style scoped>
  p {
    font-size: 2em;
    text-align: center;
  }
  div{
    width: 100%;
    height: 100%;
  }
  iframe {
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 0;
    flex-grow: 1;
    border: none;
    padding: 0;
  }
  div.ErrorMessage{
    padding:30px 20px 10px 25px;
  }
</style>
