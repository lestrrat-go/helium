<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Give a string that is not a lexically-valid number as one operand of an arithmetic operator, and use result in a predicate. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:for-each select="doc/item[@width+1 != 2]">
            <ok>
               <t:value-of select="@seq"/>
            </ok>
         </t:for-each>
      </out>
   </t:template>
</t:transform>
