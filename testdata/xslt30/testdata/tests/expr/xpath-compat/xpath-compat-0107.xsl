<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test inequality relation between number and string that is a lexically-valid number. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <gtNum>
            <t:value-of select="'35' &gt; 17"/>
         </gtNum>
         <ltNum>
            <t:value-of select="'6' &lt; 44"/>
         </ltNum>
         <ltString>
            <t:value-of select="6 &lt; '44'"/>
         </ltString>
         <geNum>
            <t:value-of select="'99' &gt;= 99"/>
         </geNum>
         <leNum>
            <t:value-of select="'44' &lt;= 100"/>
         </leNum>
         <leString>
            <t:value-of select="6 &lt;= '17'"/>
         </leString>
      </out>
   </t:template>
</t:transform>
