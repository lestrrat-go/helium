<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of AVT where the value is a sequence of atomic values surrounded only in curly brackets. 
  				The result is a whitespace separated list of the values. (from spec) -->

   <t:template match="doc">
      <out test="{10.32, 5.50, 8.31}"/>
   </t:template>
</t:transform>
