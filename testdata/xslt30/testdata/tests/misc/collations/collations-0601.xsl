<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:template match="names">
      <out>
         <upper-first>
            <t:perform-sort select="tokenize(., ',\s*')">
                <t:sort select="." case-order="upper-first"/>
            </t:perform-sort>
         </upper-first>
         <lower-first>
            <t:perform-sort select="tokenize(., ',\s*')">
                <t:sort select="." case-order="lower-first"/>
            </t:perform-sort>
         </lower-first>
      </out>
   </t:template>
</t:transform>
