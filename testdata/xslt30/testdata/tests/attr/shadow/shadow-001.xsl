<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0" id="shadow-001">

   <!-- simple use of shadow attribute (static AVT) -->
   
   <xsl:variable name="N" static="yes" select="'x'"/>
   
   <xsl:template name="main">
      <xsl:variable name="x" select="3"/>
      <out><xsl:value-of _select="${}{$N}"/></out>
   </xsl:template>
   
</xsl:transform>
