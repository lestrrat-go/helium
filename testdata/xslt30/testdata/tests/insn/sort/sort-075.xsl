<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test that stable="false" can be set, but apply to data where it's moot .-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:apply-templates select="doc/alpha">
            <t:sort select="*[1]" data-type="text" stable="false"/>
         </t:apply-templates>
      </out>  
   </t:template>

   <t:template match="doc/alpha">
      <t:value-of select="display"/>
      <t:text>,</t:text>
   </t:template>
</t:transform>
