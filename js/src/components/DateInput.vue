<template>
<!-- 
<v-date-picker
  v-model='date'>
  <div
    class='field'
    slot-scope="inputValue" hb='{ inputValue, updateValue }'>
    <label class='label'>
      Enter Date
    </label>
    <div class="control has-icons-left has-icons-right">
      <input
        type='text'
         :class='["input", inputState.type]'
        :placeholder='inputState.message'
        :value="date"
        @change='updateValue($event.target.value)' />
      <span class="icon is-small is-left">
        <i class="fas fa-calendar"></i>
      </span>
      <span class='icon is-small is-right'>
        <i class="fas fa-check"></i>
      </span>
    </div>
    <p class="{ff:log(inputValue)}">This username is available</p>
  </div>
</v-date-picker>
-->
<v-date-picker v-model="date" mode="dateTime" is24hr>
  <template v-slot="{ inputValue, inputEvents }">
    <div class="control-group form-group" lass="{row:layout=='row',material:layout=='material'}">
        <div>
            <input class="form-control form-control-sm" :class="[className, {nonempty:this.value!=''}]" :value="inputValue" v-on="inputEvents" @input="$emit('input', $event.target.value)" :placeholder="placeholder" :type="type">
            <label>{{ label }}</label>
            <span class="bar"></span>
        </div>
    </div>
    <!--input
      class="px-2 py-1 border rounded focus:outline-none focus:border-blue-300"
      :value="inputValue"
      v-on="inputEvents"
            <input class="form-control form-control-sm" :class="[className, {nonempty:this.value!=''}]" :value="inputValue" v-on="inputEvents" @input="$emit('input', $event.target.value)" :placeholder="placeholder" :type="type">
    /-->
  </template>
</v-date-picker>
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
        placeholder: {default:'YYYY-MM-DD', type:String},
        type: {default:'text', type:String},
        layout: {default:'material', type:String},
        className: String,
    },
  data() {
    return {
      date: new Date(),
    };
  },
  computed: {
      log(o) {
          console.log(o);
          return true
      },
    inputState() {
      if (!this.date) {
        return {
          type: "is-danger",
          message: "Date required."
        };
      }
      return {
        type: "is-primary",
        message: ""
      };
    }
  }
};
</script>

