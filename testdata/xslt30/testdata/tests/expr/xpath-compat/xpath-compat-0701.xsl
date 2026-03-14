<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Give a path to a comment node that is a lexically-valid number as one operand of an arithmetic operator. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:value-of select="17 + doc/comment()"/>
      </out>
   </t:template>
</t:transform>
