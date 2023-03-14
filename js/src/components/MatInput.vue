<template>
    <div class="md-input control-group form-group">
        <input class="md-form-control" required="" type="text">
        <label>{{ label }}</label>
        <span class="bar"></span>
    </div>
</template>

<style lang="scss">
.md-input {
    position: relative;

    input {
        &:focus {
            outline: none;
        }
        &:focus ~ .bar:before, &:focus ~ .bar:after {
            width: 50%;
        }
        &:focus ~ label, &:valid ~ label {
            top: 0;
            font-size: 0.65rem;
            color: rgba(0, 0, 0, 0.5);;
        }
        &:focus ~ label {
            color: #03A9F4;
        }

        font-size: 16px;
        padding: 12px 10px 0px 5px;
        display: block;
        border: none;
        border-bottom: 2px solid #CACACA;
        box-shadow: none;
        width: 100%;
    }

    label {
        color: rgba(0, 0, 0, 0.5);
        font-size: 16px;
        font-weight: normal;
        position: absolute;
        pointer-events: none;
        left: 5px;
        top: 5px;
        transition: 0.2s ease all;
        -moz-transition: 0.2s ease all;
        -webkit-transition: 0.2s ease all;
    }

    .bar {
        &:before, &:after {
            content: '';
            height: 2px;
            width: 0;
            bottom: 0px;
            position: absolute;
            background: #03A9F4;
            transition: 0.3s ease all;
            -moz-transition: 0.3s ease all;
            -webkit-transition: 0.3s ease all;
        }
        &:before {
            left: 50%;
        }
        &:after {
            right: 50%;
        }
    }
}
</style>

<script>
export default {
    props: {
        label: String,
        value: String,
        placeholder: String,
        type: {default:'text', type:String},
        layout: {default:'row', type:String},
    },
}
</script>
