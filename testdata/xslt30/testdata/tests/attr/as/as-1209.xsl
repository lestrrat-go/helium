<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test type of global xsl:variable selecting a processing instruction 
  				from input file or from a temporary tree, @as="processing-instruction()". -->

   <t:variable name="temporary-tree">
      <a>
         <t:processing-instruction name="name" select="processing-instruction"/>
         <b att="one">hello</b>
      </a>
   </t:variable>

   <t:variable name="var1" select="/doc/processing-instruction()" as="processing-instruction()"/>

   <t:variable name="var2" select="$temporary-tree/a/processing-instruction()"
      as="processing-instruction()"/>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of processing-instruction()"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of processing-instruction()"/>
         </e2>
      </out>
   </t:template>
</t:transform>
