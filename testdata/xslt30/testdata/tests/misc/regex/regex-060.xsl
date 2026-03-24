<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Basic test of tokenize($input,$pattern) where $pattern is a non-whitespace separator.  -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:value-of select="tokenize('8,1,4',',')" separator="; "/>
      </out>
   </t:template>
</t:transform>
