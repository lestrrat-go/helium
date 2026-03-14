<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test substring-before() first or second argument choosing first of multiple nodes. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <first>
            <t:value-of select="substring-before(doc/a,'o')"/>
         </first>
         <second>
            <t:value-of select="substring-before('Aprilfool',doc/c)"/>
         </second>
         <third>
            <t:value-of select="substring-before(doc/a,doc/d)"/>
         </third>
      </out>
   </t:template>
</t:transform>
