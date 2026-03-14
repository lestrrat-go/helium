<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" id="shadow-002">

   <!-- shadow attribute (static AVT) on xsl:variable/@static -->
   
   <xsl:variable name="N" _static="{if (system-property('xsl:version') = '3.0') then 'yes' else 'no'}" select="'x'"/>
   
   <xsl:template name="main">
      <xsl:variable name="x" select="3"/>
      <out><xsl:value-of _select="${$N}"/></out>
   </xsl:template>
   
</xsl:transform>
