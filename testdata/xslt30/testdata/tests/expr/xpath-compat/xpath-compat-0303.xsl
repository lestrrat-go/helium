<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test string-join() second argument choosing first of multiple nodes and being cast to string. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <first>
            <t:value-of select="string-join(doc/a,'-')"/>
         </first>
         <second>
            <t:value-of select="string-join(doc/a,doc/b)"/>
         </second>
         <third>
            <t:value-of select="string-join(doc/a,4+4)"/>
         </third>
         <third>
            <t:value-of select="string-join(doc/b,true())"/>
         </third>
      </out>
   </t:template>
</t:transform>
