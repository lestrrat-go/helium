<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case using a parameter to select nodes by position to test the pitfalls with numeric predicates.  -->

   <t:template match="doc">
      <t:param name="n">2</t:param>
      <out>
         <t:value-of select="item[$n]"/>
      </out>
   </t:template>
</t:transform>
