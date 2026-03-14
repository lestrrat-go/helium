<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="my" exclude-result-prefixes="xs my"
  version="3.0">



  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:apply-templates select="1 to 5"/>
    </out>
  </xsl:template>

  <xsl:template match=".[. ge 5]" priority="5">
    <xsl:text>A</xsl:text>
    <xsl:apply-imports/>
  </xsl:template>

  <xsl:template match=".[. ge 3]" priority="3">
    <xsl:text>B</xsl:text>
    <xsl:apply-imports/>
  </xsl:template>
  
  <xsl:import href="apply-imports-001a.xsl"/>
  <xsl:import href="apply-imports-001b.xsl"/>

</xsl:stylesheet>
