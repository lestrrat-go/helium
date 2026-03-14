<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  xmlns:xs="http://www.w3.org/2001/XMLSchema" version="3.0">

   <!-- Static param without select attr, but with an as-clause, implicitly mandatory-->
   
   <xsl:param name="static-param" static="yes" as="xs:integer" select="xs:date('2014-03-03')" />

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {999 * $static-param}
   </xsl:template>
   
</xsl:transform>
