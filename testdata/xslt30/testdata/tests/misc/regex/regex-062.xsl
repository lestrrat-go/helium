<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase with tokenize($input,$pattern) where $pattern is a multi-character separator using character class regular expression notation. -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
  
      <out>
         <t:value-of select="tokenize('1, 15, 24, 50', ',\s*')" separator="; "/>
      </out>
   </t:template>
</t:transform>
