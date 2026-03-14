<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <!-- Static on non-global param -->
   
   <xsl:template name="xsl:initial-template" expand-text="yes">
      <xsl:param name="p" static="yes" select="2"/>
      <out/>
   </xsl:template>
   
</xsl:transform>
