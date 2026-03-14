<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: General compare of boolean constant to zero-length string. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <eqTrue>
            <t:value-of select="''=true()"/>
         </eqTrue>
         <eqFalse>
            <t:value-of select="false()=''"/>
         </eqFalse>
         <neTrue>
            <t:value-of select="'' != true()"/>
         </neTrue>
         <neFalse>
            <t:value-of select="false() != ''"/>
         </neFalse>
      </out>
   </t:template>
</t:transform>
