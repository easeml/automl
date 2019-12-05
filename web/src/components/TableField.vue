<template>
    <td><span data-toggle="tooltip" :title=itemTitle>
        {{itemValue}}
    </span></td>
</template>

<script>

export default {
    name: "TableField",
    props: {
        itemTitle: {
            type: [Number, String],
            default: ""
        },
        itemValue: {
            type: [Number, String],
            default: ""
        }
    }
};
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped>
    .mainField{
        font-weight: bold;
    }
</style>
