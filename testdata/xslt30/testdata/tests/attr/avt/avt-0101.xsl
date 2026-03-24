<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of single AVT where the value is a sequence with mixed atomic content.
  				The result is a whitespace separated list of the values. -->

   <t:template match="doc">
      <out test="{('1', 1.2, item)}"/>
   </t:template>
</t:transform>
