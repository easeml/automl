<template>
    <div>
        <form ref="dropzone" class="dropzone" @click="openFileDialog">
            <div class="droparea text-muted">
                <h1><i class="mdi mdi-cloud-upload"></i></h1>
                <h1> {{msg}}</h1>
                <input type="file" ref="fileinput" style="opacity: 0" @change="fileInputFieldChange($event)">
            </div>
        </form>
        <div class="text-danger" v-if="error">
            <p><i class="mdi mdi-close-circle"></i>&nbsp;{{error}}</p>
        </div>
    </div>
</template>

<script>

/*
  Determines if the drag and drop functionality is in the
  window
*/
function determineDragAndDropCapable() {
  /*
    Create a test element to see if certain events
    are present that let us do drag and drop.
  */
  var div = document.createElement('div');

  /*
    Check to see if the `draggable` event is in the element
    or the `ondragstart` and `ondrop` events are in the element. If
    they are, then we have what we need for dragging and dropping files.

    We also check to see if the window has `FormData` and `FileReader` objects
    present so we can do our AJAX uploading
  */
  return ( ( 'draggable' in div )
          || ( 'ondragstart' in div && 'ondrop' in div ) )
          && 'FormData' in window
          && 'FileReader' in window;
}

export default {
    name: "FileDropzone",
    props: {
        msg: {
            type: String,
            default: "Drop dataset *.tar file or click to upload."
        },
        allowedTypes: {
            type: Array,
            default() { return ["application/x-tar", "application/gzip"] }
        }
    },
    data() {
        return {
            file: null,
            error: "",
            fileInputValue: null
        }
    },
    methods: {
        openFileDialog() {
            this.$refs.fileinput.click();
        },
        fileInputFieldChange(e) {
            if (e.target.files) {
                this.acceptFileInput(e.target.files);
            }
        },
        acceptFileInput(files) {

            if (files.length > 1) {
                this.error = "Only one file can be uploaded."
                return
            }

            let file = files[0];
            
            if (!this.allowedTypes.includes(file.type)) {
                this.error = "Uploaded file must be one of the following types: " + this.allowedTypes.join(", ");
                return
            }

            this.error = "";
            this.file = file;

            // The file has been accepted. We can trigger the event to the host container.
            this.$emit("new-file", file);
        }
    },
    mounted() {

        if (determineDragAndDropCapable()) {

            ['drag', 'dragstart', 'dragend', 'dragover', 'dragenter', 'dragleave', 'drop'].forEach( function( evt ) {
            /*
                For each event add an event listener that prevents the default action
                (opening the file in the browser) and stop the propagation of the event (so
                no other elements open the file in the browser)
            */
            this.$refs.dropzone.addEventListener(evt, function(e){
                e.preventDefault();
                e.stopPropagation();
            }.bind(this), false);
            }.bind(this));

            /*
            Add an event listener for drop to the form
            */
            this.$refs.dropzone.addEventListener('drop', function(e){
                /*
                    Capture the files from the drop event and add them to our local files
                    array.
                */
                this.acceptFileInput(e.dataTransfer.files);

            }.bind(this));

        }
    }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
.droparea {
    position: absolute;
    top: 50%;
    left: 50%;
    width: 100%;
    transform: translate(-50%, -50%);
    text-align: center;
}
.dropzone {
    cursor: pointer;
}
h3 {
  margin: 40px 0 0;
}
ul {
  list-style-type: none;
  padding: 0;
}
li {
  display: inline-block;
  margin: 0 10px;
}
a {
  color: #42b983;
}
</style>
