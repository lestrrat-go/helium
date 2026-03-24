<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test to set the value of a variable inside xsl:apply-imports instruction. -->

   <t:import href="import2.xsl"/>

   <t:template match="doc">
      <out>
         <t:apply-imports>
            <t:with-param name="vpar1" select="'Assigned Value'"/>
            <t:with-param name="Ppar1" select="'Assigned Value'"/>
         </t:apply-imports>
      </out>
   </t:template>
</t:transform>
