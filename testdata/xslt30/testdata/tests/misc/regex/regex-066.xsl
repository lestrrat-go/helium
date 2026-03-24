<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with matches($input?,$pattern, $flag) using 'dot-all' mode ($flag is set to 's'). 
  				Test contrasts the result of metacharacter '.' when the flag is on and off. -->

   
   <t:variable name="var" select="/doc/str1"/>

   <t:template match="/">
      <out>
         <t:value-of select="matches(//str1,'hello.*world')"/>
         <t:text>, </t:text>
         <t:value-of select="matches(//str1,'hello.*world','s')"/>
      </out>
   </t:template>
</t:transform>
