<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test of fn:static-base-uri accessor invoked under several diffreent current context items -->

   <t:template match="/doc">
      <out>
         <t1>
            <t:value-of select="ends-with(static-base-uri(), 'accessor-030.xsl')"/>
         </t1>
         <t2>
            <t:for-each select="inner">
               <t:value-of select="ends-with(static-base-uri(), 'accessor-030.xsl')"/>
            </t:for-each>
         </t2>
      </out>
   </t:template>
</t:transform>
