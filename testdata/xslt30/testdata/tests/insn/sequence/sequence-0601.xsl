<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:f="http://boag.org"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="f">
<!-- Purpose: Simple test of calling xsl:function from within the for loop, which stresses the variable stack. -->

   <t:output indent="no" />

   <t:template match="@*|node()">
      <t:copy>
         <t:apply-templates select="@*|node()"/>
      </t:copy>
   </t:template>

   <t:function name="f:make-row">
      <t:param name="row-seq"/>
      <TR xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:for-each select="$row-seq">
            <t:copy-of select="."/>
         </t:for-each>
      </TR>
   </t:function>

   <t:template match="TABLE">
      <t:copy>
         <t:apply-templates select="@*"/>
         <t:copy-of select="for $columnNum in (1 to count(TR[1]/TD))                           return f:make-row(TR/TD[$columnNum])"/>
      </t:copy>
   </t:template>
</t:transform>
