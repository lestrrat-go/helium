<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <!-- Static param with private visibility, previously allowed but changed: attribute must not appear on xsl:param -->
   
   <xsl:param name="static-param" static="yes" select="'visibility not allowed'" visibility="private" />

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {$static-param}
   </xsl:template>
   
</xsl:transform>
