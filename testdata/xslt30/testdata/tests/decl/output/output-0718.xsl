<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns="http://www.w3.org/1999/xhtml" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0">

   <xsl:output method="text" build-tree="yes" indent="no" item-separator="&#x20;&#x20;&#x20;"/>

   <!-- Purpose: Test serialization with item separator set to whitespace (three spaces) -->

   <xsl:template name="xsl:initial-template">
      <xsl:sequence select="11 to 20"/>
   </xsl:template>
</xsl:transform>
