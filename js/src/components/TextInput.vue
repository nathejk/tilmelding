<template>
    <div class="control-group form-group" :class="{row:layout=='row',material:layout=='material'}">
        <label v-if="layout!='material'" class="col-form-label col-form-label-sm col-sm" :class="{'col-sm-3':layout=='row'}">{{ label }}</label>
        <div :class="{col:layout=='row'}">
            <input class="form-control form-control-sm" :class="[className, {nonempty:this.value}]" :value="value" @input="$emit('input', $event.target.value)" :placeholder="placeholder" :type="type">
            <label v-if="layout=='material'">{{ label }}</label>
            <span class="bar"></span>
            <!--small v-if="!!$slots.default" class="form-text text-muted"><slot /></small-->
        </div>
    </div>
</template>

<style lang="scss">
.material {
    position: relative;

    input.form-control {
        &:focus {
            outline: none ! important;
        }
        &:focus ~ .bar:before, &:focus ~ .bar:after {
            width: 50%;
        }
        &:focus ~ label, &.nonempty ~ label {
            top: 0;
            font-size: 0.65rem;
            color: rgba(0, 0, 0, 0.3);;
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
        color: rgba(0, 0, 0, 0.3);
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
        className: String,
        format: String,
    },
}
</script>
