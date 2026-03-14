<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <xsl:param name="twiddle" select="'~'"/>

   <!-- Purpose: Test serialization with item separator -->

   <xsl:template name="xsl:initial-template">
      <xsl:result-document method="text" build-tree="no" indent="no" item-separator="{$twiddle}">
         <xsl:sequence select="11 to 20"/>
      </xsl:result-document>
   </xsl:template>
</xsl:transform>
