<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:my="http://www.example.com/hal"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="my"
             version="2.0">
<!-- Purpose: Show that @xpath-default-namespace does not affect variable names.-->

   <t:variable name="var" select="'correct'"/>

   <t:variable name="my:var" select="'wrong'"/>

   <t:template match="/">
      <out>
         <t:value-of select="$var" xpath-default-namespace="http://www.example.com/hal"/>
      </out>
   </t:template>
</t:transform>
