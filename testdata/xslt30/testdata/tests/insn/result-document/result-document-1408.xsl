<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <xsl:param name="twiddle" select="'~'"/>

   <!-- Purpose: Test serialization with item separator, outputting nodes -->

   <xsl:template name="xsl:initial-template">
      <xsl:result-document method="xml" indent="no" item-separator="{$twiddle}">
         <xsl:comment>start</xsl:comment>
         <xsl:sequence select="11 to 15"/>
         <xsl:comment>middle</xsl:comment>
         <xsl:sequence select="16 to 20"/>
         <xsl:comment>end</xsl:comment>
      </xsl:result-document>
   </xsl:template>
</xsl:transform>
