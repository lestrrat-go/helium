<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <!-- Static param with tunner attr set, error -->
   
   <xsl:param name="static-param" static="yes" tunnel="yes" />

   <xsl:template name="xsl:initial-template">
      We never get here!!!
   </xsl:template>
   
</xsl:transform>
